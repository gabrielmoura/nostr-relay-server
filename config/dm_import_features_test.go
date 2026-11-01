package config

import "testing"

func TestValidateDMImportFeatures(t *testing.T) {
	base := Config{Relay: RelayConfig{MaxEventSize: 300}, DMImport: DMImportConfig{Enabled: true, MaxEventBytes: 200, RateLimit: IPRequestRateLimitConfig{Enabled: true, RequestsPerSec: 1, Burst: 1, MaxClients: 1, IdleTTLSeconds: 1}}}
	if err := base.ValidateDMImportFeatures(); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	base.DMImport.MaxEventBytes = 301
	if err := base.ValidateDMImportFeatures(); err == nil {
		t.Fatal("expected oversize config to fail")
	}
	base.DMImport.MaxEventBytes = 200
	base.DMImport.RateLimit.Burst = 0
	if err := base.ValidateDMImportFeatures(); err == nil {
		t.Fatal("expected invalid rate limit to fail")
	}
	base.DMImport.RateLimit.Burst = 1
	base.DMImport.RateLimit.Enabled = false
	if err := base.ValidateDMImportFeatures(); err == nil {
		t.Fatal("expected disabled DM import rate limiter to fail")
	}
	base.DMImport.Enabled = false
	if err := base.ValidateDMImportFeatures(); err != nil {
		t.Fatalf("disabled feature must be inert: %v", err)
	}
}
