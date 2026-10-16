package config

import "testing"

func TestValidateRelaySyncFeatures(t *testing.T) {
	base := RelaySyncConfig{Enabled: true, AllowedIPs: []string{"127.0.0.1"}, ProjectorEnabled: true, ProjectorIntervalSeconds: 1, ProjectorBatchSize: 10, JournalEpoch: 1, OutboxRetentionSeconds: 3600, JournalRetentionSeconds: 86400, RetentionBatchSize: 100}
	tests := []struct {
		name string
		cfg  RelaySyncConfig
		want bool
	}{
		{"disabled defaults", RelaySyncConfig{}, false},
		{"disabled feature", RelaySyncConfig{HTTPEnabled: true}, true},
		{"http requires snapshot", func() RelaySyncConfig { c := base; c.HTTPEnabled = true; return c }(), true},
		{"valid projector", base, false},
		{"live requires catchup", func() RelaySyncConfig { c := base; c.LiveEnabled = true; c.ApplyEnabled = true; return c }(), true},
		{"invalid interval", func() RelaySyncConfig { c := base; c.ProjectorIntervalSeconds = 0; return c }(), true},
		{"resilience validates retries", func() RelaySyncConfig {
			c := base
			c.Client.Resilience = RelaySyncResilienceConfig{Enabled: true, MaxAttempts: 0, InitialBackoffMilliseconds: 1, MaxBackoffMilliseconds: 1}
			return c
		}(), true},
		{"resilience validates backoff", func() RelaySyncConfig {
			c := base
			c.Client.Resilience = RelaySyncResilienceConfig{Enabled: true, MaxAttempts: 3, InitialBackoffMilliseconds: 100, MaxBackoffMilliseconds: 10}
			return c
		}(), true},
		{"resilience enabled", func() RelaySyncConfig {
			c := base
			c.Client.Resilience = RelaySyncResilienceConfig{Enabled: true, MaxAttempts: 3, InitialBackoffMilliseconds: 10, MaxBackoffMilliseconds: 100}
			return c
		}(), false},
		{"fetch then subscribe fails closed", func() RelaySyncConfig { c := base; c.Client.FetchThenSubscribe.Enabled = true; return c }(), true},
		{"multi relay fails closed", func() RelaySyncConfig { c := base; c.Client.MultiRelay.Enabled = true; return c }(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{RelaySync: tt.cfg}
			if err := cfg.ValidateRelaySyncFeatures(); (err != nil) != tt.want {
				t.Fatalf("ValidateRelaySyncFeatures() error = %v, wantErr %v", err, tt.want)
			}
		})
	}
}
