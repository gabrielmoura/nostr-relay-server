package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

type NIP05Identity struct {
	Name        string    `json:"name"`
	PublicKey   string    `json:"public_key"`
	DisplayName string    `json:"display_name,omitempty"`
	ProfileName string    `json:"profile_name,omitempty"`
	Picture     string    `json:"picture,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

const upsertNIP05Identity = `-- name: UpsertNIP05Identity :exec
WITH cleaned AS (
    DELETE FROM nip05_identities
    WHERE name = $1::text OR public_key = $2::text
)
INSERT INTO nip05_identities (name, public_key, created_at, updated_at)
VALUES ($1::text, $2::text, NOW(), NOW());
`

func (q *Queries) UpsertNIP05Identity(ctx context.Context, name string, publicKey string) error {
	_, err := q.db.Exec(ctx, upsertNIP05Identity, name, publicKey)
	return err
}

const deleteNIP05IdentityByName = `-- name: DeleteNIP05IdentityByName :exec
DELETE FROM nip05_identities
WHERE name = $1::text;
`

func (q *Queries) DeleteNIP05IdentityByName(ctx context.Context, name string) error {
	_, err := q.db.Exec(ctx, deleteNIP05IdentityByName, name)
	return err
}

const getNIP05IdentityByPublicKey = `-- name: GetNIP05IdentityByPublicKey :one
SELECT n.name,
       n.public_key,
       COALESCE(NULLIF(p.display_name, ''), NULLIF(p.name, ''), p.public_key) AS display_name,
       p.name,
       p.picture,
       n.created_at,
       n.updated_at
FROM nip05_identities n
JOIN profiles p ON p.public_key = n.public_key
WHERE n.public_key = $1::text
LIMIT 1;
`

func (q *Queries) GetNIP05IdentityByPublicKey(ctx context.Context, publicKey string) (NIP05Identity, error) {
	var row NIP05Identity
	err := q.db.QueryRow(ctx, getNIP05IdentityByPublicKey, publicKey).Scan(
		&row.Name,
		&row.PublicKey,
		&row.DisplayName,
		&row.ProfileName,
		&row.Picture,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	return row, err
}

const listNIP05IdentitiesBase = `
SELECT n.name,
       n.public_key,
       COALESCE(NULLIF(p.display_name, ''), NULLIF(p.name, ''), p.public_key) AS display_name,
       p.name,
       p.picture,
       n.created_at,
       n.updated_at
FROM nip05_identities n
JOIN profiles p ON p.public_key = n.public_key
`

func buildNIP05IdentitiesQuery(base string, query string) (string, []any) {
	query = strings.TrimSpace(query)
	if query == "" {
		return base, []any{}
	}

	needle := "%" + query + "%"
	return base + `
WHERE n.name ILIKE $1
   OR n.public_key ILIKE $1
   OR p.name ILIKE $1
   OR p.display_name ILIKE $1`, []any{needle}
}

func (q *Queries) ListNIP05Identities(ctx context.Context, query string, limit int, offset int) ([]NIP05Identity, int64, error) {
	countQuery, countArgs := buildNIP05IdentitiesQuery(`SELECT COUNT(*) FROM nip05_identities n JOIN profiles p ON p.public_key = n.public_key`, query)
	var total int64
	if err := q.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery, listArgs := buildNIP05IdentitiesQuery(listNIP05IdentitiesBase, query)
	listArgs = append(listArgs, limit, offset)
	listQuery += fmt.Sprintf(" ORDER BY n.name LIMIT $%d OFFSET $%d", len(listArgs)-1, len(listArgs))

	rows, err := q.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]NIP05Identity, 0, limit)
	for rows.Next() {
		var item NIP05Identity
		if err := rows.Scan(
			&item.Name,
			&item.PublicKey,
			&item.DisplayName,
			&item.ProfileName,
			&item.Picture,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	return items, total, nil
}

const listNIP05IdentitiesForDocument = `-- name: ListNIP05IdentitiesForDocument :many
SELECT name, public_key
FROM nip05_identities
ORDER BY name;
`

func (q *Queries) ListNIP05IdentitiesForDocument(ctx context.Context) ([]NIP05Identity, error) {
	rows, err := q.db.Query(ctx, listNIP05IdentitiesForDocument)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]NIP05Identity, 0, 128)
	for rows.Next() {
		var item NIP05Identity
		if err := rows.Scan(&item.Name, &item.PublicKey); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

const getLatestRelayListEventsByPubKeys = `-- name: GetLatestRelayListEventsByPubKeys :many
SELECT DISTINCT ON (pubkey)
       id,
       pubkey,
       created_at,
       kind,
       tags,
       content,
       sig
FROM event
WHERE kind = 10002
  AND pubkey = ANY($1::text[])
ORDER BY pubkey, created_at DESC, id DESC;
`

func (q *Queries) GetLatestRelayListEventsByPubKeys(ctx context.Context, pubkeys []string) (map[string]*nostr.Event, error) {
	result := make(map[string]*nostr.Event, len(pubkeys))
	if len(pubkeys) == 0 {
		return result, nil
	}

	rows, err := q.db.Query(ctx, getLatestRelayListEventsByPubKeys, pubkeys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var evt nostr.Event
		var createdAt int64
		if err := rows.Scan(
			&evt.ID,
			&evt.PubKey,
			&createdAt,
			&evt.Kind,
			&evt.Tags,
			&evt.Content,
			&evt.Sig,
		); err != nil {
			return nil, err
		}
		evt.CreatedAt = nostr.Timestamp(createdAt)
		result[evt.PubKey] = &evt
	}

	return result, nil
}
