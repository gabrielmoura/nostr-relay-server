package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestIPRateLimitDefaultsAreEnabledAndGenerous(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig() error = %v", err)
	}
	for name, limit := range map[string]IPRequestRateLimitConfig{
		"api":        cfg.API.RateLimit,
		"negentropy": cfg.NegentropyRateLimit,
	} {
		if !limit.Enabled || limit.RequestsPerSec <= 0 || limit.Burst <= 0 || limit.MaxClients <= 0 || limit.IdleTTLSeconds <= 0 {
			t.Fatalf("%s rate limit defaults = %#v, want enabled usable limits", name, limit)
		}
	}
}
