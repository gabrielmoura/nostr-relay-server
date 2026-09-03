package policies

import (
	"encoding/json"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
)

func newNIP70PolicyConfig() *config.Config {
	return &config.Config{
		Relay: config.RelayConfig{
			MaxEventSize:      1024 * 1024,
			MaxTagValueLength: 512,
			FilterLimit:       100,
		},
	}
}

func TestRejectProtectedEvent_NoTag(t *testing.T) {
	p := Policies{Config: newNIP70PolicyConfig()}
	evt := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{},
		Content:   "hello world",
	})

	reject, _ := p.rejectProtectedEvent(evt, "")
	if reject {
		t.Fatal("should not reject event without protected tag")
	}
}

func TestRejectProtectedEvent_WithTagNotAuthenticated(t *testing.T) {
	p := Policies{Config: newNIP70PolicyConfig()}
	evt := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{{"-"}},
		Content:   "secret message",
	})

	reject, reason := p.rejectProtectedEvent(evt, "")
	if !reject {
		t.Fatal("should reject protected event when not authenticated")
	}
	if reason != "auth-required: this event may only be published by its author" {
		t.Fatalf("unexpected reason: %q", reason)
	}
}

func TestRejectProtectedEvent_WithTagAuthenticatedAsAuthor(t *testing.T) {
	p := Policies{Config: newNIP70PolicyConfig()}
	evt := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{{"-"}},
		Content:   "secret message",
	})

	reject, _ := p.rejectProtectedEvent(evt, evt.PubKey)
	if reject {
		t.Fatal("should accept protected event when authenticated as the author")
	}
}

func TestRejectProtectedEvent_WithTagAuthenticatedAsDifferentUser(t *testing.T) {
	p := Policies{Config: newNIP70PolicyConfig()}
	evt := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{{"-"}},
		Content:   "secret message",
	})

	reject, reason := p.rejectProtectedEvent(evt, "aaaaaaaabbbbbbbbccccccccddddddddaaaaaaaabbbbbbbbccccccccdddddddd")
	if !reject {
		t.Fatal("should reject protected event when authenticated as a different user")
	}
	if reason != "restricted: protected event can only be published by its author" {
		t.Fatalf("unexpected reason: %q", reason)
	}
}

func TestRejectRepostOfProtectedEvent_NonRepostKind(t *testing.T) {
	p := Policies{Config: newNIP70PolicyConfig()}
	evt := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{},
		Content:   "not a repost",
	})

	reject, _ := p.rejectRepostOfProtectedEvent(evt)
	if reject {
		t.Fatal("should not reject non-repost events")
	}
}

func TestRejectRepostOfProtectedEvent_EmptyContent(t *testing.T) {
	p := Policies{Config: newNIP70PolicyConfig()}
	evt := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindRepost,
		Tags: nostr.Tags{
			{"e", "abc123"},
		},
		Content: "",
	})

	reject, _ := p.rejectRepostOfProtectedEvent(evt)
	if reject {
		t.Fatal("should not reject repost with empty content (no embedded event)")
	}
}

func TestRejectRepostOfProtectedEvent_EmbeddedProtectedEvent(t *testing.T) {
	p := Policies{Config: newNIP70PolicyConfig()}

	// Create a "protected" inner event.
	inner := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{{"-"}},
		Content:   "protected note",
	})
	innerJSON, _ := json.Marshal(inner)

	repost := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindRepost,
		Tags: nostr.Tags{
			{"e", inner.ID},
			{"p", inner.PubKey},
		},
		Content: string(innerJSON),
	})

	reject, reason := p.rejectRepostOfProtectedEvent(repost)
	if !reject {
		t.Fatal("should reject repost that embeds a protected event")
	}
	if reason != "restricted: repost must not embed a protected event" {
		t.Fatalf("unexpected reason: %q", reason)
	}
}

func TestRejectRepostOfProtectedEvent_EmbeddedNonProtectedEvent(t *testing.T) {
	p := Policies{Config: newNIP70PolicyConfig()}

	inner := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{},
		Content:   "normal note",
	})
	innerJSON, _ := json.Marshal(inner)

	repost := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindRepost,
		Tags: nostr.Tags{
			{"e", inner.ID},
			{"p", inner.PubKey},
		},
		Content: string(innerJSON),
	})

	reject, _ := p.rejectRepostOfProtectedEvent(repost)
	if reject {
		t.Fatal("should not reject repost of non-protected event")
	}
}

func TestRejectRepostOfProtectedEvent_GenericRepostKind16(t *testing.T) {
	p := Policies{Config: newNIP70PolicyConfig()}

	inner := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{{"-"}},
		Content:   "protected note",
	})
	innerJSON, _ := json.Marshal(inner)

	repost := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindGenericRepost,
		Tags: nostr.Tags{
			{"e", inner.ID},
			{"p", inner.PubKey},
		},
		Content: string(innerJSON),
	})

	reject, _ := p.rejectRepostOfProtectedEvent(repost)
	if reject == false {
		t.Fatal("should reject kind-16 generic repost that embeds a protected event")
	}
}

func TestRejectRepostOfProtectedEvent_InvalidJSON(t *testing.T) {
	p := Policies{Config: newNIP70PolicyConfig()}

	repost := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindRepost,
		Tags:      nostr.Tags{{"e", "abc123"}},
		Content:   "this is not valid json",
	})

	reject, _ := p.rejectRepostOfProtectedEvent(repost)
	if reject {
		t.Fatal("should not reject repost with invalid JSON content")
	}
}

func TestHasProtectedTag(t *testing.T) {
	tests := []struct {
		name string
		tags nostr.Tags
		want bool
	}{
		{"no tags", nostr.Tags{}, false},
		{"unrelated tags", nostr.Tags{{"e", "abc"}, {"p", "def"}}, false},
		{"dash tag with extra value", nostr.Tags{{"-", "extra"}}, false},
		{"correct protected tag", nostr.Tags{{"-"}}, true},
		{"protected tag among others", nostr.Tags{{"e", "abc"}, {"-"}, {"p", "def"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := &nostr.Event{Tags: tt.tags}
			if got := hasProtectedTag(evt); got != tt.want {
				t.Errorf("hasProtectedTag() = %v, want %v", got, tt.want)
			}
		})
	}
}
