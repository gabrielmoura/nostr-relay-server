package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/jackc/pgx/v5"
	"github.com/nbd-wtf/go-nostr"
)

const deleteAllEventsByPubkey = `-- name: DeleteAllEventsByPubkey :exec
DELETE FROM event WHERE pubkey = $1::text
`

const deleteEvent = `-- name: DeleteEvent :exec
DELETE FROM event WHERE id = $1::text
`

const fakeDeletionEvent = `-- name: FakeDeletionEvent :exec
UPDATE event SET deleted_by = $2::text WHERE id = $1::text
`

const deleteOldsEvents = `-- name: DeleteOldsEvents :exec
DELETE FROM event WHERE pubkey = $1::text AND kind = $2::int AND created_at < (
    SELECT created_at FROM event WHERE pubkey = $1::text
    ORDER BY created_at DESC, id OFFSET 100 LIMIT 1
)
`

const insertEvent = `-- name: InsertEvent :exec
INSERT INTO event (id, pubkey, created_at, kind, tags, content, sig)
VALUES ($1::text, $2::text, $3::int, $4::int, $5::jsonb, $6::text, $7::text)
ON CONFLICT (id) DO NOTHING
`

type DeleteOldsEventsParams struct {
	Pubkey string `db:"pubkey" json:"pubkey"`
	Kind   int32  `db:"kind" json:"kind"`
}

type InsertEventParams struct {
	ID        string `db:"id" json:"id"`
	Pubkey    string `db:"pubkey" json:"pubkey"`
	CreatedAt int32  `db:"created_at" json:"created_at"`
	Kind      int32  `db:"kind" json:"kind"`
	Tags      []byte `db:"tags" json:"tags"`
	Content   string `db:"content" json:"content"`
	Sig       string `db:"sig" json:"sig"`
}

func (q *Queries) DeleteAllEventsByPubkey(ctx context.Context, pubkey string) error {
	_, err := q.db.Exec(ctx, deleteAllEventsByPubkey, pubkey)
	if err != nil {
		return fmt.Errorf("failed to delete events for pubkey %s: %w", pubkey, err)
	}
	_ = cache.InvalidateQueryCache()
	return nil
}

func (q *Queries) DeleteEvent(ctx context.Context, id, reasonID string) error {
	query := deleteEvent
	args := []any{id}
	if config.Cfg.Relay.FakeDeletion {
		query = fakeDeletionEvent
		args = []any{id, reasonID}
	}

	_, err := q.db.Exec(ctx, query, args...)
	if err == nil {
		_ = cache.InvalidateQueryCache()
	}
	return err
}

func (q *Queries) DeleteOldsEvents(ctx context.Context, arg DeleteOldsEventsParams) error {
	_, err := q.db.Exec(ctx, deleteOldsEvents, arg.Pubkey, arg.Kind)
	return err
}

func (q *Queries) InsertEvent(ctx context.Context, arg *nostr.Event) error {
	res, err := q.db.Exec(ctx, insertEvent, arg.ID, arg.PubKey, arg.CreatedAt, arg.Kind, arg.Tags, arg.Content, arg.Sig)
	if res.RowsAffected() == 0 {
		return ErrDupEvent
	}
	if err == nil {
		_ = cache.InvalidateQueryCache()
	}
	return err
}

func (q *Queries) InsertEventBatch(ctx context.Context, arg []*nostr.Event) error {
	if len(arg) == 0 {
		return errors.New("no events to insert")
	}

	batch := pgx.Batch{}
	for _, evt := range arg {
		batch.Queue(insertEvent, evt.ID, evt.PubKey, evt.CreatedAt, evt.Kind, evt.Tags, evt.Content, evt.Sig)
	}

	results := q.db.SendBatch(ctx, &batch)
	defer results.Close()
	for i := range arg {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("failed to insert event %d: %w", i, err)
		}
	}

	_ = cache.InvalidateQueryCache()
	return nil
}
