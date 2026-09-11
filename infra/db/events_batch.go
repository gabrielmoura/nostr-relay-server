package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

// maxUnnestBatchSize bounds each set-based INSERT. It is intentionally aligned
// with the default ingestion batch size and can be changed independently from
// the ingestion configuration when database limits require it.
const maxUnnestBatchSize = 1000

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
// duplicates, not errors; another constraint failure aborts the current chunk.
func (q *Queries) InsertEventBatch(ctx context.Context, events []*nostr.Event) (BatchInsertResult, error) {
	result := BatchInsertResult{
		InsertedIDs: []string{},
	}
	if len(events) == 0 {
		return result, errors.New("no events to insert")
	}

	for start := 0; start < len(events); start += maxUnnestBatchSize {
		end := min(start+maxUnnestBatchSize, len(events))
		chunkResult, err := q.insertEventBatchChunk(ctx, events[start:end])
		if err != nil {
			return result, newBatchInsertError(start, events[start], err)
		}

		result.Inserted += chunkResult.Inserted
		result.Duplicates += chunkResult.Duplicates
		result.InsertedIDs = append(result.InsertedIDs, chunkResult.InsertedIDs...)
	}

	if result.Inserted > 0 {
		_ = cache.InvalidateQueryCache()
	}

	return result, nil
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
			return BatchInsertResult{}, errors.New("event is nil")
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
