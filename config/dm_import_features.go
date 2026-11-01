package config

import "fmt"

// ValidateDMImportFeatures fails closed for the optional DM import bypass.
func (cfg *Config) ValidateDMImportFeatures() error {
	if !cfg.DMImport.Enabled {
		return nil
	}
	if cfg.DMImport.MaxEventBytes <= 0 || cfg.DMImport.MaxEventBytes > cfg.Relay.MaxEventSize {
		return fmt.Errorf("dm_import.max_event_bytes must be positive and no greater than relay.max_size_event_in_bytes")
	}
	r := cfg.DMImport.RateLimit
	if !r.Enabled {
		return fmt.Errorf("dm_import.rate_limit must be enabled when dm_import is enabled")
	}
	if r.RequestsPerSec <= 0 || r.Burst <= 0 || r.MaxClients <= 0 || r.IdleTTLSeconds <= 0 {
		return fmt.Errorf("dm_import.rate_limit values must be positive when enabled")
	}
	return nil
}
