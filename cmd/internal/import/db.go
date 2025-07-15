package _import

import (
	"context"
	"fmt"
	dbx "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/nbd-wtf/go-nostr"
)

func saveToDatabase(ctx context.Context, store *dbx.Queries, event *nostr.Event) error {
	nCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := store.InsertEvent(nCtx, event); err != nil {
		return fmt.Errorf("failed to save event to event store: %w", err)
	}
	return nil
}

func saveBatchToDatabase(ctx context.Context, store *dbx.Queries, events []*nostr.Event) error {
	nCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := store.InsertEventBatch(nCtx, events); err != nil {
		return fmt.Errorf("failed to save batch of events to event store: %w", err)
	}
	return nil
}
