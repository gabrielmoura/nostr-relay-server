package privacy

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"errors"
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

const (
	torGracefulShutdownTimeout = 5 * time.Second
	torForcedShutdownTimeout   = 5 * time.Second
)

// torService exposes the relay on a Tor onion address and provides outbound
// connectivity through Tor's SOCKS proxy.
//
// Modes:
//   - native: bine starts and manages a local `tor` process (from PATH or
//     ExePath) and creates the onion service. Requires a `tor` binary.
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
	startedAt     time.Time
	startErr      string
	txBytes       int64
	rxBytes       int64
	connections   int
	startFailures uint64
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
		s.startFailures++
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
	if !s.cfg.UseV3 {
		return errors.New("only Tor v3 onion services are supported; set privacy.tor.v3 to true")
	}
	if err := prepareTorDataDir(s.cfg.DataDir, s.logger); err != nil {
		return err
	}

	conf := &tor.StartConf{EnableNetwork: true}
	if s.cfg.DataDir != "" {
		conf.DataDir = s.cfg.DataDir
	}
	if s.cfg.ControlPort != 0 {
		conf.ControlPort = s.cfg.ControlPort
	}

	t, err := tor.Start(ctx, conf)
	if err != nil {
		if t != nil && t.Process != nil {
			s.logger.Debug("recovering Tor process after failed startup")
			if closeErr := stopTorProcess(t, s.logger); closeErr != nil {
				return errors.Join(err, fmt.Errorf("recover failed Tor process: %w", closeErr))
			}
		}
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
		Version3:    true,
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
	var closeErrs []error
	if s.onion != nil {
		s.logger.Debug("closing Tor onion service")
		if err := s.onion.Close(); err != nil {
			closeErrs = append(closeErrs, fmt.Errorf("close onion service: %w", err))
		} else {
			s.onion = nil
		}
	}
	if s.proc != nil {
		if err := stopTorProcess(s.proc, s.logger); err != nil {
			closeErrs = append(closeErrs, err)
		} else {
			s.proc = nil
		}
	}
	if s.proc == nil {
		s.started = false
		s.startedAt = time.Time{}
		s.onionID = ""
	}
	if len(closeErrs) > 0 {
		return errors.Join(closeErrs...)
	}
	s.logger.Info("Tor network stopped")
	return nil
}

func stopTorProcess(proc *tor.Tor, logger *zap.Logger) error {
	startedAt := time.Now()
	if proc.Control != nil {
		if proc.Control.Authenticated && proc.StopProcessOnClose {
			logger.Debug("sending HALT to Tor process")
			if err := proc.Control.Signal("HALT"); err != nil {
				logger.Warn("failed to send HALT to Tor process", zap.Error(err))
			} else {
				logger.Debug("HALT sent to Tor process")
			}
		}
		if err := proc.Control.Close(); err != nil {
			logger.Warn("failed to close Tor control connection", zap.Error(err))
		}
		proc.Control = nil
	}
	if proc.Process == nil {
		return nil
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- proc.Process.Wait() }()
	select {
	case err := <-waitCh:
		proc.Process = nil
		if err != nil {
			return fmt.Errorf("Tor process exited after HALT: %w", err)
		}
		logger.Info("Tor process stopped", zap.Duration("duration", time.Since(startedAt)))
		return nil
	case <-time.After(torGracefulShutdownTimeout):
		logger.Warn("Tor process did not stop after HALT; forcing termination",
			zap.Duration("timeout", torGracefulShutdownTimeout))
	}

	if proc.ProcessCancelFunc == nil {
		return fmt.Errorf("Tor process did not stop after HALT and has no cancellation function")
	}
	logger.Debug("forcing Tor process termination")
	proc.ProcessCancelFunc()
	select {
	case err := <-waitCh:
		proc.Process = nil
		if err != nil {
			logger.Warn("Tor process was forcefully terminated", zap.Duration("duration", time.Since(startedAt)), zap.Error(err))
			return nil
		}
		logger.Warn("Tor process stopped after forced termination", zap.Duration("duration", time.Since(startedAt)))
		return nil
	case <-time.After(torForcedShutdownTimeout):
		return fmt.Errorf("Tor process did not exit after forced termination within %s", torForcedShutdownTimeout)
	}
}

// Status returns a copy of the Tor network's observability snapshot.
func (s *torService) Status() StatusSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	var addresses []string
	if s.onionID != "" {
		addresses = []string{s.onionID + ".onion"}
	}
	return StatusSnapshot{
		ID:            "tor",
		Mode:          resolveMode(s.cfg.Mode, true),
		Enabled:       s.cfg.Mode != "" && s.cfg.Mode != "disabled",
		Started:       s.started,
		StartErr:      s.startErr,
		Addresses:     addresses,
		Uptime:        uptimeSince(s.startedAt, s.started),
		TxBytes:       s.txBytes,
		RxBytes:       s.rxBytes,
		Connections:   s.connections,
		Peers:         nil, // bine does not expose a uniform circuit/peer counter
		StartFailures: s.startFailures,
	}
}
