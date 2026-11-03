package sync

import (
	"context"
	"errors"
	"testing"

	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
)

func TestQueueJobHonorsCanceledContext(t *testing.T) {
	registry := jobcore.NewRegistry()
	if err := RegisterQueueHandlers(registry); err != nil {
		t.Fatal(err)
	}
	handler, ok := registry.Lookup(queueJobName)
	if !ok {
		t.Fatal("sync handler not registered")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := handler.Handle(ctx, QueueJob{Remote: "wss://relay.example", Direction: "down"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
