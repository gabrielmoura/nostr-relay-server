package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNIP11CapabilitiesReflectEnabledRelayFeatures(t *testing.T) {
	cfg := &Config{
		EnableNegentropy: true,
		NIP29:            NIP29Config{Enabled: true},
		NIP70:            NIP70Config{Enabled: true},
		NIP86:            NIP86Config{Enabled: true},
		Search:           SearchConfig{Enabled: true},
		Relay: RelayConfig{
			MinimumPOWLimit: 20,
			VanishEvent:     true,
		},
		Store: StoreConfig{Enabled: true},
		Ws:    WsConfig{Auth: true},
	}

	cfg.applyNIP11Capabilities()
	for _, nip := range []int{1, 9, 11, 13, 29, 40, 42, 45, 50, 62, 70, 77, 86, 96, 98} {
		if !containsNIP(cfg.RelayInformation.SupportedNIPs, nip) {
			t.Fatalf("supported_nips = %v, want %d", cfg.RelayInformation.SupportedNIPs, nip)
		}
	}
}

func TestNIP11CapabilitiesRemoveDisabledOptionalFeatures(t *testing.T) {
	cfg := &Config{
		RelayInformation: RelayInformationDocument{
			SupportedNIPs: []int{1, 9, 11, 13, 29, 40, 42, 45, 50, 62, 70, 77, 86, 96, 98},
		},
	}

	cfg.applyNIP11Capabilities()
	for _, nip := range []int{13, 29, 42, 62, 70, 77, 86, 96, 98} {
		if containsNIP(cfg.RelayInformation.SupportedNIPs, nip) {
			t.Fatalf("supported_nips = %v, unexpectedly contains %d", cfg.RelayInformation.SupportedNIPs, nip)
		}
	}
}

func TestNIP11CapabilitiesAdvertiseEnabledNIP50(t *testing.T) {
	cfg := &Config{Search: SearchConfig{Enabled: true}}

	cfg.applyNIP11Capabilities()
	if !containsNIP(cfg.RelayInformation.SupportedNIPs, 50) {
		t.Fatalf("supported_nips = %v, want 50", cfg.RelayInformation.SupportedNIPs)
	}
}

func TestNIP11CapabilitiesDoNotAdvertiseDisabledNIP50(t *testing.T) {
	cfg := &Config{RelayInformation: RelayInformationDocument{SupportedNIPs: []int{50}}}

	cfg.applyNIP11Capabilities()
	require.False(t, containsNIP(cfg.RelayInformation.SupportedNIPs, 50))
}

func TestPublicNIP11IncludesAmethystCompatibleMetadataWithoutPrivateKey(t *testing.T) {
	doc := (&RelayInformationDocument{
		PrivKey:                "secret",
		PrivacyPolicy:          "https://relay.example/privacy",
		Retention:              []RelayRetentionDocument{{Kinds: []int{1}, Time: 86400}},
		NIP50:                  []string{"search"},
		SupportedNIPExtensions: []string{"example"},
		SupportedGRASPs:        []string{"42"},
	}).PublicNIP11().(publicRelayInformationDocument)

	if doc.PrivacyPolicy == "" || len(doc.Retention) != 1 || len(doc.NIP50) != 1 || len(doc.SupportedNIPExtensions) != 1 || len(doc.SupportedGRASPs) != 1 {
		t.Fatalf("public NIP-11 document did not retain configured metadata: %#v", doc)
	}
}

func TestRelayInformationDocumentEffectiveIcon(t *testing.T) {
	tests := []struct {
		name string
		cfg  RelayInformationDocument
		want string
	}{
		{
			name: "preserves explicit icon",
			cfg: RelayInformationDocument{
				Icon:         "https://cdn.example/icon.png",
				CanonicalURL: "wss://relay.example",
			},
			want: "https://cdn.example/icon.png",
		},
		{
			name: "derives HTTP icon from websocket canonical URL",
			cfg:  RelayInformationDocument{CanonicalURL: "ws://relay.example:8080/relay?ignored=true"},
			want: "http://relay.example:8080/nostr.png",
		},
		{
			name: "derives HTTPS icon from secure websocket canonical URL",
			cfg:  RelayInformationDocument{CanonicalURL: "wss://relay.example"},
			want: "https://relay.example/nostr.png",
		},
		{
			name: "falls back to public HTTP URL",
			cfg: RelayInformationDocument{
				CanonicalURL: "not-a-websocket-url",
				URL:          "https://relay.example/base",
			},
			want: "https://relay.example/nostr.png",
		},
		{
			name: "omits icon without usable public URL",
			cfg:  RelayInformationDocument{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.EffectiveIcon(); got != tt.want {
				t.Fatalf("EffectiveIcon() = %q, want %q", got, tt.want)
			}

			doc := tt.cfg.PublicNIP11().(publicRelayInformationDocument)
			if doc.Icon != tt.want {
				t.Fatalf("PublicNIP11().Icon = %q, want %q", doc.Icon, tt.want)
			}
		})
	}
}

func containsNIP(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
