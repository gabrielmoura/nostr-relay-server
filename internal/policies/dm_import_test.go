package policies

import (
	"context"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
)

func TestDMImportKeepsKindBlacklist(t *testing.T) {
	cfg := newTestPolicyConfig()
	cfg.Relay.BlacklistKinds = []int{4}
	p := Policies{Config: cfg}
	event := mustSignEvent(t, nostr.Event{CreatedAt: nostr.Now(), Kind: 4, Content: "cipher"})
	reject, reason := p.ValidateDMImportEvent(context.Background(), event)
	if !reject || reason == "" {
		t.Fatalf("blacklisted DM import must be rejected, got reject=%v reason=%q", reject, reason)
	}
}

func TestDMImportRejectsInvalidIdentity(t *testing.T) {
	cfg := newTestPolicyConfig()
	p := Policies{Config: cfg}
	reject, _ := p.ValidateDMImportEvent(context.Background(), &nostr.Event{Kind: 4, PubKey: "invalid"})
	if !reject {
		t.Fatal("unsigned/invalid ID event must be rejected")
	}
}

func TestDMImportRejectsCorruptedSignature(t *testing.T) {
	cfg := newTestPolicyConfig()
	p := Policies{Config: cfg}
	event := mustSignEvent(t, nostr.Event{CreatedAt: nostr.Now(), Kind: 4, Content: "cipher"})
	event.Sig = "00" + event.Sig[2:]
	reject, _ := p.ValidateDMImportEvent(context.Background(), event)
	if !reject {
		t.Fatal("valid ID with corrupt signature must be rejected")
	}
}

func TestDMReadFiltersRequireInvolvedAuthentication(t *testing.T) {
	p := Policies{Config: &config.Config{Ws: config.WsConfig{Auth: true}}}
	owner := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, tt := range []struct {
		name   string
		filter nostr.Filter
		authed string
		reject bool
	}{
		{"giftwrap recipient", nostr.Filter{Kinds: []int{1059}, Tags: nostr.TagMap{"p": []string{owner}}}, owner, false},
		{"giftwrap unauthenticated", nostr.Filter{Kinds: []int{1059}, Tags: nostr.TagMap{"p": []string{owner}}}, "", true},
		{"giftwrap wrong recipient", nostr.Filter{Kinds: []int{1059}, Tags: nostr.TagMap{"p": []string{owner}}}, "other", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rejected, _ := p.checkDirectMessageAccess(tt.filter, &dto.WsServer{Authed: tt.authed})
			if rejected != tt.reject {
				t.Fatalf("reject = %v", rejected)
			}
		})
	}
}
