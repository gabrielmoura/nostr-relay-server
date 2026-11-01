package dmimport

import (
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

func TestAllowed(t *testing.T) {
	const owner = "owner"
	for _, tt := range []struct {
		name  string
		event *nostr.Event
		want  bool
	}{
		{"gift wrap recipient", &nostr.Event{Kind: 1059, Tags: nostr.Tags{{"p", owner}}}, true},
		{"gift wrap other", &nostr.Event{Kind: 1059, Tags: nostr.Tags{{"p", "other"}}}, false},
		{"kind 4 author", &nostr.Event{Kind: 4, PubKey: owner}, true},
		{"kind 4 recipient", &nostr.Event{Kind: 4, Tags: nostr.Tags{{"p", owner}}}, true},
		{"relay list author", &nostr.Event{Kind: 10050, PubKey: owner}, true},
		{"relay list other", &nostr.Event{Kind: 10050, PubKey: "other"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := Allowed(owner, tt.event); got != tt.want {
				t.Fatalf("Allowed() = %v", got)
			}
		})
	}
}

func TestAuthorizeRejectsWrongIdentityForEachImportKind(t *testing.T) {
	for _, event := range []*nostr.Event{{Kind: 1059, Tags: nostr.Tags{{"p", "owner"}}}, {Kind: 4, PubKey: "owner"}, {Kind: 10050, PubKey: "owner"}} {
		if _, ok := Authorize("other", event); ok {
			t.Fatalf("wrong identity admitted kind %d", event.Kind)
		}
	}
}

func TestVisibleToRestrictsPrivateDelivery(t *testing.T) {
	for _, event := range []*nostr.Event{{Kind: 4, PubKey: "owner"}, {Kind: 1059, Tags: nostr.Tags{{"p", "owner"}}}} {
		if !VisibleTo("owner", event) || VisibleTo("other", event) {
			t.Fatalf("delivery privacy failed for kind %d", event.Kind)
		}
	}
}

func TestAllowedOwnedImportKindsRequireAuthorAuthentication(t *testing.T) {
	kinds := []int{0, 1, 3, 7, 20, 21, 22, 6, 16, 1111, 1063, 10000, 10002, 10003, 10063, 30000, 30023, 30024, 30078}
	for _, kind := range kinds {
		t.Run("kind", func(t *testing.T) {
			event := &nostr.Event{Kind: kind, PubKey: "owner"}
			if !IsImportKind(kind) || !Allowed("owner", event) {
				t.Fatalf("owner was not admitted for kind %d", kind)
			}
			if Allowed("other", event) {
				t.Fatalf("other identity was admitted for kind %d", kind)
			}
		})
	}
}

func TestVisibleToLeavesConcordGiftwrapToItsAuthorPolicy(t *testing.T) {
	event := &nostr.Event{Kind: 1059, PubKey: "stream-author"}
	if VisibleTo("unrelated", event) {
		t.Fatal("giftwrap without a p tag was visible while Concord was disabled")
	}
	if !VisibleToWithConcord("unrelated", event, true) {
		t.Fatal("Concord giftwrap without a p tag was hidden before author policy could apply")
	}
}
