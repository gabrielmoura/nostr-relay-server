package groups

import (
	"context"
	"testing"

	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
)

func TestFinalizeDeletedGroupStateClearsStalePublicCacheBeforeEmission(t *testing.T) {
	stalePublic := &dbstore.NIP29Group{GroupID: "group", Private: false}
	steps := make([]string, 0, 3)
	err := finalizeDeletedGroupState(context.Background(), "group", func(groupID string) {
		if groupID != "group" {
			t.Fatalf("invalidated group = %q", groupID)
		}
		stalePublic = nil
		steps = append(steps, "invalidate")
	}, func(context.Context, string) error {
		steps = append(steps, "roles")
		return nil
	}, func(context.Context, string) error {
		if stalePublic != nil {
			t.Fatalf("state emission observed stale public cache: %#v", stalePublic)
		}
		steps = append(steps, "emit")
		return nil
	})
	if err != nil {
		t.Fatalf("finalizeDeletedGroupState() error = %v", err)
	}
	want := []string{"invalidate", "roles", "emit"}
	for i := range want {
		if len(steps) != len(want) || steps[i] != want[i] {
			t.Fatalf("steps = %#v, want %#v", steps, want)
		}
	}
}
