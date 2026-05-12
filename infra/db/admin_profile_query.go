package db

import (
	"context"

	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
)

const getProfileByPublicKey = `-- name: GetProfileByPublicKey :one
SELECT id, public_key, name, about, picture, banner, website, display_name, lud16, pronouns, nip05, bot
FROM profiles
WHERE public_key = $1::text
LIMIT 1;
`

const getProfilesByPublicKeys = `-- name: GetProfilesByPublicKeys :many
SELECT id, public_key, name, about, picture, banner, website, display_name, lud16, pronouns, nip05, bot
FROM profiles
WHERE public_key = ANY($1::text[]);
`

const searchProfilesBase = `
SELECT id, public_key, name, about, picture, banner, website, display_name, lud16, pronouns, nip05, bot
FROM profiles
`

func (q *Queries) GetProfileByPublicKey(ctx context.Context, key string) (Profile, error) {
	if cached, ok := cache.GetProfile(key); ok {
		return Profile{PublicKey: key, Name: cached.Name, DisplayName: cached.DisplayName, About: cached.About, Picture: cached.Picture, Website: cached.Website, Nip05: cached.NIP05, Lud16: cached.LUD16, Bot: cached.Bot}, nil
	}

	profile, err := scanProfile(q.db.QueryRow(ctx, getProfileByPublicKey, key))
	if err != nil {
		return Profile{}, err
	}
	_ = cache.SetProfile(key, &cache.ProfileCache{Name: profile.Name, DisplayName: profile.DisplayName, About: profile.About, Picture: profile.Picture, Website: profile.Website, NIP05: profile.Nip05, LUD16: profile.Lud16, Bot: profile.Bot})
	return profile, nil
}

func (q *Queries) GetProfilesByPublicKeys(ctx context.Context, keys []string) (map[string]Profile, error) {
	profiles := make(map[string]Profile, len(keys))
	if len(keys) == 0 {
		return profiles, nil
	}

	rows, err := q.db.Query(ctx, getProfilesByPublicKeys, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		profile, scanErr := scanProfile(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		profiles[profile.PublicKey] = profile
	}

	return profiles, rows.Err()
}
