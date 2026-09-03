package privacy

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/voluminor/ratatoskr"
	"github.com/voluminor/ratatoskr/mod/forward"
	yggconfig "github.com/yggdrasil-network/yggdrasil-go/src/config"
	"go.uber.org/zap"
)

// yggService embeds a Yggdrasil mesh node inside the relay process (ratatoskr
// wraps yggdrasil-go + a gVisor userspace stack) and forwards the relay's local
// listener out onto the Yggdrasil network, so the relay becomes reachable at its
// Yggdrasil IPv6 address.
//
// Unlike Tor/I2P there is no separate "external daemon + proxy" split for
// Yggdrasil: the mesh node runs in-process (native). "external" is accepted and
// mapped onto the same embedded node for config symmetry.
type yggService struct {
	cfg    config.YggConfig
	logger *zap.Logger
	store  *KeyStore

	mu      sync.Mutex
	started bool
	node    *ratatoskr.Obj
	fwd     *forward.Obj
	addr    *net.TCPAddr

	// observability (see Status)
	startedAt   time.Time
	startErr    string
	txBytes     int64
	rxBytes     int64
	connections int
}

func newYggService(cfg config.YggConfig, logger *zap.Logger, store *KeyStore) Service {
	return &yggService{cfg: cfg, logger: logger, store: store}
}

func (s *yggService) Name() string { return "yggdrasil" }

func (s *yggService) Start(ctx context.Context, relayPort int) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	// Record the observability state regardless of the return path.
	defer func() {
		if err != nil {
			s.startErr = err.Error()
			s.startedAt = time.Time{}
			s.started = false
		} else {
			s.startErr = ""
			s.startedAt = time.Now()
		}
	}()

	// Yggdrasil is always embedded; "disabled" is the only disabling mode.
	mode := resolveMode(s.cfg.Mode, true)
	if mode == "disabled" {
		return nil
	}

	// Build a yggdrasil NodeConfig. We generate a default config and, when a
	// persistent store is available, reuse a stable identity so the node keeps
	// the same Yggdrasil IPv6 address across restarts.
	nc := yggconfig.GenerateConfig()
	nc.PrivateKey = nil
	if s.store != nil {
		keyBytes, kerr := s.store.LoadOrCreate("ygg.key", func() ([]byte, error) {
			gen := yggconfig.GenerateConfig()
			gen.NewPrivateKey()
			return []byte(gen.PrivateKey), nil
		})
		if kerr != nil {
			return fmt.Errorf("persistent yggdrasil identity: %w", kerr)
		}
		nc.PrivateKey = yggconfig.KeyBytes(keyBytes)
	}
	if len(s.cfg.Peers) > 0 {
		nc.Peers = s.cfg.Peers
	}

	rcfg := ratatoskr.ConfigObj{
		Ctx:    ctx,
		Logger: nil,
		Config: nc,
	}

	node, err := ratatoskr.New(rcfg)
	if err != nil {
		return fmt.Errorf("yggdrasil: %w", err)
	}
	s.node = node

	if len(s.cfg.Peers) > 0 {
		s.logger.Info("yggdrasil: static peers wired via NodeConfig",
			zap.Int("peers", len(s.cfg.Peers)))
	}

	listenPort := s.cfg.ListenPort
	if listenPort == 0 {
		listenPort = relayPort
	}

	// Expose the relay on the Yggdrasil network: map the node's own IPv6
	// address:<listenPort> to the local relay listener 127.0.0.1:<relayPort>.
	fwd, err := forward.New(forward.ConfigObj{
		Node: node, // *ratatoskr.Obj satisfies forward.NetworkInterface
		RemoteTCP: []forward.TCPMappingObj{{
			Listen: &net.TCPAddr{IP: node.Address(), Port: listenPort},
			Mapped: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: relayPort},
		}},
	})
	if err != nil {
		_ = node.Close()
		return fmt.Errorf("yggdrasil forward: %w", err)
	}
	s.fwd = fwd
	s.addr = &net.TCPAddr{IP: node.Address(), Port: listenPort}
	s.started = true
	s.logger.Info("yggdrasil started", zap.String("address", s.addr.String()))
	return nil
}

func (s *yggService) Addresses() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.addr != nil {
		return []string{s.addr.String()}
	}
	return nil
}

func (s *yggService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fwd != nil {
		_ = s.fwd.Close()
		s.fwd = nil
	}
	if s.node != nil {
		_ = s.node.Close()
		s.node = nil
	}
	s.addr = nil
	s.started = false
	return nil
}

// dialPort is a small helper to format an address for logging.
func dialPort(p int) string { return strconv.Itoa(p) }

// Status returns a copy of the Yggdrasil network's observability snapshot.
func (s *yggService) Status() StatusSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	var peers *int
	if len(s.cfg.Peers) > 0 {
		n := len(s.cfg.Peers)
		peers = &n
	}
	return StatusSnapshot{
		ID:          "yggdrasil",
		Mode:        resolveMode(s.cfg.Mode, true),
		Enabled:     s.cfg.Mode != "" && s.cfg.Mode != "disabled",
		Started:     s.started,
		StartErr:    s.startErr,
		Addresses:   s.Addresses(),
		Uptime:      uptimeSince(s.startedAt, s.started),
		TxBytes:     s.txBytes,
		RxBytes:     s.rxBytes,
		Connections: s.connections,
		Peers:       peers,
	}
}
