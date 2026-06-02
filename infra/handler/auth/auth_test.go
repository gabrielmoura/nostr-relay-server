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
