package nip86

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
)

func TestAuthenticateSuccess(t *testing.T) {
	body := []byte(`{"method":"supportedmethods","params":[]}`)
	requestURL := "http://relay.example.com/"
	privKey := nostr.GeneratePrivateKey()
	pubKey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		t.Fatalf("get public key: %v", err)
	}

	cfg := &config.Config{AdminPubKey: pubKey, NIP86: config.NIP86Config{Enabled: true, AuthWindowSeconds: 60}}
	authHeader := buildAuthHeader(t, privKey, requestURL, body)

	result, err := Authenticate(cfg, AuthInput{Authorization: authHeader, Method: "POST", URL: requestURL, Body: body})
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if result.PubKey != pubKey {
		t.Fatalf("Authenticate() pubkey = %q want %q", result.PubKey, pubKey)
	}
}

func TestAuthenticateRejectsPayloadMismatch(t *testing.T) {
	body := []byte(`{"method":"supportedmethods","params":[]}`)
	requestURL := "http://relay.example.com/"
	privKey := nostr.GeneratePrivateKey()
	pubKey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		t.Fatalf("get public key: %v", err)
	}

	cfg := &config.Config{AdminPubKey: pubKey, NIP86: config.NIP86Config{Enabled: true, AuthWindowSeconds: 60}}
	authHeader := buildAuthHeader(t, privKey, requestURL, []byte(`{"wrong":true}`))

	if _, err := Authenticate(cfg, AuthInput{Authorization: authHeader, Method: "POST", URL: requestURL, Body: body}); err == nil {
		t.Fatal("Authenticate() error = nil, want payload mismatch")
	}
}

func buildAuthHeader(t *testing.T, privKey, requestURL string, body []byte) string {
	t.Helper()
	sum := sha256.Sum256(body)
	evt := nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      27235,
		Tags: nostr.Tags{
			{"u", requestURL},
			{"method", "POST"},
			{"payload", hex.EncodeToString(sum[:])},
		},
	}
	if err := evt.Sign(privKey); err != nil {
		t.Fatalf("sign event: %v", err)
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	return authScheme + base64.StdEncoding.EncodeToString(payload)
}
