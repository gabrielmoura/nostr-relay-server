package db

import (
	"context"
	"fmt"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/internal/cordn"
	"github.com/jackc/pgx/v5"
)

func (q *Queries) InsertCordnKeyPackage(ctx context.Context, value cordn.KeyPackage) error {
	_, err := q.db.Exec(ctx, `
INSERT INTO cordn_key_package
    (key_package_ref, stable_pubkey, key_package, publication_event, is_last_resort, published_at)
VALUES ($1, $2, $3, $4, $5, $6)`,
		value.Reference,
		value.StablePubkey,
		value.Payload,
		value.PublicationJSON,
		value.LastResort,
		value.PublishedAt,
	)
	if err != nil {
		return fmt.Errorf("insert Cordn key package: %w", err)
	}
	return nil
}

func (q *Queries) ListCordnKeyPackages(ctx context.Context, stablePubkey string) ([]cordn.KeyPackage, error) {
	rows, err := q.db.Query(ctx, `
SELECT key_package_ref, stable_pubkey, key_package, publication_event, is_last_resort, published_at
FROM cordn_key_package
WHERE stable_pubkey = $1
ORDER BY published_at, key_package_ref`, stablePubkey)
	if err != nil {
		return nil, fmt.Errorf("list Cordn key packages: %w", err)
	}
	defer rows.Close()

	values := []cordn.KeyPackage{}
	for rows.Next() {
		var value cordn.KeyPackage
		if err := rows.Scan(
			&value.Reference,
			&value.StablePubkey,
			&value.Payload,
			&value.PublicationJSON,
			&value.LastResort,
			&value.PublishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan Cordn key package: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Cordn key packages: %w", err)
	}
	return values, nil
}

func (q *Queries) ListAllCordnKeyPackages(ctx context.Context) ([]cordn.KeyPackage, error) {
	rows, err := q.db.Query(ctx, `
SELECT key_package_ref, stable_pubkey, key_package, publication_event, is_last_resort, published_at
FROM cordn_key_package
ORDER BY published_at, key_package_ref`)
	if err != nil {
		return nil, fmt.Errorf("list all Cordn key packages: %w", err)
	}
	defer rows.Close()

	values := []cordn.KeyPackage{}
	for rows.Next() {
		var value cordn.KeyPackage
		if err := rows.Scan(&value.Reference, &value.StablePubkey, &value.Payload, &value.PublicationJSON, &value.LastResort, &value.PublishedAt); err != nil {
			return nil, fmt.Errorf("scan Cordn key package: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all Cordn key packages: %w", err)
	}
	return values, nil
}

func (q *Queries) TakeCordnKeyPackage(ctx context.Context, reference string) (cordn.KeyPackage, bool, error) {
	var value cordn.KeyPackage
	err := q.db.QueryRow(ctx, `
DELETE FROM cordn_key_package
WHERE key_package_ref = $1
RETURNING key_package_ref, stable_pubkey, key_package, publication_event, is_last_resort, published_at`, reference).Scan(
		&value.Reference,
		&value.StablePubkey,
		&value.Payload,
		&value.PublicationJSON,
		&value.LastResort,
		&value.PublishedAt,
	)
	if err == pgx.ErrNoRows {
		return cordn.KeyPackage{}, false, nil
	}
	if err != nil {
		return cordn.KeyPackage{}, false, fmt.Errorf("take Cordn key package: %w", err)
	}
	return value, true, nil
}

func (q *Queries) DeleteCordnKeyPackage(ctx context.Context, reference string) (cordn.KeyPackage, bool, error) {
	return q.TakeCordnKeyPackage(ctx, reference)
}

func (q *Queries) InsertCordnWelcome(ctx context.Context, value cordn.Welcome) error {
	_, err := q.db.Exec(ctx, `
INSERT INTO cordn_pending_welcome
    (target_pubkey, key_package_ref, welcome, join_after_cursor, created_at)
VALUES ($1, $2, $3, $4, $5)`,
		value.TargetPubkey,
		value.KeyPackageRef,
		value.Payload,
		value.JoinAfterCursor,
		value.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert Cordn welcome: %w", err)
	}
	return nil
}

func (q *Queries) ListCordnWelcomes(ctx context.Context, targetPubkey string) ([]cordn.Welcome, error) {
	rows, err := q.db.Query(ctx, `
SELECT target_pubkey, key_package_ref, welcome, join_after_cursor, created_at
FROM cordn_pending_welcome
WHERE target_pubkey = $1
ORDER BY created_at, key_package_ref`, targetPubkey)
	if err != nil {
		return nil, fmt.Errorf("list Cordn welcomes: %w", err)
	}
	defer rows.Close()

	values := []cordn.Welcome{}
	for rows.Next() {
		var value cordn.Welcome
		if err := rows.Scan(&value.TargetPubkey, &value.KeyPackageRef, &value.Payload, &value.JoinAfterCursor, &value.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan Cordn welcome: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Cordn welcomes: %w", err)
	}
	return values, nil
}

func (q *Queries) DeleteCordnWelcome(ctx context.Context, targetPubkey, keyPackageRef string, createdAt time.Time) error {
	command, err := q.db.Exec(ctx, `
DELETE FROM cordn_pending_welcome
WHERE target_pubkey = $1 AND key_package_ref = $2 AND created_at = $3`, targetPubkey, keyPackageRef, createdAt)
	if err != nil {
		return fmt.Errorf("delete Cordn welcome: %w", err)
	}
	if command.RowsAffected() == 0 {
		return fmt.Errorf("delete Cordn welcome: %w", pgx.ErrNoRows)
	}
	return nil
}

func (q *Queries) InsertCordnJoinRequest(ctx context.Context, value cordn.JoinRequest) error {
	_, err := q.db.Exec(ctx, `
INSERT INTO cordn_pending_join_request (gid, requester_pubkey, key_package_ref, created_at)
VALUES ($1, $2, $3, $4)`, value.GroupID, value.RequesterPubkey, value.KeyPackageRef, value.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert Cordn join request: %w", err)
	}
	return nil
}

func (q *Queries) ListCordnJoinRequests(ctx context.Context, groupID string) ([]cordn.JoinRequest, error) {
	rows, err := q.db.Query(ctx, `
SELECT gid, requester_pubkey, key_package_ref, created_at
FROM cordn_pending_join_request
WHERE gid = $1
ORDER BY created_at, requester_pubkey, key_package_ref`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list Cordn join requests: %w", err)
	}
	defer rows.Close()

	values := []cordn.JoinRequest{}
	for rows.Next() {
		var value cordn.JoinRequest
		if err := rows.Scan(&value.GroupID, &value.RequesterPubkey, &value.KeyPackageRef, &value.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan Cordn join request: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Cordn join requests: %w", err)
	}
	return values, nil
}

func (q *Queries) DeleteCordnJoinRequest(ctx context.Context, value cordn.JoinRequest) error {
	command, err := q.db.Exec(ctx, `
DELETE FROM cordn_pending_join_request
WHERE gid = $1 AND requester_pubkey = $2 AND created_at = $3`,
		value.GroupID,
		value.RequesterPubkey,
		value.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("delete Cordn join request: %w", err)
	}
	if command.RowsAffected() == 0 {
		return fmt.Errorf("delete Cordn join request: %w", pgx.ErrNoRows)
	}
	return nil
}

func (q *Queries) DeleteExpiredCordnRecords(ctx context.Context, before time.Time) error {
	statements := []string{
		"DELETE FROM cordn_pending_welcome WHERE created_at < $1",
		"DELETE FROM cordn_pending_join_request WHERE created_at < $1",
		"DELETE FROM cordn_group_message WHERE created_at < $1",
	}
	for _, statement := range statements {
		if _, err := q.db.Exec(ctx, statement, before); err != nil {
			return fmt.Errorf("delete expired Cordn records: %w", err)
		}
	}
	return nil
}

// AppendCordnGroupMessage allocates the next cursor within one gid and stores
// the opaque payload atomically. Cursors have no meaning outside that gid.
func (q *Queries) AppendCordnGroupMessage(ctx context.Context, groupID string, payload []byte, now time.Time) (cordn.GroupMessage, error) {
	transactional, ok := q.db.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return cordn.GroupMessage{}, fmt.Errorf("Cordn message storage does not support transactions")
	}
	tx, err := transactional.Begin(ctx)
	if err != nil {
		return cordn.GroupMessage{}, fmt.Errorf("begin Cordn message transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var cursor int64
	err = tx.QueryRow(ctx, `
INSERT INTO cordn_group_cursor (gid, next_cursor)
VALUES ($1, 1)
ON CONFLICT (gid) DO UPDATE SET next_cursor = cordn_group_cursor.next_cursor + 1
RETURNING next_cursor`, groupID).Scan(&cursor)
	if err != nil {
		return cordn.GroupMessage{}, fmt.Errorf("allocate Cordn cursor: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO cordn_group_message (gid, cursor, payload, created_at)
VALUES ($1, $2, $3, $4)`, groupID, cursor, payload, now); err != nil {
		return cordn.GroupMessage{}, fmt.Errorf("insert Cordn group message: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return cordn.GroupMessage{}, fmt.Errorf("commit Cordn group message: %w", err)
	}
	return cordn.GroupMessage{GroupID: groupID, Cursor: cursor, Payload: payload, CreatedAt: now}, nil
}

func (q *Queries) ListCordnGroupMessages(ctx context.Context, groupID string, afterCursor int64) ([]cordn.GroupMessage, error) {
	rows, err := q.db.Query(ctx, `
SELECT gid, cursor, payload, created_at
FROM cordn_group_message
WHERE gid = $1 AND cursor > $2
ORDER BY cursor`, groupID, afterCursor)
	if err != nil {
		return nil, fmt.Errorf("list Cordn group messages: %w", err)
	}
	defer rows.Close()

	values := []cordn.GroupMessage{}
	for rows.Next() {
		var value cordn.GroupMessage
		if err := rows.Scan(&value.GroupID, &value.Cursor, &value.Payload, &value.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan Cordn group message: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Cordn group messages: %w", err)
	}
	return values, nil
}
