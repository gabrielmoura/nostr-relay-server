package db

import "context"

const upsertContentSpamFlag = `
INSERT INTO content_spam_flags (event_id)
VALUES ($1)
ON CONFLICT (event_id) DO NOTHING`

const hasContentSpamFlag = `SELECT EXISTS (SELECT 1 FROM content_spam_flags WHERE event_id = $1)`

func (q *Queries) UpsertContentSpamFlag(ctx context.Context, eventID string) error {
	_, err := q.db.Exec(ctx, upsertContentSpamFlag, eventID)
	return err
}

func (q *Queries) HasContentSpamFlag(ctx context.Context, eventID string) (bool, error) {
	var flagged bool
	err := q.db.QueryRow(ctx, hasContentSpamFlag, eventID).Scan(&flagged)
	return flagged, err
}
