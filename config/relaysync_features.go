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
	if err := validateRelaySyncClientFeatures(r.Client); err != nil {
		return err
	}
	if !r.Enabled {
		if r.HTTPEnabled || r.SnapshotEnabled || r.ApplyEnabled || r.CatchupEnabled || r.LiveEnabled || r.ProjectorEnabled {
			return fmt.Errorf("relay_sync feature requires relay_sync.enabled")
		}
		return nil
	}
	if len(r.AllowedIPs) == 0 {
		return fmt.Errorf("relay_sync.allowed_ips must contain at least one IP or CIDR")
	}
	if r.HTTPEnabled && !r.SnapshotEnabled {
		return fmt.Errorf("relay_sync.http_enabled requires relay_sync.snapshot_enabled")
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

func validateRelaySyncClientFeatures(client RelaySyncClientConfig) error {
	r := client.Resilience
	if r.Enabled {
		if r.MaxAttempts < 1 || r.MaxAttempts > 10 {
			return fmt.Errorf("relay_sync.client.resilience.max_attempts must be between 1 and 10")
		}
		if r.InitialBackoffMilliseconds < 1 || r.MaxBackoffMilliseconds < r.InitialBackoffMilliseconds || r.MaxBackoffMilliseconds > 5000 {
			return fmt.Errorf("relay_sync.client.resilience backoff values must be positive, ordered, and at most 5000 milliseconds")
		}
	}
	// These strategies need a client-side handoff/orchestrator that does not
	// yet exist. Fail closed instead of accepting a no-op feature flag.
	if client.FetchThenSubscribe.Enabled {
		return fmt.Errorf("relay_sync.client.fetch_then_subscribe is not implemented")
	}
	if client.MultiRelay.Enabled {
		if client.MultiRelay.MaxParallelRelays < 1 || client.MultiRelay.MaxParallelRelays > 4 {
			return fmt.Errorf("relay_sync.client.multi_relay.max_parallel_relays must be between 1 and 4")
		}
		return fmt.Errorf("relay_sync.client.multi_relay is not implemented")
	}
	return nil
}
