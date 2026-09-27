package privacy

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/cretz/bine/control"
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

// startNative spawns a Tor process via bine and publishes an onion service with
// ADD_ONION. Tor forwards directly to Fiber's already-owned relay TCP endpoint;
// Bine never receives or closes the Fiber listener.
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

	remotePorts := s.cfg.RemotePorts
	if len(remotePorts) == 0 {
		remotePorts = []int{80}
	}
	// Persistent identity: reuse the same v3 ed25519 key across restarts so the
	// .onion address stays stable. Load-or-create a 64-byte ed25519 private key.
	var key control.Key = control.GenKey(control.KeyAlgoED25519V3)
	if s.store != nil {
		keyBytes, err := s.store.LoadOrCreate("tor.key", func() ([]byte, error) {
			_, priv, kerr := ed25519.GenerateKey(nil)
			if kerr != nil {
				return nil, fmt.Errorf("generating onion key: %w", kerr)
			}
			return []byte(priv), nil
		})
		if err != nil {
			return s.cleanupFailedNativeStart(fmt.Errorf("persistent onion key: %w", err))
		}
		key = &control.ED25519Key{KeyPair: bined25519.FromCryptoPrivateKey(ed25519.PrivateKey(keyBytes))}
	}

	localPort := s.cfg.OnionPort
	if localPort == 0 {
		localPort = relayPort
	}
	onion, err := t.Control.AddOnion(newTorAddOnionRequest(remotePorts, net.JoinHostPort("127.0.0.1", strconv.Itoa(localPort)), key))
	if err != nil {
		return s.cleanupFailedNativeStart(err)
	}
	s.onionID = onion.ServiceID
	if err := waitForTorOnionPublication(ctx, t, s.onionID); err != nil {
		return s.cleanupFailedNativeStart(fmt.Errorf("wait for onion service publication: %w", err))
	}
	if err := s.saveNativeManifest(remotePorts, localPort); err != nil {
		s.logger.Warn("could not persist active Tor onion metadata", zap.Error(err))
	}
	return nil
}

type torManifest struct {
	SchemaVersion       int       `json:"schema_version"`
	OnionAddress        string    `json:"onion_address"`
	OnionServiceVersion int       `json:"onion_service_version"`
	RelayURLs           []string  `json:"relay_urls"`
	RemotePorts         []int     `json:"remote_ports"`
	Target              string    `json:"target"`
	PublishedAt         time.Time `json:"published_at"`
}

func (s *torService) saveNativeManifest(remotePorts []int, localPort int) error {
	if s.store == nil {
		return nil
	}

	manifest := torManifest{
		SchemaVersion:       1,
		OnionAddress:        s.onionID + ".onion",
		OnionServiceVersion: 3,
		RelayURLs:           torAuthURLs(s.onionID, remotePorts),
		RemotePorts:         append([]int(nil), remotePorts...),
		Target:              net.JoinHostPort("127.0.0.1", strconv.Itoa(localPort)),
		PublishedAt:         time.Now().UTC(),
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal Tor manifest: %w", err)
	}
	if err := s.store.Save("tor.json", data); err != nil {
		return fmt.Errorf("save Tor manifest: %w", err)
	}
	return nil
}

func newTorAddOnionRequest(remotePorts []int, target string, key control.Key) *control.AddOnionRequest {
	ports := make([]*control.KeyVal, 0, len(remotePorts))
	for _, remotePort := range remotePorts {
		ports = append(ports, &control.KeyVal{Key: strconv.Itoa(remotePort), Val: target})
	}
	return &control.AddOnionRequest{Key: key, Ports: ports}
}

// waitForTorOnionPublication preserves the readiness contract of tor.Listen:
// successful ADD_ONION only means Tor accepted the configuration, not that a
// descriptor has reached a hidden-service directory.
func waitForTorOnionPublication(ctx context.Context, proc *tor.Tor, onionID string) error {
	if proc == nil || proc.Control == nil {
		return errors.New("Tor control connection is unavailable")
	}
	if err := proc.EnableNetwork(ctx, true); err != nil {
		return fmt.Errorf("enable Tor network: %w", err)
	}

	uploadsAttempted := 0
	failures := make([]string, 0)
	_, err := proc.Control.EventWait(ctx, []control.EventCode{control.EventCodeHSDesc}, func(evt control.Event) (bool, error) {
		hs, _ := evt.(*control.HSDescEvent)
		if hs == nil || hs.Address != onionID {
			return false, nil
		}

		switch hs.Action {
		case "UPLOAD":
			uploadsAttempted++
		case "FAILED":
			failures = append(failures, fmt.Sprintf("directory %s: %s", hs.HSDir, hs.Reason))
			if uploadsAttempted > 0 && len(failures) == uploadsAttempted {
				return false, fmt.Errorf("all onion descriptor uploads failed: %v", failures)
			}
		case "UPLOADED":
			return true, nil
		}
		return false, nil
	})
	return err
}

// cleanupFailedNativeStart rolls back the process that was acquired before a
// later native-Tor setup step failed. It retains the process reference when
// termination cannot be confirmed so a subsequent Close can still report it.
func (s *torService) cleanupFailedNativeStart(startErr error) error {
	if s.proc == nil {
		return startErr
	}

	var cleanupErrs []error
	if s.onionID != "" && s.proc.Control != nil {
		if err := s.proc.Control.DelOnion(s.onionID); err != nil {
			cleanupErrs = append(cleanupErrs, fmt.Errorf("delete onion service after failed startup: %w", err))
		} else {
			s.onionID = ""
		}
	}
	if err := stopTorProcess(s.proc, s.logger); err != nil {
		cleanupErrs = append(cleanupErrs, fmt.Errorf("stop Tor after failed startup: %w", err))
	} else {
		s.proc = nil
	}
	if len(cleanupErrs) == 0 {
		return startErr
	}
	return errors.Join(append([]error{startErr}, cleanupErrs...)...)
}

func (s *torService) Addresses() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.onionID != "" {
		return []string{s.onionID + ".onion"}
	}
	return nil
}

// AuthURLs returns only URLs derived from the currently published native onion
// service. It intentionally does not use persisted tor.json metadata.
func (s *torService) AuthURLs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started || resolveMode(s.cfg.Mode, true) != "native" || s.onionID == "" {
		return nil
	}
	return torAuthURLs(s.onionID, s.cfg.RemotePorts)
}

func torAuthURLs(onionID string, remotePorts []int) []string {
	if onionID == "" {
		return nil
	}
	if len(remotePorts) == 0 {
		remotePorts = []int{80}
	}

	host := onionID + ".onion"
	urls := make([]string, 0, len(remotePorts))
	for _, remotePort := range remotePorts {
		if remotePort == 80 {
			urls = append(urls, "ws://"+host)
			continue
		}
		urls = append(urls, "ws://"+net.JoinHostPort(host, strconv.Itoa(remotePort)))
	}
	return urls
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
	if s.onionID != "" && s.proc != nil && s.proc.Control != nil {
		s.logger.Debug("closing Tor onion service", zap.String("onion_id", s.onionID))
		if err := s.proc.Control.DelOnion(s.onionID); err != nil {
			closeErrs = append(closeErrs, fmt.Errorf("delete onion service: %w", err))
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
