package db

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/jackc/pgx/v5"
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

// InsertDMImportBatch atomically stores an import batch and enforces the
// replaceable/addressable winner for every imported key. A serializable
// transaction plus deterministic advisory locks prevents two import workers
// from independently accepting incompatible versions of the same key.
func (q *Queries) InsertDMImportBatch(ctx context.Context, events []*nostr.Event) (BatchInsertResult, error) {
	const maxSerializationAttempts = 3
	var lastErr error
	for attempt := 0; attempt < maxSerializationAttempts; attempt++ {
		result, err := q.insertDMImportBatchOnce(ctx, events)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !isSerializationFailure(err) || ctx.Err() != nil {
			return BatchInsertResult{}, err
		}
	}
	return BatchInsertResult{}, fmt.Errorf("DM import batch serialization retry limit reached: %w", lastErr)
}

func (q *Queries) insertDMImportBatchOnce(ctx context.Context, events []*nostr.Event) (BatchInsertResult, error) {
	starter, ok := q.db.(transactionStarter)
	if !ok {
		return BatchInsertResult{}, errors.New("DM import batch requires transaction-capable database")
	}
	tx, err := starter.Begin(ctx)
	if err != nil {
		return BatchInsertResult{}, fmt.Errorf("begin DM import batch transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE"); err != nil {
		return BatchInsertResult{}, fmt.Errorf("set DM import batch isolation: %w", err)
	}

	keys := replaceableKeys(events)
	for _, key := range keys {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", key.lock); err != nil {
			return BatchInsertResult{}, fmt.Errorf("lock DM import replacement key: %w", err)
		}
	}

	txQueries := q.WithTx(tx)
	result, err := txQueries.InsertEventBatch(ctx, events)
	if err != nil {
		return BatchInsertResult{}, err
	}

	pruned, err := pruneSupersededImportEvents(ctx, tx, keys)
	if err != nil {
		return BatchInsertResult{}, err
	}
	result = removePrunedInsertions(result, pruned)
	if err := tx.Commit(ctx); err != nil {
		return BatchInsertResult{}, fmt.Errorf("commit DM import batch transaction: %w", err)
	}
	if result.Inserted > 0 {
		_ = cache.InvalidateQueryCache()
	}
	return result, nil
}

type replaceableKey struct {
	pubkey      string
	kind        int
	d           string
	addressable bool
	lock        string
}

func replaceableKeys(events []*nostr.Event) []replaceableKey {
	unique := make(map[string]replaceableKey)
	for _, evt := range events {
		if evt == nil {
			continue
		}
		key, ok := replacementKey(evt)
		if !ok {
			continue
		}
		unique[key.lock] = key
	}
	keys := make([]replaceableKey, 0, len(unique))
	for _, key := range unique {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].lock < keys[j].lock })
	return keys
}

func replacementKey(evt *nostr.Event) (replaceableKey, bool) {
	if nostr.IsReplaceableKind(evt.Kind) {
		return replaceableKey{pubkey: evt.PubKey, kind: evt.Kind, lock: fmt.Sprintf("r:%s:%d", evt.PubKey, evt.Kind)}, true
	}
	if nostr.IsAddressableKind(evt.Kind) {
		d := evt.Tags.GetFirst([]string{"d", ""})
		if d == nil {
			return replaceableKey{}, false
		}
		value := d.Value()
		return replaceableKey{pubkey: evt.PubKey, kind: evt.Kind, d: value, addressable: true, lock: fmt.Sprintf("a:%s:%d:%s", evt.PubKey, evt.Kind, value)}, true
	}
	return replaceableKey{}, false
}

func pruneSupersededImportEvents(ctx context.Context, tx pgx.Tx, keys []replaceableKey) (map[string]struct{}, error) {
	pruned := make(map[string]struct{})
	for _, key := range keys {
		query := `
DELETE FROM event AS older
WHERE older.id IN (
    SELECT id FROM (
        SELECT id, row_number() OVER (ORDER BY created_at DESC, id ASC) AS position
        FROM event
        WHERE pubkey = $1 AND kind = $2
    ) AS ranked
    WHERE position > 1
)
RETURNING older.id`
		args := []any{key.pubkey, key.kind}
		if key.addressable {
			query = `
DELETE FROM event AS older
WHERE older.id IN (
    SELECT id FROM (
        SELECT id, row_number() OVER (ORDER BY created_at DESC, id ASC) AS position
        FROM event
        WHERE pubkey = $1 AND kind = $2
          AND EXISTS (
              SELECT 1 FROM jsonb_array_elements(tags) AS tag
              WHERE tag ->> 0 = 'd' AND tag ->> 1 = $3
          )
    ) AS ranked
    WHERE position > 1
)
RETURNING older.id`
			args = append(args, key.d)
		}
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("prune superseded DM import events: %w", err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			pruned[id] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return pruned, nil
}

func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "40001"
}

func removePrunedInsertions(result BatchInsertResult, pruned map[string]struct{}) BatchInsertResult {
	if len(pruned) == 0 {
		return result
	}
	retained := make([]string, 0, len(result.InsertedIDs))
	for _, id := range result.InsertedIDs {
		if _, removed := pruned[id]; removed {
			result.Inserted--
			result.Duplicates++
			continue
		}
		retained = append(retained, id)
	}
	result.InsertedIDs = retained
	return result
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
