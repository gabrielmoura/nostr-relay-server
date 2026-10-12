package relaysync

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fixedClock struct{ value time.Time }

func (c fixedClock) Now() time.Time { return c.value }

func TestMemorySessionStore_CompareAndSwapAndExpiry(t *testing.T) {
	now := time.Unix(100, 0)
	store := NewMemorySessionStore(fixedClock{value: now})
	id := SessionID{1}
	session := SessionRecord{ID: id, State: SessionStateCreated, ExpiresAt: now.Add(time.Minute)}
	if err := store.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	stored, err := store.GetSession(context.Background(), id)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if stored.Version != 1 {
		t.Fatalf("Version = %d, want 1", stored.Version)
	}
	update := SessionUpdate{ID: id, ExpectedVersion: stored.Version, State: SessionStateReconciling, ExpiresAt: now.Add(time.Minute)}
	if err := store.UpdateSession(context.Background(), update); err != nil {
		t.Fatalf("UpdateSession() error = %v", err)
	}
	if err := store.UpdateSession(context.Background(), update); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale UpdateSession() error = %v, want ErrConflict", err)
	}

	expiredStore := NewMemorySessionStore(fixedClock{value: now.Add(time.Minute)})
	if err := expiredStore.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if _, err := expiredStore.GetSession(context.Background(), id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired GetSession() error = %v, want ErrNotFound", err)
	}
}

func TestMemoryCheckpointStore_Monotonic(t *testing.T) {
	store := NewMemoryCheckpointStore()
	checkpoint := Checkpoint{Key: CheckpointKey{PeerID: "peer"}, Cursor: Cursor{Epoch: 1, Sequence: 2}}
	if err := store.AdvanceCheckpoint(context.Background(), checkpoint); err != nil {
		t.Fatalf("AdvanceCheckpoint() error = %v", err)
	}
	checkpoint.Cursor.Sequence = 1
	if err := store.AdvanceCheckpoint(context.Background(), checkpoint); !errors.Is(err, ErrConflict) {
		t.Fatalf("regressing AdvanceCheckpoint() error = %v, want ErrConflict", err)
	}
	checkpoint.Cursor = Cursor{Epoch: 2, Sequence: 1}
	if err := store.AdvanceCheckpoint(context.Background(), checkpoint); !errors.Is(err, ErrConflict) {
		t.Fatalf("epoch-changing AdvanceCheckpoint() error = %v, want ErrConflict", err)
	}
	if err := store.ResetCheckpoint(context.Background(), checkpoint); err != nil {
		t.Fatalf("ResetCheckpoint() error = %v", err)
	}
}
