// Package privacy exposes the Nostr relay over the Tor (onion), I2P (eepsite)
// and Yggdrasil (mesh) privacy networks, either by running each network
// in-process ("native") or by connecting to an already-running daemon
// ("external").
//
// Everything is opt-in via the `privacy:` block in conf.yaml. Native modes are
// pure Go (no external binaries, no root). I2P native mode is EXPERIMENTAL:
// the embedded router is early-stage software, so it is gated behind an
// explicit `privacy.i2p.mode: native` and logs a warning. The production
// default for I2P is `external` via the standard SAM API (port 7656), which
// interoperates with stock i2pd / Java-I2P routers.
package privacy

import (
	"context"
	"fmt"
	"sync"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"go.uber.org/zap"
)

// active registry: the set of privacy addresses (onion/b32/ygg) currently
// advertised, used by the NIP-11 handler to publish them without coupling the
// config package to the privacy implementation.
var (
	activeMu    sync.RWMutex
	activeAddrs []string
)

// GetActiveAddresses returns the currently advertised privacy addresses.
func GetActiveAddresses() []string {
	activeMu.RLock()
	defer activeMu.RUnlock()
	out := make([]string, len(activeAddrs))
	copy(out, activeAddrs)
	return out
}

// setActiveAddresses replaces the registry contents.
func setActiveAddresses(addrs []string) {
	activeMu.Lock()
	defer activeMu.Unlock()
	activeAddrs = append([]string(nil), addrs...)
}

// Service is a single privacy network instance. Start brings up the network and
// arranges for the relay's local TCP listener to be reachable on it.
type Service interface {
	// Start initialises the network and forwards the local relay port into it.
	// relayPort is the host TCP port the public relay listens on.
	Start(ctx context.Context, relayPort int) error
	// Addresses returns the advertised addresses of this service (e.g.
	// "xyz.onion", "abc.i2p", "200:db8::1/18") — empty if not started.
	Addresses() []string
	// Close tears the network down. Idempotent.
	Close() error
	// Name returns the network name for logging ("tor", "i2p", "yggdrasil").
	Name() string
}

// Manager owns the enabled privacy networks and their lifecycle.
type Manager struct {
	cfg      config.PrivacyConfig
	logger   *zap.Logger
	services []Service
	mu       sync.Mutex
}

// NewManager builds a Manager from the privacy config block.
func NewManager(cfg config.PrivacyConfig, logger *zap.Logger) *Manager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Manager{cfg: cfg, logger: logger}
}

// Start constructs and starts every enabled network service.
func (m *Manager) Start(ctx context.Context, relayPort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.cfg.Enabled {
		return nil
	}

	if err := m.buildServices(); err != nil {
		return err
	}

	for _, svc := range m.services {
		if err := svc.Start(ctx, relayPort); err != nil {
			m.logger.Warn("privacy network failed to start",
				zap.String("network", svc.Name()), zap.Error(err))
			continue
		}
		m.logger.Info("privacy network started",
			zap.String("network", svc.Name()),
			zap.Strings("addresses", svc.Addresses()))
	}
	setActiveAddresses(m.Addresses())
	return nil
}

// buildServices instantiates a Service for each configured (non-disabled) network.
func (m *Manager) buildServices() error {
	var svcs []Service
	if m.cfg.Tor.Mode != "" && m.cfg.Tor.Mode != "disabled" {
		svcs = append(svcs, newTorService(m.cfg.Tor, m.logger.Named("tor")))
	}
	if m.cfg.I2P.Mode != "" && m.cfg.I2P.Mode != "disabled" {
		svcs = append(svcs, newI2PService(m.cfg.I2P, m.logger.Named("i2p")))
	}
	if m.cfg.Ygg.Mode != "" && m.cfg.Ygg.Mode != "disabled" {
		svcs = append(svcs, newYggService(m.cfg.Ygg, m.logger.Named("yggdrasil")))
	}
	if len(svcs) == 0 {
		return fmt.Errorf("privacy.enabled is true but no network has a mode other than disabled")
	}
	m.services = svcs
	return nil
}

// Addresses returns the union of advertised addresses across all started services.
func (m *Manager) Addresses() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for _, svc := range m.services {
		out = append(out, svc.Addresses()...)
	}
	return out
}

// Close stops all services. Idempotent.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, svc := range m.services {
		if err := svc.Close(); err != nil {
			m.logger.Warn("error closing privacy network",
				zap.String("network", svc.Name()), zap.Error(err))
		} else {
			m.logger.Info("privacy network closed", zap.String("network", svc.Name()))
		}
	}
	m.services = nil
	setActiveAddresses(nil)
}

// resolveMode picks a concrete mode for an "auto"/empty value.
func resolveMode(mode string, preferNative bool) string {
	switch mode {
	case "", "auto":
		if preferNative {
			return "native"
		}
		return "external"
	default:
		return mode
	}
}
