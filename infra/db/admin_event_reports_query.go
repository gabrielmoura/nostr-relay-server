package db

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
)

const getEventByID = `-- name: GetEventByID :one
SELECT id, pubkey, created_at, kind, tags, content, sig
FROM event
WHERE id = $1::text
LIMIT 1;
`

const getReportsForEventCount = `-- name: GetReportsForEventCount :one
SELECT COUNT(*)
FROM event e
WHERE e.kind = 1984
  AND EXISTS (
    SELECT 1 FROM jsonb_array_elements(e.tags) tag WHERE tag->>0 = 'e' AND tag->>1 = $1::text
  );
`

const getReportsForEvent = `-- name: GetReportsForEvent :many
SELECT e.id, e.pubkey, e.created_at, e.kind, e.tags, e.content, e.sig
FROM event e
WHERE e.kind = 1984
  AND EXISTS (
    SELECT 1 FROM jsonb_array_elements(e.tags) tag WHERE tag->>0 = 'e' AND tag->>1 = $1::text
  )
ORDER BY e.created_at DESC
LIMIT $2 OFFSET $3;
`

func (q *Queries) GetEventByID(ctx context.Context, id string) (*nostr.Event, error) {
	return scanNostrEvent(q.db.QueryRow(ctx, getEventByID, id))
}

func (q *Queries) GetReportsForEvent(ctx context.Context, eventID string, limit int, offset int) ([]*nostr.Event, int64, error) {
	var total int64
	if err := q.db.QueryRow(ctx, getReportsForEventCount, eventID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := q.db.Query(ctx, getReportsForEvent, eventID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]*nostr.Event, 0, limit)
	for rows.Next() {
		evt, scanErr := scanNostrEvent(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, evt)
	}

	return items, total, rows.Err()
}
