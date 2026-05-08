package http

import (
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func ensureTestConfig() {
	if config.Cfg == nil {
		config.Cfg = &config.Config{}
	}
}

func TestBuildAdminLabelEvent(t *testing.T) {
	ensureTestConfig()
	privKey := nostr.GeneratePrivateKey()
	config.Cfg.RelayInformation.PrivKey = privKey
	t.Cleanup(func() {
		config.Cfg.RelayInformation.PrivKey = ""
	})

	pubKey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		t.Fatalf("GetPublicKey() error = %v", err)
	}
	npub, err := nip19.EncodePublicKey(pubKey)
	if err != nil {
		t.Fatalf("EncodePublicKey() error = %v", err)
	}

	evt, err := buildAdminLabelEvent(adminCreateLabelRequest{
		Namespace: "ugc",
		Labels:    []string{"Spam", "spam", "Scam"},
		Comment:   "  flood detected  ",
		Target: adminCreateLabelTargetRequest{
			Type:      "pubkey",
			Value:     npub,
			RelayHint: "wss://relay.example",
		},
	}, time.Unix(1778000000, 0).UTC())
	if err != nil {
		t.Fatalf("buildAdminLabelEvent() error = %v", err)
	}

	if evt.Kind != 1985 {
		t.Fatalf("event kind = %d want 1985", evt.Kind)
	}
	if evt.Content != "flood detected" {
		t.Fatalf("event content = %q want %q", evt.Content, "flood detected")
	}
	if len(evt.Tags) != 4 {
		t.Fatalf("len(tags) = %d want 4", len(evt.Tags))
	}
	if evt.Tags[0][0] != "L" || evt.Tags[0][1] != "ugc" {
		t.Fatalf("namespace tag = %#v", evt.Tags[0])
	}
	if evt.Tags[1][0] != "l" || evt.Tags[1][1] != "spam" || evt.Tags[1][2] != "ugc" {
		t.Fatalf("first label tag = %#v", evt.Tags[1])
	}
	if evt.Tags[2][0] != "l" || evt.Tags[2][1] != "scam" || evt.Tags[2][2] != "ugc" {
		t.Fatalf("second label tag = %#v", evt.Tags[2])
	}
	if evt.Tags[3][0] != "p" || evt.Tags[3][1] != pubKey || evt.Tags[3][2] != "wss://relay.example" {
		t.Fatalf("target tag = %#v", evt.Tags[3])
	}

	if err := evt.Sign(privKey); err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if ok, err := evt.CheckSignature(); err != nil || !ok {
		t.Fatalf("CheckSignature() = (%v, %v), want (true, nil)", ok, err)
	}
}

func TestBuildAdminLabelEventRequiresValidTargetType(t *testing.T) {
	ensureTestConfig()
	config.Cfg.RelayInformation.PrivKey = nostr.GeneratePrivateKey()
	t.Cleanup(func() {
		config.Cfg.RelayInformation.PrivKey = ""
	})

	_, err := buildAdminLabelEvent(adminCreateLabelRequest{
		Namespace: "ugc",
		Labels:    []string{"spam"},
		Target: adminCreateLabelTargetRequest{
			Type:  "invalid",
			Value: "abc",
		},
	}, time.Now().UTC())
	if err == nil {
		t.Fatal("buildAdminLabelEvent() error = nil, want invalid target type")
	}
}

func TestBuildAdminLabelEventRequiresNamespaceAndLabels(t *testing.T) {
	ensureTestConfig()
	config.Cfg.RelayInformation.PrivKey = nostr.GeneratePrivateKey()
	t.Cleanup(func() {
		config.Cfg.RelayInformation.PrivKey = ""
	})

	_, err := buildAdminLabelEvent(adminCreateLabelRequest{
		Target: adminCreateLabelTargetRequest{Type: "topic", Value: "nostr"},
	}, time.Now().UTC())
	if err == nil {
		t.Fatal("buildAdminLabelEvent() error = nil, want validation error")
	}
}
