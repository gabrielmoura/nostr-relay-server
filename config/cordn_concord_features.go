package config

import "fmt"

// ValidateCordnConcordFeatures keeps both optional protocol integrations
// fail-closed while preserving the existing relay defaults.
func (cfg *Config) ValidateCordnConcordFeatures() error {
	if cfg.Cordn.Enabled {
		if cfg.RelayInformation.PrivKey == "" {
			return fmt.Errorf("cordn.enabled requires relay_information.priv_key")
		}
		if cfg.Cordn.MaxMessageBytes < 1 || cfg.Cordn.MaxMessageBytes > cfg.Relay.MaxEventSize {
			return fmt.Errorf("cordn.max_message_bytes must be positive and no greater than relay.max_size_event_in_bytes")
		}
		if cfg.Cordn.MaxPendingJoinRequests < 1 {
			return fmt.Errorf("cordn.max_pending_join_requests must be positive")
		}
		if cfg.Cordn.RetentionDays < 1 {
			return fmt.Errorf("cordn.retention_days must be positive")
		}
		if cfg.Cordn.MaxConcurrentSubscriptions < 1 {
			return fmt.Errorf("cordn.max_concurrent_subscriptions must be positive")
		}
	}
	return nil
}
