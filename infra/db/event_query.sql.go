package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/db/helper"
	"github.com/jackc/pgx/v5"
	"github.com/nbd-wtf/go-nostr"
	"time"
)

const deleteAllEventsByPubkey = `-- name: DeleteAllEventsByPubkey :exec
DELETE FROM event WHERE pubkey = $1::text
`

// DeleteAllEventsByPubkey deletes all events from the database for a given pubkey.
func (q *Queries) DeleteAllEventsByPubkey(ctx context.Context, pubkey string) error {
	_, err := q.db.Exec(ctx, deleteAllEventsByPubkey, pubkey)
	if err != nil {
		return fmt.Errorf("failed to delete events for pubkey %s: %w", pubkey, err)
	}
	return nil
}

const deleteEvent = `-- name: DeleteEvent :exec
DELETE FROM event WHERE id = $1::text
`
const fakeDeletionEvent = `-- name: FakeDeletionEvent :exec
UPDATE event SET deleted_by = $2::text WHERE id = $1::text
`

// DeleteEvent deletes an event from the database by its ID.
// If the relay is configured to use fake deletion, it will set the deleted_by field instead.
func (q *Queries) DeleteEvent(ctx context.Context, id, reasonId string) error {
	if config.Cfg.Relay.FakeDeletion {
		_, err := q.db.Exec(ctx, fakeDeletionEvent, id, reasonId)
		return err
	} else {
		_, err := q.db.Exec(ctx, deleteEvent, id)
		return err
	}
}

const deleteOldsEvents = `-- name: DeleteOldsEvents :exec
DELETE FROM event WHERE pubkey = $1::text AND kind = $2::int AND created_at < (
    SELECT created_at FROM event WHERE pubkey = $1::text
    ORDER BY created_at DESC, id OFFSET 100 LIMIT 1
    )
`

type DeleteOldsEventsParams struct {
	Pubkey string `db:"pubkey" json:"pubkey"`
	Kind   int32  `db:"kind" json:"kind"`
}

func (q *Queries) DeleteOldsEvents(ctx context.Context, arg DeleteOldsEventsParams) error {
	_, err := q.db.Exec(ctx, deleteOldsEvents, arg.Pubkey, arg.Kind)
	return err
}

const insertEvent = `-- name: InsertEvent :exec
INSERT INTO event (
    id, pubkey, created_at, kind, tags, content, sig)
VALUES ($1::text, $2::text, $3::int, $4::int, $5::jsonb, $6::text, $7::text)
    ON CONFLICT (id) DO NOTHING
`

type InsertEventParams struct {
	ID        string `db:"id" json:"id"`
	Pubkey    string `db:"pubkey" json:"pubkey"`
	CreatedAt int32  `db:"created_at" json:"created_at"`
	Kind      int32  `db:"kind" json:"kind"`
	Tags      []byte `db:"tags" json:"tags"`
	Content   string `db:"content" json:"content"`
	Sig       string `db:"sig" json:"sig"`
}

func (q *Queries) InsertEvent(ctx context.Context, arg *nostr.Event) error {
	res, err := q.db.Exec(ctx, insertEvent,
		arg.ID,
		arg.PubKey,
		arg.CreatedAt,
		arg.Kind,
		arg.Tags,
		arg.Content,
		arg.Sig,
	)
	if res.RowsAffected() == 0 {
		return ErrDupEvent
	}
	return err
}
func (q *Queries) InsertEventBatch(ctx context.Context, arg []*nostr.Event) error {
	if len(arg) == 0 {
		return errors.New("no events to insert")
	}
	b := pgx.Batch{}
	for _, evt := range arg {
		b.Queue(insertEvent,
			evt.ID,
			evt.PubKey,
			evt.CreatedAt,
			evt.Kind,
			evt.Tags,
			evt.Content,
			evt.Sig,
		)
	}
	br := q.db.SendBatch(ctx, &b)
	defer br.Close()
	for i := 0; i < len(arg); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to insert event %d: %w", i, err)
		}
	}
	return nil
}
func (q *Queries) QueryEventsChan(ctx context.Context, filter nostr.Filter) (ch chan *nostr.Event, err error) {
	query, err := q.QueryEvents(ctx, filter)
	if err != nil {
		return nil, err
	}

	ch = make(chan *nostr.Event)
	go func() {
		defer close(ch)
		for _, evt := range query {
			select {
			case ch <- evt:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}
func (q *Queries) QueryEvents(ctx context.Context, filter nostr.Filter) (events []*nostr.Event, err error) {
	query, params, err := helper.QueryEventsSql(&config.Cfg.Relay, filter, false)
	if err != nil {
		return nil, err
	}

	rows, err := q.db.Query(ctx, query, params...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to fetch events using query %q: %w", query, err)
	}

	defer rows.Close()
	for rows.Next() {
		var evt nostr.Event
		var timestamp int64
		err := rows.Scan(&evt.ID, &evt.PubKey, &timestamp,
			&evt.Kind, &evt.Tags, &evt.Content, &evt.Sig)
		if err != nil {
			return nil, err
		}
		evt.CreatedAt = nostr.Timestamp(timestamp)
		events = append(events, &evt)
	}

	return events, nil
}

func (q *Queries) CountEvents(ctx context.Context, filter nostr.Filter) (int64, error) {
	query, params, err := helper.QueryEventsSql(&config.Cfg.Relay, filter, true)
	if err != nil {
		return 0, err
	}

	var count int64

	if err = q.db.QueryRow(ctx, query, params...).Scan(&count); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("failed to fetch events using query %q: %w", query, err)
	}
	return count, nil
}

const getOldEvents = `-- name: GetOldEvents :many
SELECT id, pubkey, created_at, kind, tags, content, sig
FROM event
WHERE created_at < $1::timestamptz
`

func (q *Queries) GetOldEvents(ctx context.Context, before time.Time) ([]*nostr.Event, error) {
	rows, err := q.db.Query(ctx, getOldEvents, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*nostr.Event
	for rows.Next() {
		var evt nostr.Event
		var timestamp int64
		err := rows.Scan(&evt.ID, &evt.PubKey, &timestamp,
			&evt.Kind, &evt.Tags, &evt.Content, &evt.Sig)
		if err != nil {
			return nil, err
		}
		evt.CreatedAt = nostr.Timestamp(timestamp)
		events = append(events, &evt)
	}

	return events, nil
}

const getAllEvents = `-- name: GetAllEvents :many
SELECT id, pubkey, created_at, kind, tags, content, sig
FROM event
`

func (q *Queries) GetAllEvents(ctx context.Context) ([]*nostr.Event, error) {
	rows, err := q.db.Query(ctx, getAllEvents)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*nostr.Event
	for rows.Next() {
		var evt nostr.Event
		var timestamp int64
		err := rows.Scan(&evt.ID, &evt.PubKey, &timestamp,
			&evt.Kind, &evt.Tags, &evt.Content, &evt.Sig)
		if err != nil {
			return nil, err
		}
		evt.CreatedAt = nostr.Timestamp(timestamp)
		events = append(events, &evt)
	}

	return events, nil
}

const streamAllEventsCursor = `-- name: StreamAllEvents :many
SELECT id, pubkey, created_at, kind, tags, content, sig
FROM event
WHERE (created_at > $1 OR (created_at = $1 AND id > $2))
ORDER BY created_at, id
LIMIT $3
`

// StreamAllEvents streams all events from the database in batches, using a cursor approach
// to avoid loading all events into memory at once. It returns a channel that emits slices of events.
// The channel will be closed when there are no more events to stream.
// The parameters are:
// - ctx: the context for cancellation and timeout
// - pageSize: the number of events to fetch in each batch
func (q *Queries) StreamAllEvents(ctx context.Context, pageSize int) <-chan *[]nostr.Event {
	out := make(chan *[]nostr.Event)

	go func() {
		defer close(out)

		var lastCreatedAt int64 = 0
		var lastID string = ""

		for {
			rows, err := q.db.Query(ctx, streamAllEventsCursor, lastCreatedAt, lastID, pageSize)
			if err != nil {
				return // você pode logar o erro aqui
			}

			var batch []nostr.Event
			count := 0

			for rows.Next() {
				var evt nostr.Event
				var timestamp int64

				err := rows.Scan(&evt.ID, &evt.PubKey, &timestamp,
					&evt.Kind, &evt.Tags, &evt.Content, &evt.Sig)
				if err != nil {
					rows.Close()
					return
				}

				evt.CreatedAt = nostr.Timestamp(timestamp)
				batch = append(batch, evt)

				// Atualiza o cursor
				lastCreatedAt = timestamp
				lastID = evt.ID
				count++
			}

			rows.Close()

			if count > 0 {
				// envia um ponteiro para a fatia
				out <- &batch
			}

			if count < pageSize {
				break
			}
		}
	}()

	return out
}
