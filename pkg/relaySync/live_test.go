package relaysync

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestLiveSync_ReplaysAfterDisconnectBeforeAck(t *testing.T) {
	journal := NewMemoryJournalProvider(4, 10)
	first := journal.Append(JournalChange{Type: ChangeDelete, ID: EventID{1}})
	checkpoints := NewMemoryCheckpointStore()
	key := CheckpointKey{PeerID: "peer", Scope: Scope{Name: "public", Version: 1}}

	firstConnection := newLiveSyncForTest(t, journal, checkpoints, key, LiveConfig{})
	if err := firstConnection.Resume(context.Background()); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if _, err := firstConnection.Pump(context.Background(), 10); err != nil {
		t.Fatalf("Pump() error = %v", err)
	}
	if err := firstConnection.SetFlowControl(FlowControl{MaxEvents: 1, MaxBytes: 1024}); err != nil {
		t.Fatalf("SetFlowControl() error = %v", err)
	}
	batch, err := firstConnection.NextBatch()
	if err != nil || batch.Last != first {
		t.Fatalf("NextBatch() = %+v, %v", batch, err)
	}
	if err := firstConnection.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	secondConnection := newLiveSyncForTest(t, journal, checkpoints, key, LiveConfig{})
	if err := secondConnection.Resume(context.Background()); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if read, err := secondConnection.Pump(context.Background(), 10); err != nil || read != 1 {
		t.Fatalf("Pump() = %d, %v; want 1, nil", read, err)
	}
	if err := secondConnection.SetFlowControl(FlowControl{MaxEvents: 1, MaxBytes: 1024}); err != nil {
		t.Fatalf("SetFlowControl() error = %v", err)
	}
	batch, err = secondConnection.NextBatch()
	if err != nil || batch.Last != first {
		t.Fatalf("replayed NextBatch() = %+v, %v", batch, err)
	}
}

func TestLiveSync_DoesNotReplayAfterDurableAck(t *testing.T) {
	journal := NewMemoryJournalProvider(4, 10)
	cursor := journal.Append(JournalChange{Type: ChangeDelete, ID: EventID{1}})
	checkpoints := NewMemoryCheckpointStore()
	key := CheckpointKey{PeerID: "peer", Scope: Scope{Name: "public", Version: 1}}
	connection := newLiveSyncForTest(t, journal, checkpoints, key, LiveConfig{})
	if err := connection.Resume(context.Background()); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if _, err := connection.Pump(context.Background(), 10); err != nil {
		t.Fatalf("Pump() error = %v", err)
	}
	if err := connection.SetFlowControl(FlowControl{MaxEvents: 1, MaxBytes: 1024}); err != nil {
		t.Fatalf("SetFlowControl() error = %v", err)
	}
	if _, err := connection.NextBatch(); err != nil {
		t.Fatalf("NextBatch() error = %v", err)
	}
	if err := connection.Ack(context.Background(), cursor, "decision-1"); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}
	_ = connection.Close()

	reconnected := newLiveSyncForTest(t, journal, checkpoints, key, LiveConfig{})
	if err := reconnected.Resume(context.Background()); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if read, err := reconnected.Pump(context.Background(), 10); err != nil || read != 0 {
		t.Fatalf("Pump() = %d, %v; want 0, nil", read, err)
	}
	if err := reconnected.SetFlowControl(FlowControl{MaxEvents: 1, MaxBytes: 1024}); err != nil {
		t.Fatalf("SetFlowControl() error = %v", err)
	}
	if _, err := reconnected.NextBatch(); !errors.Is(err, io.EOF) {
		t.Fatalf("NextBatch() error = %v, want EOF", err)
	}
}

func TestLiveSync_BackpressureBoundsRetainedChanges(t *testing.T) {
	journal := NewMemoryJournalProvider(2, 10)
	first := journal.Append(JournalChange{Type: ChangeDelete, ID: EventID{1}})
	second := journal.Append(JournalChange{Type: ChangeDelete, ID: EventID{2}})
	checkpoints := NewMemoryCheckpointStore()
	key := CheckpointKey{PeerID: "peer", Scope: Scope{Name: "public", Version: 1}}
	connection := newLiveSyncForTest(t, journal, checkpoints, key, LiveConfig{
		QueueEvents: 1,
		QueueBytes:  1024,
		BatchEvents: 1,
		BatchBytes:  1024,
	})
	if err := connection.Resume(context.Background()); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if read, err := connection.Pump(context.Background(), 10); err != nil || read != 1 {
		t.Fatalf("first Pump() = %d, %v; want 1, nil", read, err)
	}
	if read, err := connection.Pump(context.Background(), 10); err != nil || read != 0 {
		t.Fatalf("full Pump() = %d, %v; want 0, nil", read, err)
	}
	if err := connection.SetFlowControl(FlowControl{MaxEvents: 1, MaxBytes: 1024}); err != nil {
		t.Fatalf("SetFlowControl() error = %v", err)
	}
	batch, err := connection.NextBatch()
	if err != nil || batch.Last != first {
		t.Fatalf("NextBatch() = %+v, %v", batch, err)
	}
	if err := connection.Ack(context.Background(), first, "one"); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}
	if read, err := connection.Pump(context.Background(), 10); err != nil || read != 1 {
		t.Fatalf("second Pump() = %d, %v; want 1, nil", read, err)
	}
	if err := connection.SetFlowControl(FlowControl{MaxEvents: 1, MaxBytes: 1024}); err != nil {
		t.Fatalf("SetFlowControl() error = %v", err)
	}
	batch, err = connection.NextBatch()
	if err != nil || batch.Last != second {
		t.Fatalf("second NextBatch() = %+v, %v", batch, err)
	}
}

func TestLiveSync_RequiresReconciliationForJournalGap(t *testing.T) {
	journal := NewMemoryJournalProvider(2, 1)
	journal.Append(JournalChange{Type: ChangeDelete, ID: EventID{1}})
	journal.Append(JournalChange{Type: ChangeDelete, ID: EventID{2}})
	connection := newLiveSyncForTest(t, journal, NewMemoryCheckpointStore(), CheckpointKey{PeerID: "peer"}, LiveConfig{})
	if err := connection.Start(context.Background(), Cursor{Epoch: 2, Sequence: 0}); !errors.Is(err, ErrReconcileRequired) {
		t.Fatalf("Start() error = %v, want ErrReconcileRequired", err)
	}
	state, reason := connection.State()
	if state != LiveStateReconcileRequired || reason != ReconcileReasonGap {
		t.Fatalf("State() = %v, %q", state, reason)
	}
}

func TestLiveSync_HeartbeatTimeoutClosesOnlyLiveConnection(t *testing.T) {
	now := time.Unix(100, 0)
	journal := NewMemoryJournalProvider(1, 10)
	connection := newLiveSyncForTest(t, journal, NewMemoryCheckpointStore(), CheckpointKey{PeerID: "peer"}, LiveConfig{
		HeartbeatInterval: time.Second,
		HeartbeatTimeout:  2 * time.Second,
		Clock:             fixedClock{value: now},
	})
	if err := connection.Start(context.Background(), Cursor{}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	heartbeat, due, err := connection.HeartbeatDue(now.Add(time.Second))
	if err != nil || !due || heartbeat.Nonce != 1 {
		t.Fatalf("HeartbeatDue() = %+v, %v, %v", heartbeat, due, err)
	}
	if _, _, err := connection.HeartbeatDue(now.Add(3 * time.Second)); !errors.Is(err, ErrHeartbeatTimeout) {
		t.Fatalf("HeartbeatDue() error = %v, want ErrHeartbeatTimeout", err)
	}
	state, _ := connection.State()
	if state != LiveStateClosed {
		t.Fatalf("State() = %v, want closed", state)
	}
}

func newLiveSyncForTest(t *testing.T, journal JournalProvider, checkpoints CheckpointStore, key CheckpointKey, config LiveConfig) *LiveSync {
	t.Helper()
	connection, err := NewLiveSync(journal, checkpoints, key, key.Scope, config)
	if err != nil {
		t.Fatalf("NewLiveSync() error = %v", err)
	}
	return connection
}
