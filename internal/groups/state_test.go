package groups

import (
	"context"
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

func TestIsRelevantEventIncludesGroupContentWithGroupTag(t *testing.T) {
	t.Parallel()

	m := &Manager{enabled: true}
	evt := &nostr.Event{
		Kind: 1,
		Tags: nostr.Tags{{"h", "missing-group"}},
	}

	if !m.isRelevantEvent(evt) {
		t.Fatal("expected an event with h tag to be subject to NIP-29 validation")
	}
}

func TestIsRelevantEventKeepsExplicitNIP29Kinds(t *testing.T) {
	t.Parallel()

	m := &Manager{enabled: true}

	moderation := &nostr.Event{
		Kind: nostr.KindSimpleGroupPutUser,
		Tags: nostr.Tags{{"h", "group-1"}},
	}
	if !m.isRelevantEvent(moderation) {
		t.Fatal("expected moderation event to remain in NIP-29 scope")
	}

	metadata := &nostr.Event{Kind: nostr.KindSimpleGroupMetadata}
	if !m.isRelevantEvent(metadata) {
		t.Fatal("expected metadata event to remain in NIP-29 scope")
	}
}

func TestFilterScopeSeparatesPreValidationFromResultFiltering(t *testing.T) {
	t.Parallel()

	m := &Manager{enabled: true}

	hFilter := nostr.Filter{Tags: nostr.TagMap{"h": []string{"group-1"}}}
	if !m.shouldValidateFilter(hFilter) {
		t.Fatal("expected #h filter to require pre-validation")
	}
	if !m.shouldFilterQueryResults(hFilter) {
		t.Fatal("expected #h filter to keep result filtering enabled")
	}

	idFilter := nostr.Filter{IDs: []string{"event-id"}}
	if m.shouldValidateFilter(idFilter) {
		t.Fatal("expected id-only filter to skip pre-validation")
	}
	if !m.shouldFilterQueryResults(idFilter) {
		t.Fatal("expected id-only filter to keep post-query filtering enabled")
	}
}

func TestValidateIncomingEventBypassesGenericKindsWithoutGroupTag(t *testing.T) {
	t.Parallel()

	prev := M
	M = &Manager{enabled: true}
	t.Cleanup(func() {
		M = prev
	})

	evt := &nostr.Event{
		Kind: 1,
		Tags: nostr.Tags{{"t", "general"}},
	}

	reject, reason := ValidateIncomingEvent(context.Background(), evt)
	if reject {
		t.Fatalf("expected generic event without h tag to bypass NIP-29 rejection, got %q", reason)
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got %q", reason)
	}
}
