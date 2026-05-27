package ingestion

import (
	"context"
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

func TestPrepareEventForStorageReplacesOlderAddressableKind30443(t *testing.T) {
	prevQuery := queryFirstStoredEvent
	prevDelete := deleteStoredEventByID
	t.Cleanup(func() {
		queryFirstStoredEvent = prevQuery
		deleteStoredEventByID = prevDelete
	})

	older := &nostr.Event{
		ID:        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		PubKey:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Kind:      30443,
		CreatedAt: nostr.Timestamp(100),
		Tags:      nostr.Tags{{"d", "slot-1"}},
	}
	newer := &nostr.Event{
		ID:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PubKey:    older.PubKey,
		Kind:      30443,
		CreatedAt: nostr.Timestamp(200),
		Tags:      nostr.Tags{{"d", "slot-1"}},
	}

	var gotFilter nostr.Filter
	var deleteCalls int
	var deletedID string
	var deletedBy string

	queryFirstStoredEvent = func(_ context.Context, filter nostr.Filter) (*nostr.Event, error) {
		gotFilter = filter
		return older, nil
	}
	deleteStoredEventByID = func(_ context.Context, id, replacedBy string) error {
		deleteCalls++
		deletedID = id
		deletedBy = replacedBy
		return nil
	}

	if err := prepareEventForStorage(context.Background(), newer); err != nil {
		t.Fatalf("prepareEventForStorage() error = %v", err)
	}

	if len(gotFilter.Authors) != 1 || gotFilter.Authors[0] != newer.PubKey {
		t.Fatalf("prepareEventForStorage() authors filter = %#v", gotFilter.Authors)
	}
	if len(gotFilter.Kinds) != 1 || gotFilter.Kinds[0] != 30443 {
		t.Fatalf("prepareEventForStorage() kinds filter = %#v", gotFilter.Kinds)
	}
	dValues := gotFilter.Tags["d"]
	if len(dValues) != 1 || dValues[0] != "slot-1" {
		t.Fatalf("prepareEventForStorage() d-tag filter = %#v", dValues)
	}
	if deleteCalls != 1 {
		t.Fatalf("deleteStoredEventByID call count = %d, want 1", deleteCalls)
	}
	if deletedID != older.ID {
		t.Fatalf("deleted event id = %q, want %q", deletedID, older.ID)
	}
	if deletedBy != newer.ID {
		t.Fatalf("deleted_by marker = %q, want %q", deletedBy, newer.ID)
	}
}
