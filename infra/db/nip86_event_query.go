package db

import "context"

const upsertBannedEvent = `
INSERT INTO nip86_banned_events (event_id, reason, created_by, created_at, updated_at)
VALUES ($1::varchar, $2::text, $3::varchar, NOW(), NOW())
ON CONFLICT (event_id) DO UPDATE SET
    reason = EXCLUDED.reason,
    created_by = EXCLUDED.created_by,
    updated_at = NOW();
`

const deleteBannedEvent = `DELETE FROM nip86_banned_events WHERE event_id = $1::varchar`

const getBannedEvent = `
SELECT event_id, COALESCE(reason, ''), created_by, created_at, updated_at
FROM nip86_banned_events
WHERE event_id = $1::varchar
LIMIT 1
`

const listBannedEvents = `
SELECT event_id, COALESCE(reason, ''), created_by, created_at, updated_at
FROM nip86_banned_events
ORDER BY updated_at DESC, event_id ASC
`

func (q *Queries) UpsertBannedEvent(ctx context.Context, eventID, reason, createdBy string) error {
	_, err := q.db.Exec(ctx, upsertBannedEvent, eventID, reason, createdBy)
	return err
}

func (q *Queries) DeleteBannedEvent(ctx context.Context, eventID string) error {
	_, err := q.db.Exec(ctx, deleteBannedEvent, eventID)
	return err
}

func (q *Queries) GetBannedEvent(ctx context.Context, eventID string) (NIP86EventRecord, bool, error) {
	var item NIP86EventRecord
	err := q.db.QueryRow(ctx, getBannedEvent, eventID).Scan(
		&item.EventID,
		&item.Reason,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if isNotFound(err) {
		return NIP86EventRecord{}, false, nil
	}
	if err != nil {
		return NIP86EventRecord{}, false, err
	}
	return item, true, nil
}

func (q *Queries) ListBannedEvents(ctx context.Context) ([]NIP86EventRecord, error) {
	rows, err := q.db.Query(ctx, listBannedEvents)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]NIP86EventRecord, 0)
	for rows.Next() {
		var item NIP86EventRecord
		if err := rows.Scan(&item.EventID, &item.Reason, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
