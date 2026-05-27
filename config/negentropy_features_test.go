package config

import "testing"

func TestValidateNegentropyFeatures(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "disabled is inert",
			cfg:     &Config{},
			wantErr: false,
		},
		{
			name: "requires negentropy enabled",
			cfg: &Config{
				NegentropyAuth: true,
				Ws:             WsConfig{AuthMode: "optional"},
				RelayInformation: RelayInformationDocument{
					PubKey: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			wantErr: true,
		},
		{
			name: "requires websocket auth",
			cfg: &Config{
				EnableNegentropy: true,
				NegentropyAuth:   true,
				RelayInformation: RelayInformationDocument{
					PubKey: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			wantErr: true,
		},
		{
			name: "requires relay pubkey",
			cfg: &Config{
				EnableNegentropy: true,
				NegentropyAuth:   true,
				Ws:               WsConfig{AuthMode: "optional"},
			},
			wantErr: true,
		},
		{
			name: "valid configuration",
			cfg: &Config{
				EnableNegentropy: true,
				NegentropyAuth:   true,
				Ws:               WsConfig{AuthMode: "optional"},
				RelayInformation: RelayInformationDocument{
					PubKey: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateNegentropyFeatures()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestNegentropyAuthorizedPubKey(t *testing.T) {
	cfg := &Config{RelayInformation: RelayInformationDocument{PubKey: "  abc  "}}
	if got := cfg.NegentropyAuthorizedPubKey(); got != "abc" {
		t.Fatalf("NegentropyAuthorizedPubKey() = %q, want %q", got, "abc")
	}
}
