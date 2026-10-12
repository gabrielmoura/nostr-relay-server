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
