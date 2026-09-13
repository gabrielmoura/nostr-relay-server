package db

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
)

const resetNIP29Projection = `
DELETE FROM nip29_groups WHERE relay = $1
`

func (q *Queries) ResetNIP29Projection(ctx context.Context, relay string) error {
	_, err := q.db.Exec(ctx, resetNIP29Projection, relay)
	return err
}

const listNIP29ModerationEvents = `
SELECT id, pubkey, created_at, kind, tags, content, sig
FROM event
WHERE kind IN (9000, 9001, 9002, 9005, 9007, 9008, 9009, 9010)
ORDER BY created_at, id
`

func (q *Queries) ListNIP29ModerationEvents(ctx context.Context) ([]*nostr.Event, error) {
	rows, err := q.db.Query(ctx, listNIP29ModerationEvents)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*nostr.Event, 0, 32)
	for rows.Next() {
		evt, err := scanNostrEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, rows.Err()
}

const listNIP29GroupIDs = `
SELECT group_id FROM nip29_groups WHERE relay = $1 ORDER BY group_id
`

func (q *Queries) ListNIP29GroupIDs(ctx context.Context, relay string) ([]string, error) {
	rows, err := q.db.Query(ctx, listNIP29GroupIDs, relay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groupIDs := make([]string, 0, 16)
	for rows.Next() {
		var groupID string
		if err := rows.Scan(&groupID); err != nil {
			return nil, err
		}
		groupIDs = append(groupIDs, groupID)
	}
	return groupIDs, rows.Err()
}
