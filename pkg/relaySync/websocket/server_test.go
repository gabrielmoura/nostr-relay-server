package websocket

import (
	"testing"

	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
)

func TestDeltaProtoPreservesContiguousCursors(t *testing.T) {
	var id core.EventID
	id[0] = 1
	batch := core.LiveBatch{First: core.Cursor{Epoch: 2, Sequence: 3}, Last: core.Cursor{Epoch: 2, Sequence: 4}, Changes: []core.JournalChange{{Cursor: core.Cursor{Epoch: 2, Sequence: 3}, ID: id, Type: core.ChangeDelete}, {Cursor: core.Cursor{Epoch: 2, Sequence: 4}, ID: id, Type: core.ChangeDelete}}}
	got := deltaProto(batch)
	if got.FirstCursor.Sequence != 3 || got.LastCursor.Sequence != 4 || len(got.Changes) != 2 {
		t.Fatalf("unexpected delta: %v", got)
	}
}
