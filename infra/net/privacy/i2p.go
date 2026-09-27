package privacy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"go.uber.org/zap"
)

// i2pService exposes the relay over the I2P network.
//
// The supported, production-default mode is "external": a minimal SAM v3 client
// (see sam.go) connects to an already-running I2P router (i2pd or Java-I2P) on
// the SAM API port and publishes the session's .b32.i2p base-address. This
// interoperates with stock daemons and avoids depending on go-i2p's unstable
// embedded-router streaming API.
//
// Native mode starts a managed go-i2p router and a local SAM bridge. It never
// falls back to an external daemon; external mode remains the interoperable
// choice for Java I2P and i2pd installations.
type i2pService struct {
	cfg    config.I2PConfig
	logger *zap.Logger
	store  *KeyStore

	mu      sync.Mutex
	started bool
	sam     *samClient
	native  *nativeI2P
	address string
	forward bool
	target  string

	// observability (see Status)
	startedAt     time.Time
	startErr      string
	txBytes       int64
	rxBytes       int64
	connections   int
	startFailures uint64
}

func newI2PService(cfg config.I2PConfig, logger *zap.Logger, store *KeyStore) Service {
	return &i2pService{cfg: cfg, logger: logger, store: store}
}

func (s *i2pService) Name() string { return "i2p" }

func (s *i2pService) Start(ctx context.Context, relayPort int) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	// Record the observability state regardless of the return path.
	defer func() {
		if err != nil {
			s.startErr = err.Error()
			s.startFailures++
			s.startedAt = time.Time{}
			s.started = false
		} else {
			s.startErr = ""
			s.startedAt = time.Now()
		}
	}()

	mode := resolveMode(s.cfg.Mode, false) // production default = external
	switch mode {
	case "native":
		native, startErr := s.startNative(ctx, relayPort)
		if startErr != nil {
			return startErr
		}
		s.native = native
		s.started = true
		if err := s.saveExternalManifest(); err != nil {
			s.logger.Warn("could not persist active I2P metadata", zap.Error(err))
		}
		s.logger.Info("i2p started (native)", zap.String("b32", s.address), zap.String("target", s.target))
		return nil
	case "external", "disabled":
		// fall through
	case "auto":
		// auto prefers external for I2P (the stable path).
		mode = "external"
	default:
		return fmt.Errorf("i2p: unknown mode %q", s.cfg.Mode)
	}
	if mode == "disabled" {
		return nil
	}

	host := s.cfg.SAMHost
	if host == "" {
		host = "127.0.0.1"
	}
	port := s.cfg.SAMPort
	if port == 0 {
		port = 7656
	}

	// Persistent identity: reuse the same SAM destination blob across restarts
	// so the .b32.i2p address stays stable.
	var persistDest string
	if s.store != nil {
		var err error
		persistDest, err = loadOrCreateString(s.store, "i2p.key")
		if err != nil {
			return fmt.Errorf("i2p external: %w", err)
		}
	}

	client := newSAMClient(host, port, s.cfg.SessionName, persistDest)
	if err := client.connect(10 * time.Second); err != nil {
		return fmt.Errorf("i2p external: %w", err)
	}
	addr := client.B32Address()
	if addr == "" {
		_ = client.Close()
		return fmt.Errorf("i2p external: could not derive .b32.i2p address")
	}
	target := net.JoinHostPort("127.0.0.1", strconv.Itoa(relayPort))
	if err := client.startForward("127.0.0.1", relayPort, 10*time.Second); err != nil {
		_ = client.Close()
		return fmt.Errorf("i2p external: start inbound stream forward to %s: %w", target, err)
	}

	// First run: persist the router-generated destination blob for reuse.
	if s.store != nil && persistDest == "" && client.Destination() != "" {
		if err := s.store.Save("i2p.key", []byte(client.Destination())); err != nil {
			s.logger.Warn("i2p: failed to persist destination", zap.Error(err))
		}
	}

	s.sam = client
	s.address = addr
	s.forward = true
	s.target = target
	s.started = true
	if err := s.saveExternalManifest(); err != nil {
		s.logger.Warn("could not persist active I2P metadata", zap.Error(err))
	}
	s.logger.Info("i2p started (external SAM)",
		zap.String("b32", addr), zap.String("sam", net.JoinHostPort(host, strconv.Itoa(port))), zap.String("target", target))
	return nil
}

func (s *i2pService) Addresses() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.address != "" {
		return []string{s.address}
	}
	return nil
}

func (s *i2pService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sam != nil {
		_ = s.sam.Close()
		s.sam = nil
	}
	if s.native != nil {
		_ = s.native.Close()
		s.native = nil
	}
	s.address = ""
	s.forward = false
	s.target = ""
	s.started = false
	return nil
}

// AuthURLs returns the address that is currently backed by the active SAM
// forward. A persisted manifest alone never grants NIP-42 authorization.
func (s *i2pService) AuthURLs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started || !s.forward || s.address == "" {
		return nil
	}
	return []string{"ws://" + s.address}
}

type i2pManifest struct {
	SchemaVersion  int       `json:"schema_version"`
	Mode           string    `json:"mode"`
	Implementation string    `json:"implementation"`
	Address        string    `json:"address"`
	RelayURLs      []string  `json:"relay_urls"`
	Target         string    `json:"target"`
	Forward        string    `json:"forward"`
	SAMAddress     string    `json:"sam_address"`
	I2CPAddress    string    `json:"i2cp_address,omitempty"`
	PublishedAt    time.Time `json:"published_at"`
}

func (s *i2pService) saveExternalManifest() error {
	if s.store == nil {
		return nil
	}
	host := s.cfg.SAMHost
	if host == "" {
		host = "127.0.0.1"
	}
	samPort := s.cfg.SAMPort
	if samPort == 0 {
		samPort = 7656
	}
	mode := resolveMode(s.cfg.Mode, false)
	manifest := i2pManifest{
		SchemaVersion:  1,
		Mode:           mode,
		Implementation: "sam-v3",
		Address:        s.address,
		RelayURLs:      []string{"ws://" + s.address},
		Target:         s.target,
		Forward:        "STREAM FORWARD SILENT=true",
		SAMAddress:     net.JoinHostPort(host, strconv.Itoa(samPort)),
		PublishedAt:    time.Now().UTC(),
	}
	if mode == "native" {
		manifest.Implementation = "go-i2p/go-sam-bridge"
		i2cpPort := s.cfg.I2CPPort
		if i2cpPort == 0 {
			i2cpPort = 7654
		}
		manifest.I2CPAddress = net.JoinHostPort(host, strconv.Itoa(i2cpPort))
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal I2P manifest: %w", err)
	}
	if err := s.store.Save("i2p.json", data); err != nil {
		return fmt.Errorf("save I2P manifest: %w", err)
	}
	return nil
}

// Status returns a copy of the I2P network's observability snapshot.
func (s *i2pService) Status() StatusSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	var addresses []string
	if s.address != "" {
		addresses = []string{s.address}
	}
	return StatusSnapshot{
		ID:            "i2p",
		Mode:          resolveMode(s.cfg.Mode, false),
		Enabled:       s.cfg.Mode != "" && s.cfg.Mode != "disabled",
		Started:       s.started,
		StartErr:      s.startErr,
		Addresses:     addresses,
		Uptime:        uptimeSince(s.startedAt, s.started),
		TxBytes:       s.txBytes,
		RxBytes:       s.rxBytes,
		Connections:   s.connections,
		Peers:         nil, // an I2P eepsite has no peer count
		StartFailures: s.startFailures,
	}
}
