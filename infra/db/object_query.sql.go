package db

import (
	"context"
	"database/sql"
	"errors"
)

const removeObject = `-- name: RemoveObject :exec
DELETE FROM objects WHERE hash = $1::text
`

func (q *Queries) RemoveObject(ctx context.Context, hash string) error {
	_, err := q.db.Exec(ctx, removeObject, hash)
	return err
}

const getObjectByHash = `-- name: GetObjectByHash :one
SELECT hash, created_at, mime_type, size, blocked, expires_at, blocked_by_reason, public_key, tags
FROM objects
WHERE hash = $1::text
LIMIT 1
`

func (q *Queries) GetObjectByHash(ctx context.Context, hash string) (Object, error) {
	var obj Object
	err := q.db.QueryRow(ctx, getObjectByHash, hash).Scan(&obj.Hash, &obj.CreatedAt, &obj.MimeType, &obj.Size, &obj.Blocked, &obj.ExpiresAt, &obj.BlockedByReason, &obj.PublicKey, &obj.Tags)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return obj, nil
	}
	return obj, err
}

const insertObject = `-- name: InsertObject :exec
INSERT INTO objects (hash, created_at, mime_type, size, blocked, expires_at, blocked_by_reason, public_key, tags)
VALUES ($1::text, $2::timestamptz, $3::text, $4::bigint, $5::bool, $6::timestamptz, $7::text, $8::text, $9::jsonb)
ON CONFLICT (hash) DO NOTHING
`

func (q *Queries) InsertObject(ctx context.Context, arg *Object) error {
	_, err := q.db.Exec(ctx, insertObject,
		arg.Hash,
		arg.CreatedAt,
		arg.MimeType,
		arg.Size,
		arg.Blocked,
		arg.ExpiresAt,
		arg.BlockedByReason,
		arg.PublicKey,
		arg.Tags,
	)
	return err
}

const getAllObjectByKey = `-- name: GetAllObjectByKey :many
SELECT hash, created_at, mime_type, size, blocked, expires_at, blocked_by_reason, public_key, tags
FROM objects
WHERE public_key = $1::text
ORDER BY created_at DESC
LIMIT $2
`

func (q *Queries) GetAllObjectByKey(ctx context.Context, publicKey string, limit int32) ([]Object, error) {
	rows, err := q.db.Query(ctx, getAllObjectByKey, publicKey, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Object
	for rows.Next() {
		var item Object
		if err := rows.Scan(&item.Hash, &item.CreatedAt, &item.MimeType, &item.Size, &item.Blocked, &item.ExpiresAt, &item.BlockedByReason, &item.PublicKey, &item.Tags); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
