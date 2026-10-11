package config

import "testing"

func TestValidateRateLimitFeatures(t *testing.T) {
	valid := IPRequestRateLimitConfig{Enabled: true, RequestsPerSec: 1, Burst: 1, MaxClients: 1, IdleTTLSeconds: 1}
	for name, cfg := range map[string]*Config{
		"disabled zero values": {},
		"valid limits":         {API: APIConfig{RateLimit: valid}, NegentropyRateLimit: valid},
		"invalid api":          {API: APIConfig{RateLimit: IPRequestRateLimitConfig{Enabled: true}}},
		"invalid negentropy":   {NegentropyRateLimit: IPRequestRateLimitConfig{Enabled: true}},
	} {
		t.Run(name, func(t *testing.T) {
			err := cfg.ValidateRateLimitFeatures()
			if (name == "invalid api" || name == "invalid negentropy") == (err == nil) {
				t.Fatalf("ValidateRateLimitFeatures() error = %v", err)
			}
		})
	}
}
