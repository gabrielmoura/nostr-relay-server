package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/db/helper"
	"github.com/nbd-wtf/go-nostr"
)

const insertUserProfile = `-- name: InsertUserProfile :exec
INSERT INTO profiles (public_key, name,about,picture,bot,banner,website, display_name, lud16, pronouns, nip05)
VALUES ($1::text, $2::text, $3::text, $4::text, $5::bool, $6::text, $7::text, $8::text, $9::text, $10::text, $11::text)
ON CONFLICT (public_key) DO UPDATE SET 
name = $2::text,
about = $3::text,
picture = $4::text,
bot = $5::bool,
banner = $6::text,
website = $7::text,
display_name = $8::text,
lud16 = $9::text,
pronouns = $10::text,
nip05 = $11::text;
`

func (q *Queries) InsertUserProfile(ctx context.Context, arg *Profile) error {
	_, err := q.db.Exec(ctx, insertUserProfile,
		arg.PublicKey,
		arg.Name,
		arg.About,
		arg.Picture,
		arg.Bot,
		arg.Banner,
		arg.Website,
		arg.DisplayName,
		arg.Lud16,
		arg.Pronouns,
		arg.Nip05,
	)
	return err
}

const checkStoreUserPermission = `-- name: CheckStoreUserPermission :one
SELECT enable_store_files
FROM profiles
WHERE public_key = $1::text
LIMIT 1;
`

// CheckStoreUserPermission checks if a user has enabled file storage permissions.
func (q *Queries) CheckStoreUserPermission(ctx context.Context, key string) (bool, error) {
	var enableStoreFiles bool
	err := q.db.QueryRow(ctx, checkStoreUserPermission, key).Scan(&enableStoreFiles)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // User not found, default to false
		}
		return false, fmt.Errorf("failed to check store user permission: %w", err)
	}
	return enableStoreFiles, nil
}

const checkNip05Permission = `-- name: CheckNip05Permission :one
SELECT enable_nip05
FROM profiles
WHERE public_key = $1::text
LIMIT 1;
`

// CheckNip05Permission checks if a user has enabled NIP-05 support.
func (q *Queries) CheckNip05Permission(ctx context.Context, key string) (bool, error) {
	var enableNip05 bool
	err := q.db.QueryRow(ctx, checkNip05Permission, key).Scan(&enableNip05)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // User not found, default to false
		}
		return false, fmt.Errorf("failed to check NIP-05 permission: %w", err)
	}
	return enableNip05, nil
}

const getUserBannedByKey = `-- name: GetUserBannedByKey :one
SELECT b.reason
FROM banned_users b
JOIN profiles p ON b.user_id = p.id
WHERE p.public_key = $1::text
ORDER BY b.id DESC
LIMIT 1;
`

// GetUserBannedByKey checks if a user is banned by their public key and returns the reason if they are banned.
func (q *Queries) GetUserBannedByKey(ctx context.Context, key string) (reason string, exists bool, err error) {
	err = q.db.QueryRow(ctx, getUserBannedByKey, key).Scan(&reason)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return reason, true, nil
}

const banUserByPubKey = `-- name: BanUserByPubKey :exec
WITH target AS (
    SELECT id
    FROM profiles
    WHERE public_key = $1::text
    LIMIT 1
), purge AS (
    DELETE FROM banned_users
    WHERE user_id = (SELECT id FROM target)
)
INSERT INTO banned_users (user_id, reason, related_ids, created_at)
SELECT id, $2::text, $3::VARCHAR(60)[], NOW()
FROM target;
`

// BanUserByPubKey bans a user by their public key, providing a reason and related IDs.
func (q *Queries) BanUserByPubKey(ctx context.Context, key, reason string, relatedIds []string) error {
	_, err := q.db.Exec(ctx, banUserByPubKey, key, reason, relatedIds)
	if err == nil {
		_ = cache.Delete("ban:" + key)
	}
	return err
}

const unbanUserByPubKey = `-- name: UnbanUserByPubKey :exec
DELETE FROM banned_users
WHERE user_id = (SELECT id FROM profiles WHERE public_key = $1::text LIMIT 1);
`

func (q *Queries) UnbanUserByPubKey(ctx context.Context, key string) error {
	_, err := q.db.Exec(ctx, unbanUserByPubKey, key)
	if err == nil {
		_ = cache.Delete("ban:" + key)
	}
	return err
}

// GetCountReportsKey fetches the number of reports for a given Key
func (q *Queries) GetCountReportsKey(ctx context.Context, key string) (int64, error) {
	filter := nostr.Filter{
		Kinds: []int{nostr.KindReporting},
		Tags: nostr.TagMap{
			"p": {key},
		},
	}
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
