package relaysync

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeProjector struct {
	mu       sync.Mutex
	projects int
	purges   int
}

func (f *fakeProjector) ProjectRelaySyncOutbox(context.Context, uint64, int) (int, error) {
	f.mu.Lock()
	f.projects++
	f.mu.Unlock()
	return 1, nil
}
func (f *fakeProjector) PurgeRelaySyncOutbox(context.Context, time.Time, int) (int64, error) {
	f.mu.Lock()
	f.purges++
	f.mu.Unlock()
	return 0, nil
}
func (f *fakeProjector) PurgeRelaySyncJournal(context.Context, time.Time, int) (int64, error) {
	f.mu.Lock()
	f.purges++
	f.mu.Unlock()
	return 0, nil
}

func TestRuntimeProjectsImmediatelyAndCloses(t *testing.T) {
	f := &fakeProjector{}
	r, err := Start(context.Background(), Options{Projector: f, Epoch: 1, Interval: time.Hour, BatchSize: 10, RetentionBatchSize: 10, OutboxRetention: time.Hour, JournalRetention: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for r.Status().Cycles == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if r.Status().Cycles == 0 {
		t.Fatal("projector did not run initial cycle")
	}
	if err := r.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeRejectsInvalidOptions(t *testing.T) {
	_, err := Start(context.Background(), Options{})
	if err == nil || errors.Is(err, context.Canceled) {
		t.Fatalf("Start() error = %v", err)
	}
}
