package config

import "fmt"

func (cfg *Config) ValidateContentDefenseFeatures() error {
	if cfg == nil {
		return nil
	}
	if err := validateContentDedupConfig(cfg.Security.Defense.ContentDedup); err != nil {
		return err
	}
	return validateContentSimilarityConfig(cfg.Security.Defense.ContentSimilarity)
}

func validateContentDedupConfig(cfg ContentDedupConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if err := validateContentDefenseConfig(
		"security.defense.content_dedup",
		cfg.Mode,
		cfg.WindowSeconds,
		cfg.ThresholdPubkeys,
		cfg.MinContentLength,
	); err != nil {
		return err
	}
	return nil
}

func validateContentSimilarityConfig(cfg ContentSimilarityConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if err := validateContentDefenseConfig(
		"security.defense.content_similarity",
		cfg.Mode,
		cfg.WindowSeconds,
		cfg.ThresholdPubkeys,
		cfg.MinContentLength,
	); err != nil {
		return err
	}
	if cfg.HammingThreshold < 0 || cfg.HammingThreshold > 64 {
		return fmt.Errorf("security.defense.content_similarity.hamming_threshold must be between zero and 64")
	}
	return nil
}

func validateContentDefenseConfig(
	path string,
	mode string,
	windowSeconds int,
	thresholdPubkeys int,
	minContentLength int,
) error {
	switch mode {
	case "observe", "flag", "reject":
	default:
		return fmt.Errorf("%s.mode must be one of: observe, flag, reject", path)
	}
	if windowSeconds <= 0 {
		return fmt.Errorf("%s.window_seconds must be greater than zero", path)
	}
	if thresholdPubkeys < 1 {
		return fmt.Errorf("%s.threshold_pubkeys must be at least one", path)
	}
	if minContentLength < 0 {
		return fmt.Errorf("%s.min_content_length must not be negative", path)
	}
	return nil
}
