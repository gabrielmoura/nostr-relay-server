package db

import (
	"context"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/nbd-wtf/go-nostr"
)

const getOldEvents = `-- name: GetOldEvents :many
SELECT id, pubkey, created_at, kind, tags, content, sig
FROM event
WHERE created_at < $1::timestamptz
`

const deleteEventsOlderThan = `
WITH doomed AS (
    SELECT id FROM event WHERE created_at < $1::int ORDER BY created_at ASC LIMIT $2::int
)
DELETE FROM event e USING doomed d WHERE e.id = d.id
`

const deleteExpiredNIP40Events = `
WITH doomed AS (
    SELECT e.id
    FROM event e
    WHERE EXISTS (
        SELECT 1 FROM jsonb_array_elements(e.tags) tag
        WHERE jsonb_typeof(tag) = 'array' AND jsonb_array_length(tag) >= 2 AND tag->>0 = 'expiration'
          AND (tag->>1) ~ '^[0-9]+$' AND (tag->>1)::bigint <= $1::bigint
    )
    ORDER BY e.created_at ASC
    LIMIT $2::int
)
DELETE FROM event e USING doomed d WHERE e.id = d.id
`

func (q *Queries) DeleteEventsOlderThan(ctx context.Context, beforeUnix int64, batchSize int) (int64, error) {
	res, err := q.db.Exec(ctx, deleteEventsOlderThan, beforeUnix, batchSize)
	if err != nil {
		return 0, err
	}
	if res.RowsAffected() > 0 {
		_ = cache.InvalidateQueryCache()
	}
	return res.RowsAffected(), nil
}

func (q *Queries) DeleteExpiredNIP40Events(ctx context.Context, nowUnix int64, batchSize int) (int64, error) {
	res, err := q.db.Exec(ctx, deleteExpiredNIP40Events, nowUnix, batchSize)
	if err != nil {
		return 0, err
	}
	if res.RowsAffected() > 0 {
		_ = cache.InvalidateQueryCache()
	}
	return res.RowsAffected(), nil
}

func (q *Queries) GetOldEvents(ctx context.Context, before time.Time) ([]*nostr.Event, error) {
	rows, err := q.db.Query(ctx, getOldEvents, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*nostr.Event, 0, 64)
	for rows.Next() {
		evt, scanErr := scanNostrEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, evt)
	}

	return events, rows.Err()
}
