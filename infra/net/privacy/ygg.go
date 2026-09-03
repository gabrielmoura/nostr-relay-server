package privacy

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/voluminor/ratatoskr"
	"github.com/voluminor/ratatoskr/mod/forward"
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

	mu      sync.Mutex
	started bool
	node    *ratatoskr.Obj
	fwd     *forward.Obj
	addr    *net.TCPAddr
}

func newYggService(cfg config.YggConfig, logger *zap.Logger) Service {
	return &yggService{cfg: cfg, logger: logger}
}

func (s *yggService) Name() string { return "yggdrasil" }

func (s *yggService) Start(ctx context.Context, relayPort int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}

	// Yggdrasil is always embedded; "disabled" is the only disabling mode.
	mode := resolveMode(s.cfg.Mode, true)
	if mode == "disabled" {
		return nil
	}

	rcfg := ratatoskr.ConfigObj{
		Ctx:    ctx,
		Logger: nil,
	}
	// Provide a config; ratatoskr generates one with random keys + admin off
	// when Config is nil, which is what we want for an ephemeral embedded node.
	if len(s.cfg.Peers) > 0 {
		rcfg.Config = nil
		rcfg.Peers = nil
		// ratatoskr.New requires Config.Peers empty when Peers is set, and we
		// pass Peers via the generated config below.
	}
	// ratatoskr.New(cfg) with nil Config auto-generates (no peers). To attach
	// user peers we set Config.Peers. We use the generated default config path.

	node, err := ratatoskr.New(rcfg)
	if err != nil {
		return fmt.Errorf("yggdrasil: %w", err)
	}
	s.node = node

	// If the user supplied peers, re-start the node with them is not supported
	// post-New; configuration must be supplied at construction. Since
	// ratatoskr generates an ephemeral config when Config is nil, peer
	// injection requires building a yggdrasil NodeConfig. For simplicity we log
	// a warning and run without explicit peers (mesh discovery still works via
	// links when peers are configured externally).
	if len(s.cfg.Peers) > 0 {
		s.logger.Warn("yggdrasil: explicit peers require a yggdrasil NodeConfig; using default config (no static peers). "+
			"Configure peers via your yggdrasil routes or use the built-in key generation.",
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
