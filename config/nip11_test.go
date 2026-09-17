package config

import "testing"

func TestNIP11CapabilitiesReflectEnabledRelayFeatures(t *testing.T) {
	cfg := &Config{
		EnableNegentropy: true,
		NIP29:            NIP29Config{Enabled: true},
		NIP70:            NIP70Config{Enabled: true},
		NIP86:            NIP86Config{Enabled: true},
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

func TestNIP11CapabilitiesAdvertiseNIP50(t *testing.T) {
	cfg := &Config{}

	cfg.applyNIP11Capabilities()
	if !containsNIP(cfg.RelayInformation.SupportedNIPs, 50) {
		t.Fatalf("supported_nips = %v, want 50", cfg.RelayInformation.SupportedNIPs)
	}
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

func containsNIP(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
