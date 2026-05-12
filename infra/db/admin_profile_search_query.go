package db

import (
	"context"
	"fmt"
	"strings"
)

func buildAdminProfileSearchQuery(base string, query string) (string, []any) {
	query = strings.TrimSpace(query)
	if query == "" {
		return base, []any{}
	}
	if len(query) == 64 {
		return base + `
WHERE public_key = $1 OR public_key ILIKE $2 OR name ILIKE $2 OR display_name ILIKE $2 OR nip05 ILIKE $2`, []any{query, "%" + query + "%"}
	}
	needle := "%" + query + "%"
	return base + `
WHERE public_key ILIKE $1 OR name ILIKE $1 OR display_name ILIKE $1 OR nip05 ILIKE $1`, []any{needle}
}

func (q *Queries) SearchProfiles(ctx context.Context, query string, limit int, offset int) ([]Profile, int64, error) {
	countQuery, countArgs := buildAdminProfileSearchQuery(`SELECT COUNT(*) FROM profiles`, query)
	var total int64
	if err := q.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery, listArgs := buildAdminProfileSearchQuery(searchProfilesBase, query)
	listArgs = append(listArgs, limit, offset)
	listQuery += fmt.Sprintf(" ORDER BY COALESCE(display_name, name, public_key), public_key LIMIT $%d OFFSET $%d", len(listArgs)-1, len(listArgs))

	rows, err := q.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	profiles := make([]Profile, 0, limit)
	for rows.Next() {
		profile, scanErr := scanProfile(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		profiles = append(profiles, profile)
	}

	return profiles, total, rows.Err()
}
