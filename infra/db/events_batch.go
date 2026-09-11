package db

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

// maxUnnestBatchSize bounds each set-based INSERT. It is intentionally aligned
// with the default ingestion batch size and can be changed independently from
// the ingestion configuration when database limits require it.
const maxUnnestBatchSize = 1000

var (
	errNilBatchEvent          = errors.New("event is nil")
	errEventIntegerOutOfRange = errors.New("event created_at or kind exceeds PostgreSQL integer range")
)

const insertEventBatchUnnest = `-- name: InsertEventBatchUnnest :many
INSERT INTO event (id, pubkey, created_at, kind, tags, content, sig)
SELECT input.id, input.pubkey, input.created_at, input.kind, input.tags::jsonb, input.content, input.sig
FROM unnest(
    $1::text[], $2::text[], $3::int[], $4::int[], $5::text[], $6::text[], $7::text[]
) AS input(id, pubkey, created_at, kind, tags, content, sig)
ON CONFLICT (id) DO NOTHING
RETURNING id
`

// InsertEventBatch persists already-validated events with one set-based INSERT
// per chunk. The event table schema stores tags as JSONB, so tags are marshaled
// to JSON text and cast by PostgreSQL inside the statement. Conflicts on id are
// duplicates, not errors. A PostgreSQL row error is isolated by splitting its
// chunk until the bad event can be logged and skipped; the remaining events are
// retried in smaller chunks.
func (q *Queries) InsertEventBatch(ctx context.Context, events []*nostr.Event) (BatchInsertResult, error) {
	result := BatchInsertResult{
		InsertedIDs:      []string{},
		RejectedEventIDs: []string{},
	}
	if len(events) == 0 {
		return result, errors.New("no events to insert")
	}

	for start := 0; start < len(events); start += maxUnnestBatchSize {
		end := min(start+maxUnnestBatchSize, len(events))
		chunkResult, err := q.insertEventBatchRecovering(ctx, events[start:end], start)
		if err != nil {
			return result, err
		}

		result.Inserted += chunkResult.Inserted
		result.Duplicates += chunkResult.Duplicates
		result.InsertedIDs = append(result.InsertedIDs, chunkResult.InsertedIDs...)
		result.RejectedEventIDs = append(result.RejectedEventIDs, chunkResult.RejectedEventIDs...)
	}

	if result.Inserted > 0 {
		_ = cache.InvalidateQueryCache()
	}

	return result, nil
}

func (q *Queries) insertEventBatchRecovering(ctx context.Context, events []*nostr.Event, start int) (BatchInsertResult, error) {
	result, err := q.insertEventBatchChunk(ctx, events)
	if err == nil {
		return result, nil
	}
	if !isRowIsolationError(err) {
		return BatchInsertResult{}, newBatchInsertError(start, events[0], err)
	}
	if len(events) == 1 {
		batchErr := newBatchInsertError(start, events[0], err)
		logBatchEventRejection(batchErr)

		return BatchInsertResult{
			InsertedIDs:      []string{},
			RejectedEventIDs: []string{batchEventID(events[0])},
		}, nil
	}

	middle := len(events) / 2
	left, err := q.insertEventBatchRecovering(ctx, events[:middle], start)
	if err != nil {
		return BatchInsertResult{}, err
	}
	right, err := q.insertEventBatchRecovering(ctx, events[middle:], start+middle)
	if err != nil {
		return BatchInsertResult{}, err
	}

	return mergeBatchInsertResults(left, right), nil
}

func isRowIsolationError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) || errors.Is(err, errNilBatchEvent) || errors.Is(err, errEventIntegerOutOfRange)
}

func batchEventID(event *nostr.Event) string {
	if event == nil {
		return ""
	}
	return event.ID
}

func logBatchEventRejection(err error) {
	if log.Logger == nil {
		return
	}

	batchErr, ok := err.(*BatchInsertError)
	if !ok {
		log.Logger.Error("batch event rejected", zap.Error(err))
		return
	}

	fields := []zap.Field{
		zap.Int("event_index", batchErr.Index),
		zap.String("event_id", batchErr.EventID),
		zap.String("pubkey", batchErr.Pubkey),
		zap.Int("kind", batchErr.Kind),
		zap.Int("content_length", batchErr.ContentLength),
		zap.Int("tags_count", batchErr.TagsCount),
		zap.Int("raw_size", batchErr.RawSize),
		zap.Error(batchErr.Err),
	}
	var pgErr *pgconn.PgError
	if errors.As(batchErr.Err, &pgErr) {
		fields = append(fields,
			zap.String("sqlstate", pgErr.Code),
			zap.String("constraint", pgErr.ConstraintName),
		)
	}
	log.Logger.Warn("batch event rejected", fields...)
}

func mergeBatchInsertResults(left, right BatchInsertResult) BatchInsertResult {
	return BatchInsertResult{
		Inserted:         left.Inserted + right.Inserted,
		Duplicates:       left.Duplicates + right.Duplicates,
		InsertedIDs:      append(left.InsertedIDs, right.InsertedIDs...),
		RejectedEventIDs: append(left.RejectedEventIDs, right.RejectedEventIDs...),
	}
}

func (q *Queries) insertEventBatchChunk(ctx context.Context, events []*nostr.Event) (BatchInsertResult, error) {
	ids := make([]string, len(events))
	pubkeys := make([]string, len(events))
	createdAts := make([]int32, len(events))
	kinds := make([]int32, len(events))
	tags := make([]string, len(events))
	contents := make([]string, len(events))
	sigs := make([]string, len(events))

	for i, event := range events {
		if event == nil {
			return BatchInsertResult{}, errNilBatchEvent
		}
		if uint64(event.CreatedAt) > math.MaxInt32 || event.Kind < math.MinInt32 || event.Kind > math.MaxInt32 {
			return BatchInsertResult{}, errEventIntegerOutOfRange
		}

		tagsJSON, err := json.Marshal(event.Tags)
		if err != nil {
			return BatchInsertResult{}, fmt.Errorf("marshal tags for event %s: %w", event.ID, err)
		}

		ids[i] = event.ID
		pubkeys[i] = event.PubKey
		createdAts[i] = int32(event.CreatedAt)
		kinds[i] = int32(event.Kind)
		tags[i] = string(tagsJSON)
		contents[i] = event.Content
		sigs[i] = event.Sig
	}

	rows, err := q.db.Query(ctx, insertEventBatchUnnest, ids, pubkeys, createdAts, kinds, tags, contents, sigs)
	if err != nil {
		return BatchInsertResult{}, err
	}
	defer rows.Close()

	insertedIDs := make([]string, 0, len(events))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return BatchInsertResult{}, err
		}
		insertedIDs = append(insertedIDs, id)
	}
	if err := rows.Err(); err != nil {
		return BatchInsertResult{}, err
	}

	return BatchInsertResult{
		Inserted:    len(insertedIDs),
		Duplicates:  len(events) - len(insertedIDs),
		InsertedIDs: insertedIDs,
	}, nil
}
