package groups

import (
	"context"
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

func TestIsRelevantEventIgnoresNonNIP29KindsWithGroupTag(t *testing.T) {
	t.Parallel()

	m := &Manager{enabled: true}
	evt := &nostr.Event{
		Kind: 1,
		Tags: nostr.Tags{{"h", "missing-group"}},
	}

	if m.isRelevantEvent(evt) {
		t.Fatal("expected non-NIP29 kind with h tag to bypass NIP-29 write validation")
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

func TestValidateIncomingEventBypassesGenericKindsWithGroupTag(t *testing.T) {
	t.Parallel()

	prev := M
	M = &Manager{enabled: true}
	t.Cleanup(func() {
		M = prev
	})

	evt := &nostr.Event{
		Kind: 1,
		Tags: nostr.Tags{{"h", "missing-group"}},
	}

	reject, reason := ValidateIncomingEvent(context.Background(), evt)
	if reject {
		t.Fatalf("expected generic event to bypass NIP-29 rejection, got %q", reason)
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got %q", reason)
	}
}
