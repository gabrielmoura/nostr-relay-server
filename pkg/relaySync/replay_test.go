package relaysync

import (
	"context"
	"testing"
)

func TestReplayServiceAdvancesCheckpointAfterTerminalDecision(t *testing.T) {
	journal := NewMemoryJournalProvider(2, 10)
	var id EventID
	id[0] = 1
	journal.Append(JournalChange{Type: ChangeDelete, ID: id})
	storage := NewMemoryStorage(nil, DefaultLimits())
	checkpoints := NewMemoryCheckpointStore()
	service, err := NewReplayService(journal, storage, checkpoints)
	if err != nil {
		t.Fatal(err)
	}
	key := CheckpointKey{PeerID: "peer", Scope: Scope{Name: "public", Version: 1}}
	cursor, err := service.Replay(context.Background(), key, key.Scope, Cursor{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if cursor.Epoch != 2 || cursor.Sequence != 1 {
		t.Fatalf("cursor = %+v", cursor)
	}
	checkpoint, ok, err := checkpoints.LoadCheckpoint(context.Background(), key)
	if err != nil || !ok || checkpoint.Cursor != cursor {
		t.Fatalf("checkpoint = %+v %v %v", checkpoint, ok, err)
	}
}
