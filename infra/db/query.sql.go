package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/jmoiron/sqlx"
	"github.com/nbd-wtf/go-nostr"
	"strings"
)

var (
	TooManyIDs       = errors.New("too many ids")
	TooManyAuthors   = errors.New("too many authors")
	TooManyKinds     = errors.New("too many kinds")
	TooManyTagValues = errors.New("too many tag values")
	EmptyTagSet      = errors.New("empty tag set")
)

const deleteEvent = `-- name: DeleteEvent :exec
DELETE FROM event WHERE id = $1::text
`

func (q *Queries) DeleteEvent(ctx context.Context, id string) error {
	_, err := q.db.Exec(ctx, deleteEvent, id)
	return err
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
	_, err := q.db.Exec(ctx, insertEvent,
		arg.ID,
		arg.PubKey,
		arg.CreatedAt,
		arg.Kind,
		arg.Tags,
		arg.Content,
		arg.Sig,
	)
	return err
}

func (q *Queries) queryEventsSql(filter nostr.Filter, doCount bool) (string, []any, error) {
	conditions := make([]string, 0, 7)
	params := make([]any, 0, 20)

	if len(filter.IDs) > 0 {
		if len(filter.IDs) > config.Cfg.Relay.QueryIDsLimit {
			return "", nil, TooManyIDs
		}
		for _, v := range filter.IDs {
			params = append(params, v)
		}
		conditions = append(conditions, `id IN (`+makePlaceHolders(len(filter.IDs))+`)`)
	}

	if len(filter.Authors) > 0 {
		if len(filter.Authors) > config.Cfg.Relay.QueryAuthorsLimit {
			return "", nil, TooManyAuthors
		}
		for _, v := range filter.Authors {
			params = append(params, v)
		}
		conditions = append(conditions, `pubkey IN (`+makePlaceHolders(len(filter.Authors))+`)`)
	}

	if len(filter.Kinds) > 0 {
		if len(filter.Kinds) > config.Cfg.Relay.QueryKindsLimit {
			return "", nil, TooManyKinds
		}
		for _, v := range filter.Kinds {
			params = append(params, v)
		}
		conditions = append(conditions, `kind IN (`+makePlaceHolders(len(filter.Kinds))+`)`)
	}

	totalTags := 0
	for _, values := range filter.Tags {
		if len(values) == 0 {
			return "", nil, EmptyTagSet
		}
		for _, tagValue := range values {
			params = append(params, tagValue)
		}
		conditions = append(conditions, `tagvalues && ARRAY[`+makePlaceHolders(len(values))+`]`)
		totalTags += len(values)
		if totalTags > config.Cfg.Relay.QueryTagsLimit {
			return "", nil, TooManyTagValues
		}
	}

	if filter.Since != nil {
		conditions = append(conditions, `created_at >= ?`)
		params = append(params, filter.Since)
	}
	if filter.Until != nil {
		conditions = append(conditions, `created_at <= ?`)
		params = append(params, filter.Until)
	}
	if filter.Search != "" {
		conditions = append(conditions, `content LIKE ?`)
		params = append(params, `%`+strings.ReplaceAll(filter.Search, `%`, `\%`)+`%`)
	}

	if len(conditions) == 0 {
		conditions = append(conditions, `true`)
	}

	if filter.Limit < 1 || filter.Limit > config.Cfg.Relay.QueryLimit {
		params = append(params, config.Cfg.Relay.QueryLimit)
	} else {
		params = append(params, filter.Limit)
	}

	var query string
	if doCount {
		query = sqlx.Rebind(sqlx.BindType("postgres"), `SELECT COUNT(*) FROM event WHERE `+
			strings.Join(conditions, " AND ")+" LIMIT ?")
	} else {
		query = sqlx.Rebind(sqlx.BindType("postgres"), `SELECT id, pubkey, created_at, kind, tags, content, sig FROM event WHERE `+
			strings.Join(conditions, " AND ")+" ORDER BY created_at DESC, id LIMIT ?")
	}

	return query, params, nil
}
func makePlaceHolders(n int) string {
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

func (q *Queries) QueryEvents(ctx context.Context, filter nostr.Filter) (ch chan *nostr.Event, err error) {
	query, params, err := q.queryEventsSql(filter, false)
	if err != nil {
		return nil, err
	}

	rows, err := q.db.Query(ctx, query, params...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to fetch events using query %q: %w", query, err)
	}

	ch = make(chan *nostr.Event)
	go func() {
		defer rows.Close()
		defer close(ch)
		for rows.Next() {
			var evt nostr.Event
			var timestamp int64
			err := rows.Scan(&evt.ID, &evt.PubKey, &timestamp,
				&evt.Kind, &evt.Tags, &evt.Content, &evt.Sig)
			if err != nil {
				return
			}
			evt.CreatedAt = nostr.Timestamp(timestamp)
			select {
			case ch <- &evt:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (q *Queries) CountEvents(ctx context.Context, filter nostr.Filter) (int64, error) {
	query, params, err := q.queryEventsSql(filter, true)
	if err != nil {
		return 0, err
	}

	var count int64

	if err = q.db.QueryRow(ctx, query, params...).Scan(&count); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("failed to fetch events using query %q: %w", query, err)
	}
	return count, nil
}

// GetCountReportsKey fetches the number of reports for a given Key
func (q *Queries) GetCountReportsKey(ctx context.Context, key string) (int64, error) {
	filter := nostr.Filter{
		Kinds: []int{nostr.KindReporting},
		Tags: nostr.TagMap{
			"p": {key},
		},
	}
	query, params, err := q.queryEventsSql(filter, true)
	if err != nil {
		return 0, err
	}

	var count int64

	if err = q.db.QueryRow(ctx, query, params...).Scan(&count); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("failed to fetch events using query %q: %w", query, err)
	}
	return count, nil
}

const getUserBannedByKey = `-- name: GetUserBannedByKey :one
SELECT b.reason
FROM banned_users b
JOIN profiles p ON b.user_id = p.id
WHERE p.public_key = $1::text
LIMIT 1;
`

func (q *Queries) GetUserBannedByKey(ctx context.Context, key string) (reason string, exists bool, err error) {
	err = q.db.QueryRow(ctx, getUserBannedByKey, key).Scan(&reason)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return reason, true, nil
}

const BanUserByPubKey = `-- name: BanUserByPubKey :exec
INSERT INTO banned_users (user_id, reason, related_ids)
VALUES (
    (SELECT id FROM profiles WHERE public_key = $1::text),
    $2::text,
    $3::VARCHAR(60)[]
);
`

func (q *Queries) BanUserByPubKey(ctx context.Context, key, reason string, relatedIds []string) error {
	_, err := q.db.Exec(ctx, BanUserByPubKey, key, reason, relatedIds)
	return err
}
