package privacy

import (
	"context"
	"fmt"
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
// "native" (embedding go-i2p's router) is EXPERIMENTAL. A fully working eepsite
// requires wiring go-i2p's embedded router together with its SAM/I2CP server,
// which is not yet integrated here; selecting native surfaces a clear warning
// and instructs the operator to use the external SAM path instead.
type i2pService struct {
	cfg    config.I2PConfig
	logger *zap.Logger

	mu      sync.Mutex
	started bool
	sam     *samClient
	address string
}

func newI2PService(cfg config.I2PConfig, logger *zap.Logger) Service {
	return &i2pService{cfg: cfg, logger: logger}
}

func (s *i2pService) Name() string { return "i2p" }

func (s *i2pService) Start(ctx context.Context, relayPort int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}

	mode := resolveMode(s.cfg.Mode, false) // production default = external
	switch mode {
	case "native":
		// EXPERIMENTAL: embedded go-i2p eepsite is not yet wired. Do not fail
		// silently — log clearly and fall through to the SAM daemon path if the
		// operator has one, otherwise report the experimental state.
		s.logger.Warn("i2p native is EXPERIMENTAL and not fully wired for eepsite "+
			"serving yet; falling back to external SAM (requires an i2pd/Java-I2P "+
			"router on port 7656). Set i2p.mode=external to silence this warning.",
			zap.String("sam_host", s.cfg.SAMHost), zap.Int("sam_port", s.cfg.SAMPort))
		mode = "external"
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

	client := newSAMClient(host, port, s.cfg.SessionName)
	if err := client.connect(10 * time.Second); err != nil {
		return fmt.Errorf("i2p external: %w", err)
	}
	addr := client.B32Address()
	if addr == "" {
		_ = client.Close()
		return fmt.Errorf("i2p external: could not derive .b32.i2p address")
	}

	s.sam = client
	s.address = addr
	s.started = true
	s.logger.Info("i2p started (external SAM)",
		zap.String("b32", addr), zap.String("sam", host+":???"))
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
	s.address = ""
	s.started = false
	return nil
}
