package auth

import (
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

func TestValidateAuthEventAcceptsTrailingSlashDifference(t *testing.T) {
	privKey := nostr.GeneratePrivateKey()
	pubKey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		t.Fatalf("get public key: %v", err)
	}

	evt := nostr.Event{
		PubKey:    pubKey,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindClientAuthentication,
		Tags: nostr.Tags{
			{"relay", "ws://relay.example.com/"},
			{"challenge", "test-challenge"},
		},
	}
	if err := evt.Sign(privKey); err != nil {
		t.Fatalf("sign event: %v", err)
	}

	authedPubkey, reason := validateAuthEvent(&evt, "test-challenge", "ws://relay.example.com")
	if reason != "" {
		t.Fatalf("validateAuthEvent() reason = %q, want success", reason)
	}
	if authedPubkey != pubKey {
		t.Fatalf("validateAuthEvent() pubkey = %q, want %q", authedPubkey, pubKey)
	}
}

func TestValidateAuthEventRejectsRelayPathMismatch(t *testing.T) {
	privKey := nostr.GeneratePrivateKey()
	pubKey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		t.Fatalf("get public key: %v", err)
	}

	evt := nostr.Event{
		PubKey:    pubKey,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindClientAuthentication,
		Tags: nostr.Tags{
			{"relay", "ws://relay.example.com/"},
			{"challenge", "test-challenge"},
		},
	}
	if err := evt.Sign(privKey); err != nil {
		t.Fatalf("sign event: %v", err)
	}

	_, reason := validateAuthEvent(&evt, "test-challenge", "ws://relay.example.com/relay")
	if reason != "relay_mismatch" {
		t.Fatalf("validateAuthEvent() reason = %q, want relay_mismatch", reason)
	}
}

func TestValidateAuthEventAcceptsCanonicalAndActiveOnionURLs(t *testing.T) {
	privKey := nostr.GeneratePrivateKey()
	pubKey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		t.Fatalf("get public key: %v", err)
	}

	tests := []struct {
		name     string
		relayURL string
		want     string
	}{
		{
			name:     "canonical URL",
			relayURL: "wss://relay.example.com",
		},
		{
			name:     "active onion URL",
			relayURL: "ws://activeonion.onion:8080",
		},
		{
			name:     "active I2P URL",
			relayURL: "ws://activei2p.b32.i2p",
		},
		{
			name:     "active Yggdrasil URL",
			relayURL: "ws://[200:db8::1]:8080",
		},
		{
			name:     "unknown onion URL",
			relayURL: "ws://unknownonion.onion:8080",
			want:     "relay_mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := nostr.Event{
				PubKey:    pubKey,
				CreatedAt: nostr.Now(),
				Kind:      nostr.KindClientAuthentication,
				Tags: nostr.Tags{
					{"relay", tt.relayURL},
					{"challenge", "test-challenge"},
				},
			}
			if err := evt.Sign(privKey); err != nil {
				t.Fatalf("sign event: %v", err)
			}

			got, reason := validateAuthEvent(
				&evt,
				"test-challenge",
				"wss://relay.example.com",
				"ws://activeonion.onion:8080",
				"ws://activei2p.b32.i2p",
				"ws://[200:db8::1]:8080",
			)
			if reason != tt.want {
				t.Fatalf("validateAuthEvent() reason = %q, want %q", reason, tt.want)
			}
			if tt.want == "" && got != pubKey {
				t.Fatalf("validateAuthEvent() pubkey = %q, want %q", got, pubKey)
			}
		})
	}
}

func TestValidateAuthEventAcceptsDefaultWebSocketPort(t *testing.T) {
	privKey := nostr.GeneratePrivateKey()
	pubKey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		t.Fatalf("get public key: %v", err)
	}

	evt := nostr.Event{
		PubKey:    pubKey,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindClientAuthentication,
		Tags: nostr.Tags{
			{"relay", "ws://publishedonion.onion:80"},
			{"challenge", "test-challenge"},
		},
	}
	if err := evt.Sign(privKey); err != nil {
		t.Fatalf("sign event: %v", err)
	}

	if _, reason := validateAuthEvent(&evt, "test-challenge", "wss://relay.example.com", "ws://publishedonion.onion"); reason != "" {
		t.Fatalf("validateAuthEvent() reason = %q, want success", reason)
	}
}

func TestValidateAuthEventRejectsMalformedRelayURLs(t *testing.T) {
	privKey := nostr.GeneratePrivateKey()
	pubKey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		t.Fatalf("get public key: %v", err)
	}

	tests := []struct {
		name     string
		relayURL string
	}{
		{name: "credentials", relayURL: "ws://user:password@publishedonion.onion"},
		{name: "query", relayURL: "ws://publishedonion.onion?token=secret"},
		{name: "fragment", relayURL: "ws://publishedonion.onion#fragment"},
		{name: "non websocket scheme", relayURL: "https://publishedonion.onion"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := nostr.Event{
				PubKey:    pubKey,
				CreatedAt: nostr.Now(),
				Kind:      nostr.KindClientAuthentication,
				Tags: nostr.Tags{
					{"relay", tt.relayURL},
					{"challenge", "test-challenge"},
				},
			}
			if err := evt.Sign(privKey); err != nil {
				t.Fatalf("sign event: %v", err)
			}

			if _, reason := validateAuthEvent(&evt, "test-challenge", "wss://relay.example.com", "ws://publishedonion.onion"); reason != "invalid_relay_tag" {
				t.Fatalf("validateAuthEvent() reason = %q, want invalid_relay_tag", reason)
			}
		})
	}
}

func TestValidateAuthEventPreservesRelayPathCase(t *testing.T) {
	privKey := nostr.GeneratePrivateKey()
	pubKey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		t.Fatalf("get public key: %v", err)
	}

	evt := nostr.Event{
		PubKey:    pubKey,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindClientAuthentication,
		Tags: nostr.Tags{
			{"relay", "ws://publishedonion.onion/Relay"},
			{"challenge", "test-challenge"},
		},
	}
	if err := evt.Sign(privKey); err != nil {
		t.Fatalf("sign event: %v", err)
	}

	if _, reason := validateAuthEvent(&evt, "test-challenge", "wss://relay.example.com", "ws://publishedonion.onion/relay"); reason != "relay_mismatch" {
		t.Fatalf("validateAuthEvent() reason = %q, want relay_mismatch", reason)
	}
}
