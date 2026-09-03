package privacy

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/cretz/bine/tor"
	bined25519 "github.com/cretz/bine/torutil/ed25519"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"
)

// torService exposes the relay on a Tor onion address and provides outbound
// connectivity through Tor's SOCKS proxy.
//
// Modes:
//   - native: bine spawns/attaches a `tor` process (from PATH or ExePath) and
//     creates the onion service in-process. Requires a `tor` binary available.
//   - external: an already-running Tor daemon (e.g. via torrc / Docker) provides
//     the onion address; we reuse its SOCKS proxy for outbound and expose the
//     configured onion URL via relay_information.
type torService struct {
	cfg    config.TorConfig
	logger *zap.Logger
	store  *KeyStore

	mu      sync.Mutex
	started bool
	onion   *tor.OnionService
	proc    *tor.Tor
	socks   string
	onionID string

	// observability (see Status)
	startedAt   time.Time
	startErr    string
	txBytes     int64
	rxBytes     int64
	connections int
}

func newTorService(cfg config.TorConfig, logger *zap.Logger, store *KeyStore) Service {
	return &torService{cfg: cfg, logger: logger, store: store}
}

func (s *torService) Name() string { return "tor" }

func (s *torService) Start(ctx context.Context, relayPort int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}

	mode := resolveMode(s.cfg.Mode, true)
	s.socks = net.JoinHostPort("127.0.0.1", strconv.Itoa(s.cfg.SocksPort))

	var startErr error
	switch mode {
	case "native":
		if err := s.startNative(ctx, relayPort); err != nil {
			startErr = fmt.Errorf("tor native: %w", err)
		}
	case "external":
		// The external daemon's inbound onion is configured out-of-band
		// (torrc / Docker). We only expose outbound SOCKS capability and log
		// the configured proxy.
		s.logger.Info("tor external: using existing daemon", zap.String("socks", s.socks))
	default:
		startErr = fmt.Errorf("tor: unknown mode %q", s.cfg.Mode)
	}
	if startErr != nil {
		s.startErr = startErr.Error()
		return startErr
	}
	s.started = true
	s.startedAt = time.Now()
	s.startErr = ""
	return nil
}

// startNative spawns a tor process via bine and publishes an onion service that
// forwards the onion ports to 127.0.0.1:relayPort (the relay's own listener),
// keeping the relay reachable on the onion address.
func (s *torService) startNative(ctx context.Context, relayPort int) error {
	conf := &tor.StartConf{EnableNetwork: true}
	if s.cfg.DataDir != "" {
		conf.DataDir = s.cfg.DataDir
	}
	if s.cfg.ControlPort != 0 {
		conf.ControlPort = s.cfg.ControlPort
	}

	t, err := tor.Start(ctx, conf)
	if err != nil {
		return err
	}
	s.proc = t

	localPort := s.cfg.OnionPort
	if localPort == 0 {
		localPort = relayPort
	}
	remotePorts := s.cfg.RemotePorts
	if len(remotePorts) == 0 {
		remotePorts = []int{80}
	}
	v3 := s.cfg.UseV3
	if !s.cfg.UseV3 {
		v3 = true // default to v3
	}

	// Persistent identity: reuse the same v3 ed25519 key across restarts so the
	// .onion address stays stable. Load-or-create a 64-byte ed25519 private key.
	var key crypto.PrivateKey
	if s.store != nil {
		keyBytes, err := s.store.LoadOrCreate("tor.key", func() ([]byte, error) {
			_, priv, kerr := ed25519.GenerateKey(nil)
			if kerr != nil {
				return nil, fmt.Errorf("generating onion key: %w", kerr)
			}
			return []byte(priv), nil
		})
		if err != nil {
			_ = t.Close()
			return fmt.Errorf("persistent onion key: %w", err)
		}
		key = bined25519.FromCryptoPrivateKey(ed25519.PrivateKey(keyBytes))
	}

	onion, err := t.Listen(ctx, &tor.ListenConf{
		LocalPort:   localPort, // bine dials 127.0.0.1:<localPort> -> the relay's own port
		RemotePorts: remotePorts,
		Version3:    v3,
		Key:         key,
	})
	if err != nil {
		_ = t.Close()
		return err
	}
	s.onion = onion
	s.onionID = onion.ID
	return nil
}

func (s *torService) Addresses() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.onionID != "" {
		return []string{s.onionID + ".onion"}
	}
	return nil
}

// DialContext returns an outbound dialer through Tor's SOCKS proxy. Used when
// the relay needs to connect to remote .onion services (e.g. relay pooling).
func (s *torService) DialContext() (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	d, err := proxy.SOCKS5("tcp", s.socks, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}
	return func(_ context.Context, network, addr string) (net.Conn, error) {
		return d.Dial(network, addr)
	}, nil
}

func (s *torService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.onion != nil {
		_ = s.onion.Close()
		s.onion = nil
	}
	if s.proc != nil {
		_ = s.proc.Close()
		s.proc = nil
	}
	s.started = false
	return nil
}

// Status returns a copy of the Tor network's observability snapshot.
func (s *torService) Status() StatusSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return StatusSnapshot{
		ID:          "tor",
		Mode:        resolveMode(s.cfg.Mode, true),
		Enabled:     s.cfg.Mode != "" && s.cfg.Mode != "disabled",
		Started:     s.started,
		StartErr:    s.startErr,
		Addresses:   s.Addresses(),
		Uptime:      uptimeSince(s.startedAt, s.started),
		TxBytes:     s.txBytes,
		RxBytes:     s.rxBytes,
		Connections: s.connections,
		Peers:       nil, // bine does not expose a uniform circuit/peer counter
	}
}
