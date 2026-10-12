package config

import (
	"fmt"
	"net/netip"
	"strings"
)

// ValidateRelaySyncFeatures rejects combinations that could advertise a live
// protocol without the durable state required to recover it safely.
func (cfg *Config) ValidateRelaySyncFeatures() error {
	r := cfg.RelaySync
	if !r.Enabled {
		if r.HTTPEnabled || r.SnapshotEnabled || r.ApplyEnabled || r.CatchupEnabled || r.LiveEnabled || r.ProjectorEnabled {
			return fmt.Errorf("relay_sync feature requires relay_sync.enabled")
		}
		return nil
	}
	if len(r.AllowedIPs) == 0 {
		return fmt.Errorf("relay_sync.allowed_ips must contain at least one IP or CIDR")
	}
	for _, raw := range r.AllowedIPs {
		value := strings.TrimSpace(raw)
		if value == "" {
			return fmt.Errorf("relay_sync.allowed_ips contains an empty entry")
		}
		if _, err := netip.ParseAddr(value); err != nil {
			if _, prefixErr := netip.ParsePrefix(value); prefixErr != nil {
				return fmt.Errorf("relay_sync.allowed_ips entry %q is not an IP or CIDR", raw)
			}
		}
	}
	if r.LiveEnabled && (!r.ProjectorEnabled || !r.CatchupEnabled || !r.ApplyEnabled) {
		return fmt.Errorf("relay_sync.live_enabled requires projector_enabled, catchup_enabled and apply_enabled")
	}
	if r.ProjectorEnabled {
		if r.ProjectorIntervalSeconds <= 0 {
			return fmt.Errorf("relay_sync.projector_interval_seconds must be positive")
		}
		if r.ProjectorBatchSize <= 0 {
			return fmt.Errorf("relay_sync.projector_batch_size must be positive")
		}
		if r.JournalEpoch == 0 {
			return fmt.Errorf("relay_sync.journal_epoch must be positive")
		}
		if r.RetentionBatchSize <= 0 {
			return fmt.Errorf("relay_sync.retention_batch_size must be positive")
		}
		if r.OutboxRetentionSeconds <= 0 || r.JournalRetentionSeconds <= 0 {
			return fmt.Errorf("relay_sync retention values must be positive")
		}
	}
	return nil
}
