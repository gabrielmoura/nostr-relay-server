package config

import "fmt"

func (cfg *Config) ValidateMarmotFeatures() error {
	if cfg == nil || !cfg.Marmot.MIP00.Enabled {
		return nil
	}
	if !cfg.Marmot.Enabled {
		return fmt.Errorf("marmot.mip00.enabled requires marmot.enabled")
	}
	if !cfg.Marmot.MIP00.AcceptKind30443 && !cfg.Marmot.MIP00.AcceptKind10051 && !cfg.Marmot.MIP00.AcceptLegacyKind443 {
		return fmt.Errorf("marmot.mip00.enabled requires at least one accepted event kind")
	}

	switch cfg.Marmot.MIP00.NormalizedValidationMode() {
	case "off", "basic":
	default:
		return fmt.Errorf("marmot.mip00.validation_mode must be one of: off, basic")
	}

	if cfg.Marmot.MIP00.MaxRelaysPerEvent <= 0 {
		return fmt.Errorf("marmot.mip00.max_relays_per_event must be greater than zero")
	}
	if cfg.Marmot.MIP00.MaxContentSizeBytes <= 0 {
		return fmt.Errorf("marmot.mip00.max_content_size_bytes must be greater than zero")
	}

	return nil
}
