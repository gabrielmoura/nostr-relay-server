package db

import (
	"context"
	"strings"
)

const upsertAllowedPubKey = `
INSERT INTO nip86_allowed_pubkeys (pubkey, reason, created_by, created_at, updated_at)
VALUES ($1::varchar, $2::text, $3::varchar, NOW(), NOW())
ON CONFLICT (pubkey) DO UPDATE SET
    reason = EXCLUDED.reason,
    created_by = EXCLUDED.created_by,
    updated_at = NOW();
`

const deleteAllowedPubKey = `DELETE FROM nip86_allowed_pubkeys WHERE pubkey = $1::varchar`

const listAllowedPubKeys = `
SELECT pubkey, COALESCE(reason, ''), created_by, created_at, updated_at
FROM nip86_allowed_pubkeys
ORDER BY pubkey ASC
`

const listBannedPubKeys = `
SELECT p.public_key, COALESCE(b.reason, ''), '' AS created_by, b.created_at, b.created_at AS updated_at
FROM (
    SELECT DISTINCT ON (user_id) user_id, reason, created_at, id
    FROM banned_users
    ORDER BY user_id, id DESC
) b
JOIN profiles p ON p.id = b.user_id
ORDER BY p.public_key ASC
`

func (q *Queries) UpsertAllowedPubKey(ctx context.Context, pubkey, reason, createdBy string) error {
	_, err := q.db.Exec(ctx, upsertAllowedPubKey, strings.ToLower(pubkey), reason, strings.ToLower(createdBy))
	return err
}

func (q *Queries) DeleteAllowedPubKey(ctx context.Context, pubkey string) error {
	_, err := q.db.Exec(ctx, deleteAllowedPubKey, strings.ToLower(pubkey))
	return err
}

func (q *Queries) ListAllowedPubKeys(ctx context.Context) ([]NIP86PubKeyRecord, error) {
	rows, err := q.db.Query(ctx, listAllowedPubKeys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]NIP86PubKeyRecord, 0)
	for rows.Next() {
		var item NIP86PubKeyRecord
		if err := rows.Scan(&item.PubKey, &item.Reason, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) ListBannedPubKeys(ctx context.Context) ([]NIP86PubKeyRecord, error) {
	rows, err := q.db.Query(ctx, listBannedPubKeys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]NIP86PubKeyRecord, 0)
	for rows.Next() {
		var item NIP86PubKeyRecord
		if err := rows.Scan(&item.PubKey, &item.Reason, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
