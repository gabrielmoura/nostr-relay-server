package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

const upsertNIP29Ban = `
INSERT INTO nip29_group_bans (relay, group_id, user_id, reason, created_by, created_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (relay, group_id, user_id) DO UPDATE SET
	reason = EXCLUDED.reason,
	created_by = EXCLUDED.created_by,
	created_at = NOW()
`

func (q *Queries) UpsertNIP29Ban(ctx context.Context, relay, groupID, userID, reason, createdBy string) error {
	_, err := q.db.Exec(ctx, upsertNIP29Ban, relay, groupID, userID, reason, createdBy)
	return err
}

const deleteNIP29Ban = `
DELETE FROM nip29_group_bans
WHERE relay = $1 AND group_id = $2 AND user_id = $3
`

func (q *Queries) DeleteNIP29Ban(ctx context.Context, relay, groupID, userID string) error {
	_, err := q.db.Exec(ctx, deleteNIP29Ban, relay, groupID, userID)
	return err
}

const getNIP29Ban = `
SELECT COALESCE(reason, '')
FROM nip29_group_bans
WHERE relay = $1 AND group_id = $2 AND user_id = $3
`

func (q *Queries) GetNIP29Ban(ctx context.Context, relay, groupID, userID string) (string, bool, error) {
	var reason string
	err := q.db.QueryRow(ctx, getNIP29Ban, relay, groupID, userID).Scan(&reason)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return reason, true, nil
}

const insertNIP29Invite = `
INSERT INTO nip29_group_invites (relay, group_id, code, created_by, max_uses, uses, expires_at, revoked_at, created_at, last_used_at)
VALUES ($1, $2, $3, $4, $5, 0, $6, NULL, NOW(), NULL)
ON CONFLICT (relay, group_id, code) DO UPDATE SET
	created_by = EXCLUDED.created_by,
	max_uses = EXCLUDED.max_uses,
	expires_at = EXCLUDED.expires_at,
	revoked_at = NULL,
	created_at = NOW()
`

func (q *Queries) UpsertNIP29Invite(ctx context.Context, invite NIP29Invite) error {
	_, err := q.db.Exec(ctx, insertNIP29Invite, invite.Relay, invite.GroupID, invite.Code, invite.CreatedBy, invite.MaxUses, invite.ExpiresAt)
	return err
}

const getNIP29Invite = `
SELECT relay, group_id, code, created_by, max_uses, uses, expires_at, revoked_at, created_at, last_used_at
FROM nip29_group_invites
WHERE relay = $1 AND group_id = $2 AND code = $3
`

func (q *Queries) GetNIP29Invite(ctx context.Context, relay, groupID, code string) (*NIP29Invite, bool, error) {
	row := q.db.QueryRow(ctx, getNIP29Invite, relay, groupID, code)
	invite, err := scanNIP29Invite(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return invite, true, nil
}

func scanNIP29Invite(row interface{ Scan(...any) error }) (*NIP29Invite, error) {
	var invite NIP29Invite
	var expiresAt *time.Time
	var revokedAt *time.Time
	var lastUsedAt *time.Time
	err := row.Scan(
		&invite.Relay,
		&invite.GroupID,
		&invite.Code,
		&invite.CreatedBy,
		&invite.MaxUses,
		&invite.Uses,
		&expiresAt,
		&revokedAt,
		&invite.CreatedAt,
		&lastUsedAt,
	)
	if err != nil {
		return nil, err
	}
	invite.ExpiresAt = expiresAt
	invite.RevokedAt = revokedAt
	invite.LastUsedAt = lastUsedAt
	return &invite, nil
}

const consumeNIP29Invite = `
UPDATE nip29_group_invites
SET uses = uses + 1,
	last_used_at = NOW()
WHERE relay = $1
	AND group_id = $2
	AND code = $3
	AND revoked_at IS NULL
	AND (expires_at IS NULL OR expires_at > NOW())
	AND uses < max_uses
`

func (q *Queries) ConsumeNIP29Invite(ctx context.Context, relay, groupID, code string) (bool, error) {
	res, err := q.db.Exec(ctx, consumeNIP29Invite, relay, groupID, code)
	if err != nil {
		return false, err
	}
	return res.RowsAffected() > 0, nil
}

const revokeNIP29Invite = `
UPDATE nip29_group_invites
SET revoked_at = NOW()
WHERE relay = $1 AND group_id = $2 AND code = $3
`

func (q *Queries) RevokeNIP29Invite(ctx context.Context, relay, groupID, code string) error {
	_, err := q.db.Exec(ctx, revokeNIP29Invite, relay, groupID, code)
	return err
}
