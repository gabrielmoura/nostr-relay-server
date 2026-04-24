package config

import (
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

func TestRelayInformationNormalizeKeysAcceptsNIP19AndDerivesPubKey(t *testing.T) {
	cfg := RelayInformationDocument{
		PrivKey: "nsec1nrhple709d2aqfpwcam3ec8p7uy52yusmr5m9878r87pswp0q7ysp8rk6k",
	}

	if err := cfg.NormalizeKeys(); err != nil {
		t.Fatalf("NormalizeKeys() error = %v", err)
	}

	const wantPriv = "98ee1fe7cf2b55d0242ec7771ce0e1f709451390d8e9b29fc719fc18382f0789"
	wantPub, err := nostr.GetPublicKey(wantPriv)
	if err != nil {
		t.Fatalf("GetPublicKey() error = %v", err)
	}

	if cfg.PrivKey != wantPriv {
		t.Fatalf("PrivKey = %q, want %q", cfg.PrivKey, wantPriv)
	}
	if cfg.PubKey != wantPub {
		t.Fatalf("PubKey = %q, want %q", cfg.PubKey, wantPub)
	}
}

func TestRelayInformationNormalizeKeysAcceptsNpubAndHex(t *testing.T) {
	cfg := RelayInformationDocument{
		PubKey:  "npub1dx0cw25dmteggk0uyp6tvks5zsrpdtq7ha6rlucua3c4wepfz5lsnspra4",
		PrivKey: "98ee1fe7cf2b55d0242ec7771ce0e1f709451390d8e9b29fc719fc18382f0789",
	}

	if err := cfg.NormalizeKeys(); err != nil {
		t.Fatalf("NormalizeKeys() error = %v", err)
	}

	const wantPub = "699f872a8ddaf28459fc2074b65a14140616ac1ebf743ff31cec71576429153f"

	if cfg.PubKey != wantPub {
		t.Fatalf("PubKey = %q, want %q", cfg.PubKey, wantPub)
	}
	if cfg.PrivKey != "98ee1fe7cf2b55d0242ec7771ce0e1f709451390d8e9b29fc719fc18382f0789" {
		t.Fatalf("PrivKey was unexpectedly changed: %q", cfg.PrivKey)
	}
}

func TestRelayInformationNormalizeKeysAllowsDifferentPublicKeyWhenExplicitlyConfigured(t *testing.T) {
	cfg := RelayInformationDocument{
		PubKey:  "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		PrivKey: "98ee1fe7cf2b55d0242ec7771ce0e1f709451390d8e9b29fc719fc18382f0789",
	}

	if err := cfg.NormalizeKeys(); err != nil {
		t.Fatalf("NormalizeKeys() error = %v", err)
	}
	if cfg.PubKey != "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" {
		t.Fatalf("PubKey = %q, want configured explicit value", cfg.PubKey)
	}
}

func TestRelayInformationCheckDoesNotPanicOnInvalidCanonicalURL(t *testing.T) {
	cfg := &RelayInformationDocument{
		URL:          "http://localhost:4869",
		CanonicalURL: "0.tcp.sa.ngrok.io:13305",
	}

	errs := cfg.Check()
	if len(errs) == 0 {
		t.Fatal("expected validation error for canonical_url without scheme")
	}
}
