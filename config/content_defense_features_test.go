package config

import "testing"

func TestValidateContentDefenseFeatures(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "disabled configurations are inert",
			cfg: &Config{
				Security: SecurityConfig{
					Defense: SecurityDefenseConfig{
						ContentDedup: ContentDedupConfig{
							Mode:             "invalid",
							WindowSeconds:    0,
							ThresholdPubkeys: 0,
						},
					},
				},
			},
		},
		{
			name: "dedup threshold must be positive",
			cfg: &Config{
				Security: SecurityConfig{
					Defense: SecurityDefenseConfig{
						ContentDedup: ContentDedupConfig{
							Enabled:          true,
							Mode:             "observe",
							WindowSeconds:    60,
							ThresholdPubkeys: 0,
						},
					},
				},
			},
			wantErr: "security.defense.content_dedup.threshold_pubkeys must be at least one",
		},
		{
			name: "similarity window must be positive",
			cfg: &Config{
				Security: SecurityConfig{
					Defense: SecurityDefenseConfig{
						ContentSimilarity: ContentSimilarityConfig{
							Enabled:          true,
							Mode:             "flag",
							ThresholdPubkeys: 1,
							HammingThreshold: 3,
						},
					},
				},
			},
			wantErr: "security.defense.content_similarity.window_seconds must be greater than zero",
		},
		{
			name: "similarity hamming threshold is bounded",
			cfg: &Config{
				Security: SecurityConfig{
					Defense: SecurityDefenseConfig{
						ContentSimilarity: ContentSimilarityConfig{
							Enabled:          true,
							Mode:             "observe",
							WindowSeconds:    60,
							ThresholdPubkeys: 1,
							HammingThreshold: 65,
						},
					},
				},
			},
			wantErr: "security.defense.content_similarity.hamming_threshold must be between zero and 64",
		},
		{
			name: "flag and reject modes are valid",
			cfg: &Config{
				Security: SecurityConfig{
					Defense: SecurityDefenseConfig{
						ContentDedup: ContentDedupConfig{
							Enabled:          true,
							Mode:             "flag",
							WindowSeconds:    60,
							ThresholdPubkeys: 1,
						},
						ContentSimilarity: ContentSimilarityConfig{
							Enabled:          true,
							Mode:             "reject",
							WindowSeconds:    60,
							ThresholdPubkeys: 1,
							HammingThreshold: 3,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateContentDefenseFeatures()
			if tt.wantErr == "" && err != nil {
				t.Fatalf("ValidateContentDefenseFeatures() error = %v, want nil", err)
			}
			if tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr) {
				t.Fatalf("ValidateContentDefenseFeatures() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
