package groups

import (
	"context"
	"errors"
	"testing"

	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/nbd-wtf/go-nostr"
)

func TestFilterFiniteEventsPropagatesErrors(t *testing.T) {
	expected := errors.New("group lookup unavailable")
	events := []*nostr.Event{{ID: "visible"}, {ID: "unknown"}}
	visible, err := filterFiniteEvents(context.Background(), events, func(_ context.Context, event *nostr.Event) (bool, error) {
		if event.ID == "unknown" {
			return false, expected
		}
		return true, nil
	})
	if !errors.Is(err, expected) || visible != nil {
		t.Fatalf("visible=%#v err=%v", visible, err)
	}
}

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

func TestHiddenGroupPinnedMetadataDeniedAnonymously(t *testing.T) {
	_, metadata := readEventGroup(&nostr.Event{Kind: kindSimpleGroupPinnedEvents, Tags: nostr.Tags{{"d", "group"}}})
	allowed, err := (&Manager{}).canReadGroup(context.Background(), "", &dbstore.NIP29Group{GroupID: "group", Hidden: true}, metadata)
	if err != nil || allowed {
		t.Fatalf("allowed=%v err=%v", allowed, err)
	}
}

func TestReadEventGroupUsesDOnlyForGroupMetadata(t *testing.T) {
	for _, tc := range []struct {
		name     string
		event    *nostr.Event
		group    string
		metadata bool
	}{
		{"unrelated addressable event", &nostr.Event{Kind: 30023, Tags: nostr.Tags{{"d", "group"}}}, "", false},
		{"group content", &nostr.Event{Kind: 1, Tags: nostr.Tags{{"h", "group"}}}, "group", false},
		{"hidden group pinned metadata", &nostr.Event{Kind: kindSimpleGroupPinnedEvents, Tags: nostr.Tags{{"d", "group"}}}, "group", true},
		{"metadata with conflicting tags", &nostr.Event{Kind: kindSimpleGroupPinnedEvents, Tags: nostr.Tags{{"h", "public"}, {"d", "private"}}}, "private", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			group, metadata := readEventGroup(tc.event)
			if group != tc.group || metadata != tc.metadata {
				t.Fatalf("group=%q metadata=%v", group, metadata)
			}
		})
	}
}

func TestReadAuthorizerCachesGroupAndMembershipPerRead(t *testing.T) {
	groupCalls, memberCalls := 0, 0
	reader := &readAuthorizer{
		authed: "member",
		groups: make(map[string]groupReadResult), members: make(map[string]memberReadResult),
		lookupGroup: func(_ context.Context, id string) (*dbstore.NIP29Group, bool, error) {
			groupCalls++
			return &dbstore.NIP29Group{GroupID: id, Private: true}, true, nil
		},
		lookupMember: func(context.Context, string, string) (bool, error) { memberCalls++; return true, nil },
	}
	for i := 0; i < 5; i++ {
		ok, err := reader.canReadEvent(context.Background(), &nostr.Event{Kind: 1, Tags: nostr.Tags{{"h", "group"}}})
		if err != nil || !ok {
			t.Fatalf("allowed=%v err=%v", ok, err)
		}
	}
	if groupCalls != 1 || memberCalls != 1 {
		t.Fatalf("lookups: groups=%d members=%d", groupCalls, memberCalls)
	}
	ok, err := reader.canReadEvent(context.Background(), &nostr.Event{Kind: kindSimpleGroupPinnedEvents, Tags: nostr.Tags{{"h", "group"}}})
	if err != nil || ok {
		t.Fatalf("metadata without d allowed=%v err=%v", ok, err)
	}
}

func TestReadAuthorizerDeniesUnprojectedExplicitNIP29EventsButPreservesGenericHTags(t *testing.T) {
	reader := &readAuthorizer{
		groups:  make(map[string]groupReadResult),
		members: make(map[string]memberReadResult),
		lookupGroup: func(context.Context, string) (*dbstore.NIP29Group, bool, error) {
			return nil, false, nil
		},
		lookupMember: func(context.Context, string, string) (bool, error) { return false, nil },
	}

	explicitGroupOperation := &nostr.Event{Kind: nostr.KindSimpleGroupPutUser, Tags: nostr.Tags{{"h", "missing"}}}
	if allowed, err := reader.canReadEvent(context.Background(), explicitGroupOperation); err != nil || allowed {
		t.Fatalf("unprojected NIP-29 operation allowed=%v err=%v", allowed, err)
	}

	genericHTag := &nostr.Event{Kind: nostr.KindTextNote, Tags: nostr.Tags{{"h", "unrelated"}}}
	if allowed, err := reader.canReadEvent(context.Background(), genericHTag); err != nil || !allowed {
		t.Fatalf("generic h-tag event allowed=%v err=%v", allowed, err)
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

	if !m.shouldFilterQueryResults(nostr.Filter{Kinds: []int{nostr.KindTextNote}}) {
		t.Fatal("expected generic filter to keep post-query filtering enabled")
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
