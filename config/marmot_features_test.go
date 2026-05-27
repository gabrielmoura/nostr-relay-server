package config

import "testing"

func TestValidateMarmotFeaturesRequiresMasterFlag(t *testing.T) {
	cfg := &Config{
		Marmot: MarmotConfig{
			Enabled: false,
			MIP00: MarmotMIP00Config{
				Enabled:             true,
				AcceptKind30443:     true,
				MaxRelaysPerEvent:   10,
				MaxContentSizeBytes: 1024,
			},
		},
	}

	err := cfg.ValidateMarmotFeatures()
	if err == nil || err.Error() != "marmot.mip00.enabled requires marmot.enabled" {
		t.Fatalf("ValidateMarmotFeatures() error = %v", err)
	}
}

func TestValidateMarmotFeaturesRejectsUnsupportedValidationMode(t *testing.T) {
	cfg := &Config{
		Marmot: MarmotConfig{
			Enabled: true,
			MIP00: MarmotMIP00Config{
				Enabled:             true,
				AcceptKind30443:     true,
				ValidationMode:      "strict",
				MaxRelaysPerEvent:   10,
				MaxContentSizeBytes: 1024,
			},
		},
	}

	err := cfg.ValidateMarmotFeatures()
	if err == nil || err.Error() != "marmot.mip00.validation_mode must be one of: off, basic" {
		t.Fatalf("ValidateMarmotFeatures() error = %v", err)
	}
}

func TestValidateMarmotFeaturesAcceptsBasicMode(t *testing.T) {
	cfg := &Config{
		Marmot: MarmotConfig{
			Enabled: true,
			MIP00: MarmotMIP00Config{
				Enabled:             true,
				AcceptKind30443:     true,
				ValidationMode:      "basic",
				MaxRelaysPerEvent:   10,
				MaxContentSizeBytes: 1024,
			},
		},
	}

	if err := cfg.ValidateMarmotFeatures(); err != nil {
		t.Fatalf("ValidateMarmotFeatures() error = %v", err)
	}
}
