package event

import (
	"context"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/ingestion"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	policies "github.com/gabrielmoura/nostr-relay-server/internal/policies"
	"github.com/nbd-wtf/go-nostr"
)

func TestDMImportRateLimitRejectsBeforeQueueing(t *testing.T) {
	previous := config.Cfg
	previousPolicies := policies.P
	config.Cfg = &config.Config{DMImport: config.DMImportConfig{Enabled: true, MaxEventBytes: 1024}}
	policies.P = &policies.Policies{Config: config.Cfg}
	t.Cleanup(func() { config.Cfg = previous; policies.P = previousPolicies })
	ws := &dto.WsServer{Ctx: context.Background(), Authed: "owner", RemoteIP: "192.0.2.1", ChanSender: make(chan any, 1), DMImportAllow: func(string) bool { return false }}
	processEvent(ws, &nostr.Event{ID: "event", Kind: 4, PubKey: "owner", Content: "cipher"})
	frame := <-ws.ChanSender
	ok, isOK := frame.(nostr.OKEnvelope)
	if !isOK || ok.OK || ok.Reason != "rate-limited: DM import rate limit exceeded" {
		t.Fatalf("unexpected response %#v", frame)
	}
}

func TestDMImportRequiresAuth(t *testing.T) {
	previous := config.Cfg
	config.Cfg = &config.Config{DMImport: config.DMImportConfig{Enabled: true, MaxEventBytes: 1024}, Ws: config.WsConfig{Auth: true}}
	t.Cleanup(func() { config.Cfg = previous })
	ws := &dto.WsServer{Ctx: context.Background(), ChanSender: make(chan any, 2), Challenge: "challenge"}
	processEvent(ws, &nostr.Event{ID: "event", Kind: 4})
	<-ws.ChanSender // AUTH challenge
	frame := <-ws.ChanSender
	if ok := frame.(nostr.OKEnvelope); ok.OK || ok.Reason != "auth-required: this relay requires NIP-42 authentication before EVENT" {
		t.Fatalf("unexpected %#v", frame)
	}
}

func TestDMImportDuplicateReceiptIsSuccessful(t *testing.T) {
	got := dmImportOK("event", ingestion.Receipt{Duplicate: true})
	if !got.OK || got.Reason != "duplicate: event already exists" {
		t.Fatalf("duplicate response %#v", got)
	}
}
