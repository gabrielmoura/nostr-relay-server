package db

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
)

const getAllEvents = `-- name: GetAllEvents :many
SELECT id, pubkey, created_at, kind, tags, content, sig
FROM event
`

const streamAllEventsCursor = `-- name: StreamAllEvents :many
SELECT id, pubkey, created_at, kind, tags, content, sig
FROM event
WHERE (created_at > $1 OR (created_at = $1 AND id > $2))
ORDER BY created_at, id
LIMIT $3
`

func (q *Queries) GetAllEvents(ctx context.Context) ([]*nostr.Event, error) {
	rows, err := q.db.Query(ctx, getAllEvents)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*nostr.Event, 0, 128)
	for rows.Next() {
		evt, scanErr := scanNostrEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, evt)
	}

	return events, rows.Err()
}

func (q *Queries) StreamAllEvents(ctx context.Context, pageSize int) <-chan *[]nostr.Event {
	out := make(chan *[]nostr.Event)
	go func() {
		defer close(out)
		var lastCreatedAt int64
		var lastID string

		for {
			rows, err := q.db.Query(ctx, streamAllEventsCursor, lastCreatedAt, lastID, pageSize)
			if err != nil {
				return
			}

			batch := make([]nostr.Event, 0, pageSize)
			for rows.Next() {
				evt, scanErr := scanNostrEvent(rows)
				if scanErr != nil {
					rows.Close()
					return
				}
				lastCreatedAt = int64(evt.CreatedAt)
				lastID = evt.ID
				batch = append(batch, *evt)
			}
			rows.Close()

			if len(batch) > 0 {
				out <- &batch
			}
			if len(batch) < pageSize {
				break
			}
		}
	}()
	return out
}
