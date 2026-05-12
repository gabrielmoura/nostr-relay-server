package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const listBannedUsersBase = `
SELECT p.id, p.public_key, p.name, p.about, p.picture, p.banner, p.website, p.display_name, p.lud16, p.pronouns, p.nip05, p.bot,
       b.reason, COALESCE(b.related_ids, '{}'), NULL::timestamptz AS created_at
FROM (
    SELECT DISTINCT ON (user_id) user_id, reason, related_ids, id
    FROM banned_users
    ORDER BY user_id, id DESC
) b
JOIN profiles p ON p.id = b.user_id
`

const countBannedUsersBase = `
SELECT COUNT(*)
FROM (
    SELECT DISTINCT ON (user_id) user_id, reason, related_ids, id
    FROM banned_users
    ORDER BY user_id, id DESC
) b
JOIN profiles p ON p.id = b.user_id
`

const getLatestBanRecordByKey = `-- name: GetLatestBanRecordByKey :one
SELECT b.reason, COALESCE(b.related_ids, '{}'), NULL::timestamptz AS created_at
FROM banned_users b
JOIN profiles p ON b.user_id = p.id
WHERE p.public_key = $1::text
ORDER BY b.id DESC
LIMIT 1;
`

const getLatestBanRecordsByKeys = `-- name: GetLatestBanRecordsByKeys :many
SELECT p.public_key, b.reason, COALESCE(b.related_ids, '{}'), NULL::timestamptz AS created_at
FROM (
    SELECT DISTINCT ON (user_id) user_id, reason, related_ids, id
    FROM banned_users
    ORDER BY user_id, id DESC
) b
JOIN profiles p ON p.id = b.user_id
WHERE p.public_key = ANY($1::text[]);
`

func buildBannedUsersQuery(base string, query string) (string, []any) {
	query = strings.TrimSpace(query)
	if query == "" {
		return base, []any{}
	}
	needle := "%" + query + "%"
	return base + `
WHERE p.public_key ILIKE $1 OR p.name ILIKE $1 OR p.display_name ILIKE $1 OR p.nip05 ILIKE $1 OR b.reason ILIKE $1`, []any{needle}
}

func (q *Queries) ListBannedUsers(ctx context.Context, query string, limit int, offset int) ([]BannedUserRecord, int64, error) {
	countQuery, countArgs := buildBannedUsersQuery(countBannedUsersBase, query)
	var total int64
	if err := q.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery, listArgs := buildBannedUsersQuery(listBannedUsersBase, query)
	listArgs = append(listArgs, limit, offset)
	listQuery += fmt.Sprintf(" ORDER BY p.public_key LIMIT $%d OFFSET $%d", len(listArgs)-1, len(listArgs))

	rows, err := q.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]BannedUserRecord, 0, limit)
	for rows.Next() {
		var item BannedUserRecord
		if err := rows.Scan(&item.ID, &item.PublicKey, &item.Name, &item.About, &item.Picture, &item.Banner, &item.Website, &item.DisplayName, &item.Lud16, &item.Pronouns, &item.Nip05, &item.Bot, &item.Reason, &item.RelatedIDs, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	return items, total, rows.Err()
}

func (q *Queries) GetLatestBanRecordByKey(ctx context.Context, key string) (BannedUserRecord, bool, error) {
	profile, err := q.GetProfileByPublicKey(ctx, key)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return BannedUserRecord{}, false, err
	}

	var record BannedUserRecord
	var relatedIDs []string
	err = q.db.QueryRow(ctx, getLatestBanRecordByKey, key).Scan(&record.Reason, &relatedIDs, &record.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return BannedUserRecord{}, false, nil
	}
	if err != nil {
		return BannedUserRecord{}, false, err
	}

	record.Profile = profile
	record.RelatedIDs = relatedIDs
	return record, true, nil
}

func (q *Queries) GetLatestBanRecordsByKeys(ctx context.Context, keys []string) (map[string]BannedUserRecord, error) {
	records := make(map[string]BannedUserRecord, len(keys))
	if len(keys) == 0 {
		return records, nil
	}

	rows, err := q.db.Query(ctx, getLatestBanRecordsByKeys, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pubkey string
		var record BannedUserRecord
		if err := rows.Scan(&pubkey, &record.Reason, &record.RelatedIDs, &record.CreatedAt); err != nil {
			return nil, err
		}
		records[pubkey] = record
	}

	return records, rows.Err()
}
