package db

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/jackc/pgx/v5"
	"github.com/nbd-wtf/go-nostr"
)

type transactionStarter interface {
	Begin(context.Context) (pgx.Tx, error)
}

const createEventCopyStaging = `
CREATE TEMP TABLE event_copy_staging (
    id text,
    pubkey text,
    created_at integer,
    kind integer,
    tags jsonb,
    content text,
    sig text
) ON COMMIT DROP
`

const copyStagingIntoEvent = `
INSERT INTO event (id, pubkey, created_at, kind, tags, content, sig)
SELECT id, pubkey, created_at, kind, tags, content, sig
FROM event_copy_staging
ON CONFLICT (id) DO NOTHING
`

// MigrateEventsViaCopy transfers a large, one-off event collection from a
// relay or import file. It is not for live ingestion: it uses COPY into a
// transaction-scoped staging table, then one set-based conflict-aware insert.
func (q *Queries) MigrateEventsViaCopy(ctx context.Context, events []*nostr.Event) (BatchInsertResult, error) {
	result := BatchInsertResult{
		InsertedIDs: []string{},
	}
	if len(events) == 0 {
		return result, errors.New("no events to migrate")
	}

	starter, ok := q.db.(transactionStarter)
	if !ok {
		return result, errors.New("database does not support transactions required for COPY migration")
	}

	tx, err := starter.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin COPY migration: %w", err)
	}
	defer func() {
		_ = tx.Rollback(context.WithoutCancel(ctx))
	}()

	if _, err := tx.Exec(ctx, createEventCopyStaging); err != nil {
		return result, fmt.Errorf("create COPY staging table: %w", err)
	}

	copied, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"event_copy_staging"},
		[]string{"id", "pubkey", "created_at", "kind", "tags", "content", "sig"},
		pgx.CopyFromSlice(len(events), func(i int) ([]any, error) {
			event := events[i]
			if event == nil {
				return nil, fmt.Errorf("event %d is nil", i)
			}
			if uint64(event.CreatedAt) > math.MaxInt32 || event.Kind < math.MinInt32 || event.Kind > math.MaxInt32 {
				return nil, fmt.Errorf("event %d has an integer value outside the event schema range", i)
			}

			tagsJSON, err := json.Marshal(event.Tags)
			if err != nil {
				return nil, fmt.Errorf("marshal tags for event %s: %w", event.ID, err)
			}

			return []any{
				event.ID,
				event.PubKey,
				int32(event.CreatedAt),
				int32(event.Kind),
				json.RawMessage(tagsJSON),
				event.Content,
				event.Sig,
			}, nil
		}),
	)
	if err != nil {
		return result, fmt.Errorf("COPY events into staging table: %w", err)
	}

	tag, err := tx.Exec(ctx, copyStagingIntoEvent)
	if err != nil {
		return result, fmt.Errorf("insert COPY staging events: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit COPY migration: %w", err)
	}

	result.Inserted = int(tag.RowsAffected())
	result.Duplicates = int(copied) - result.Inserted
	if result.Inserted > 0 {
		_ = cache.InvalidateQueryCache()
	}

	return result, nil
}
