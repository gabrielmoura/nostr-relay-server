package db

import "context"

const upsertBlockedIP = `
INSERT INTO nip86_blocked_ips (ip, reason, created_by, created_at, updated_at)
VALUES ($1::inet, $2::text, $3::varchar, NOW(), NOW())
ON CONFLICT (ip) DO UPDATE SET
    reason = EXCLUDED.reason,
    created_by = EXCLUDED.created_by,
    updated_at = NOW();
`

const deleteBlockedIP = `DELETE FROM nip86_blocked_ips WHERE ip = $1::inet`

const getBlockedIP = `
SELECT ip::text, COALESCE(reason, ''), created_by, created_at, updated_at
FROM nip86_blocked_ips
WHERE ip = $1::inet
LIMIT 1
`

const listBlockedIPs = `
SELECT ip::text, COALESCE(reason, ''), created_by, created_at, updated_at
FROM nip86_blocked_ips
ORDER BY updated_at DESC, ip ASC
`

func (q *Queries) UpsertBlockedIP(ctx context.Context, ip, reason, createdBy string) error {
	_, err := q.db.Exec(ctx, upsertBlockedIP, ip, reason, createdBy)
	return err
}

func (q *Queries) DeleteBlockedIP(ctx context.Context, ip string) error {
	_, err := q.db.Exec(ctx, deleteBlockedIP, ip)
	return err
}

func (q *Queries) GetBlockedIP(ctx context.Context, ip string) (NIP86IPRecord, bool, error) {
	var item NIP86IPRecord
	err := q.db.QueryRow(ctx, getBlockedIP, ip).Scan(
		&item.IP,
		&item.Reason,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if isNotFound(err) {
		return NIP86IPRecord{}, false, nil
	}
	if err != nil {
		return NIP86IPRecord{}, false, err
	}
	return item, true, nil
}

func (q *Queries) ListBlockedIPs(ctx context.Context) ([]NIP86IPRecord, error) {
	rows, err := q.db.Query(ctx, listBlockedIPs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]NIP86IPRecord, 0)
	for rows.Next() {
		var item NIP86IPRecord
		if err := rows.Scan(&item.IP, &item.Reason, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
