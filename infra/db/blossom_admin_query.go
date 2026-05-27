package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (q *Queries) UpdateBlossomObjectsReviewState(ctx context.Context, hashes []string, reviewState, flagReason string) (int64, error) {
	const statement = `
INSERT INTO blossom_objects_admin (hash, review_state, flag_reason, updated_at)
SELECT unnest($1::text[]), $2::text, $3::text, NOW()
ON CONFLICT (hash) DO UPDATE SET review_state = EXCLUDED.review_state, flag_reason = EXCLUDED.flag_reason, updated_at = NOW();`
	result, err := q.db.Exec(ctx, statement, hashes, reviewState, flagReason)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (q *Queries) RecordBlossomDownload(ctx context.Context, hash string, size int64, downloadedAt time.Time) error {
	const statement = `
INSERT INTO blossom_objects_admin (hash, ingress_bytes, last_downloaded_at, download_count, egress_bytes, updated_at)
VALUES ($1::text, 0, $2::timestamptz, 1, $3::bigint, NOW())
ON CONFLICT (hash) DO UPDATE SET
	last_downloaded_at = EXCLUDED.last_downloaded_at,
	download_count = blossom_objects_admin.download_count + 1,
	egress_bytes = blossom_objects_admin.egress_bytes + EXCLUDED.egress_bytes,
	updated_at = NOW();`
	_, err := q.db.Exec(ctx, statement, hash, downloadedAt, size)
	return err
}

func (q *Queries) UpdateBlossomObjectProcessing(ctx context.Context, hash, status, processingError string) error {
	const statement = `
INSERT INTO blossom_objects_admin (hash, processing_status, processing_error, updated_at)
VALUES ($1::text, COALESCE(NULLIF($2::text, ''), 'pending'), NULLIF($3::text, ''), NOW())
ON CONFLICT (hash) DO UPDATE SET
	processing_status = COALESCE(NULLIF(EXCLUDED.processing_status, ''), blossom_objects_admin.processing_status, 'pending'),
	processing_error = EXCLUDED.processing_error,
	updated_at = NOW();`
	_, err := q.db.Exec(ctx, statement, hash, status, processingError)
	return err
}

func (q *Queries) DeleteBlossomObjects(ctx context.Context, hashes []string) (int64, error) {
	result, err := q.db.Exec(ctx, `DELETE FROM objects WHERE hash = ANY($1::text[])`, hashes)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (q *Queries) UpdateObjectsBlockedState(ctx context.Context, hashes []string, blocked bool, reason string) error {
	_, err := q.db.Exec(ctx, `UPDATE objects SET blocked = $2::bool, blocked_by_reason = $3::text WHERE hash = ANY($1::text[])`, hashes, blocked, reason)
	return err
}

func (q *Queries) UpsertBlossomPubkeyQuota(ctx context.Context, arg UpsertBlossomPubkeyQuotaParams) error {
	const statement = `
INSERT INTO blossom_pubkey_quotas (pubkey, enabled, storage_quota_bytes, egress_quota_bytes, notes, created_by, created_at, updated_at)
VALUES ($1::text, $2::boolean, $3::bigint, $4::bigint, $5::text, $6::text, NOW(), NOW())
ON CONFLICT (pubkey) DO UPDATE SET
	enabled = EXCLUDED.enabled,
	storage_quota_bytes = EXCLUDED.storage_quota_bytes,
	egress_quota_bytes = EXCLUDED.egress_quota_bytes,
	notes = EXCLUDED.notes,
	created_by = EXCLUDED.created_by,
	updated_at = NOW();`
	_, err := q.db.Exec(ctx, statement, arg.Pubkey, arg.Enabled, arg.StorageQuotaBytes, arg.EgressQuotaBytes, arg.Notes, arg.CreatedBy)
	return err
}

func (q *Queries) ListBlossomUsers(ctx context.Context, query string, limit, offset int, sortBy, sortDir string) ([]BlossomUserUsageRow, int64, error) {
	whereSQL := "1=1"
	args := make([]any, 0, 2)
	if strings.TrimSpace(query) != "" {
		args = append(args, "%"+strings.TrimSpace(query)+"%")
		whereSQL = fmt.Sprintf("(o.public_key ILIKE $%d OR COALESCE(q.notes, '') ILIKE $%d OR COALESCE(p.name, '') ILIKE $%d OR COALESCE(p.display_name, '') ILIKE $%d OR COALESCE(p.nip05, '') ILIKE $%d)", len(args), len(args), len(args), len(args), len(args))
	}
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM (SELECT o.public_key FROM objects o LEFT JOIN blossom_pubkey_quotas q ON q.pubkey = o.public_key LEFT JOIN profiles p ON p.public_key = o.public_key WHERE %s GROUP BY o.public_key) AS users`, whereSQL)
	var total int64
	if err := q.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderSQL := "MAX(o.created_at) DESC, o.public_key ASC"
	sortDirSQL := "DESC"
	if strings.ToLower(sortDir) == "asc" {
		sortDirSQL = "ASC"
	}
	switch sortBy {
	case "storage_used_bytes":
		orderSQL = fmt.Sprintf("storage_used_bytes %s, o.public_key ASC", sortDirSQL)
	case "monthly_egress_bytes":
		orderSQL = fmt.Sprintf("monthly_egress %s, o.public_key ASC", sortDirSQL)
	case "object_count":
		orderSQL = fmt.Sprintf("object_count %s, o.public_key ASC", sortDirSQL)
	case "last_upload_at":
		orderSQL = fmt.Sprintf("last_upload_at %s, o.public_key ASC", sortDirSQL)
	case "pubkey":
		orderSQL = fmt.Sprintf("o.public_key %s", sortDirSQL)
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)
	querySQL := fmt.Sprintf(`
SELECT
	o.public_key,
	COUNT(*) AS object_count,
	COALESCE(SUM(COALESCE(o.size, 0)), 0) AS storage_used_bytes,
	bp.storage_quota_bytes,
	COALESCE(SUM(COALESCE(ba.egress_bytes, 0)), 0) AS monthly_egress,
	bp.egress_quota_bytes,
	COALESCE(q.enabled, FALSE) AS enabled,
	MAX(o.created_at) AS last_upload_at,
	COALESCE(q.notes, '') AS notes,
	p.name,
	p.display_name,
	p.picture
FROM objects o
LEFT JOIN blossom_objects_admin ba ON ba.hash = o.hash
LEFT JOIN blossom_pubkey_quotas q ON q.pubkey = o.public_key
LEFT JOIN blossom_plan_assignments pa ON pa.pubkey = o.public_key
LEFT JOIN blossom_plans bp ON bp.id = pa.plan_id
LEFT JOIN profiles p ON p.public_key = o.public_key
WHERE %s
GROUP BY o.public_key, q.enabled, bp.storage_quota_bytes, bp.egress_quota_bytes, q.notes, p.name, p.display_name, p.picture
ORDER BY %s
LIMIT $%d OFFSET $%d`, whereSQL, orderSQL, len(listArgs)-1, len(listArgs))

	rows, err := q.db.Query(ctx, querySQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]BlossomUserUsageRow, 0, limit)
	for rows.Next() {
		var item BlossomUserUsageRow
		if err := rows.Scan(&item.Pubkey, &item.ObjectCount, &item.StorageUsedBytes, &item.StorageQuotaBytes, &item.MonthlyEgress, &item.EgressQuotaBytes, &item.Enabled, &item.LastUploadAt, &item.Notes, &item.Name, &item.DisplayName, &item.Picture); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (q *Queries) GetBlossomUserUsage(ctx context.Context, pubkey string) (BlossomUserUsageRow, bool, error) {
	items, total, err := q.ListBlossomUsers(ctx, pubkey, 1, 0, "", "")
	if err != nil {
		return BlossomUserUsageRow{}, false, err
	}
	if total == 0 || len(items) == 0 || items[0].Pubkey != pubkey {
		return BlossomUserUsageRow{}, false, nil
	}
	return items[0], true, nil
}

func (q *Queries) DeleteObjectsByPubkey(ctx context.Context, pubkey string) (int64, error) {
	result, err := q.db.Exec(ctx, `DELETE FROM objects WHERE public_key = $1::text`, pubkey)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (q *Queries) GetBlossomQuota(ctx context.Context, pubkey string) (BlossomPubkeyQuota, bool, error) {
	const statement = `SELECT pubkey, enabled, storage_quota_bytes, egress_quota_bytes, COALESCE(notes, ''), created_by, created_at, updated_at FROM blossom_pubkey_quotas WHERE pubkey = $1::text LIMIT 1`
	var item BlossomPubkeyQuota
	err := q.db.QueryRow(ctx, statement, pubkey).Scan(&item.Pubkey, &item.Enabled, &item.StorageQuotaBytes, &item.EgressQuotaBytes, &item.Notes, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BlossomPubkeyQuota{}, false, nil
		}
		return BlossomPubkeyQuota{}, false, err
	}
	return item, true, nil
}

func (q *Queries) InsertBlossomAuditLog(ctx context.Context, record BlossomAuditRecord) error {
	const statement = `
INSERT INTO blossom_audit_log (actor_pubkey, action, target_type, target_id, request_id, payload, nostr_event_id, created_at)
VALUES ($1::text, $2::text, $3::text, $4::text, $5::text, $6::jsonb, $7::text, $8::timestamptz)`
	_, err := q.db.Exec(ctx, statement, record.ActorPubkey, record.Action, record.TargetType, record.TargetID, record.RequestID, defaultJSONBytes(record.Payload, []byte("{}")), nullableString(record.NostrEventID), record.CreatedAt)
	return err
}

func (q *Queries) ListBlossomAudit(ctx context.Context, query string, limit, offset int) ([]BlossomAuditRecord, int64, error) {
	whereSQL := "1=1"
	args := make([]any, 0, 2)
	if strings.TrimSpace(query) != "" {
		args = append(args, "%"+strings.TrimSpace(query)+"%")
		whereSQL = fmt.Sprintf("(action ILIKE $%d OR target_id ILIKE $%d OR target_type ILIKE $%d OR actor_pubkey ILIKE $%d OR COALESCE(request_id, '') ILIKE $%d)", len(args), len(args), len(args), len(args), len(args))
	}
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM blossom_audit_log WHERE %s", whereSQL)
	var total int64
	if err := q.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)
	listSQL := fmt.Sprintf(`SELECT id, actor_pubkey, action, target_type, target_id, COALESCE(request_id, ''), payload, COALESCE(nostr_event_id, ''), created_at FROM blossom_audit_log WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d`, whereSQL, len(listArgs)-1, len(listArgs))
	rows, err := q.db.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]BlossomAuditRecord, 0, limit)
	for rows.Next() {
		var item BlossomAuditRecord
		if err := rows.Scan(&item.ID, &item.ActorPubkey, &item.Action, &item.TargetType, &item.TargetID, &item.RequestID, &item.Payload, &item.NostrEventID, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableInt64(value sql.NullInt64) any {
	if !value.Valid {
		return nil
	}
	return value.Int64
}

func (q *Queries) GetBlossomServerPolicy(ctx context.Context) (BlossomServerPolicy, bool, error) {
	const statement = `
SELECT
	id,
	mode,
	default_storage_quota_bytes,
	default_egress_quota_bytes,
	enabled_user_default_storage_quota_bytes,
	enabled_user_default_egress_quota_bytes,
	updated_by,
	updated_at
FROM blossom_server_policy
WHERE id = 1
LIMIT 1`
	var item BlossomServerPolicy
	err := q.db.QueryRow(ctx, statement).Scan(
		&item.ID,
		&item.Mode,
		&item.DefaultStorageQuotaBytes,
		&item.DefaultEgressQuotaBytes,
		&item.EnabledUserDefaultStorageQuotaBytes,
		&item.EnabledUserDefaultEgressQuotaBytes,
		&item.UpdatedBy,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BlossomServerPolicy{}, false, nil
		}
		return BlossomServerPolicy{}, false, err
	}
	return item, true, nil
}

func (q *Queries) UpsertBlossomServerPolicy(ctx context.Context, policy BlossomServerPolicy) error {
	const statement = `
INSERT INTO blossom_server_policy (
	id,
	mode,
	default_storage_quota_bytes,
	default_egress_quota_bytes,
	enabled_user_default_storage_quota_bytes,
	enabled_user_default_egress_quota_bytes,
	updated_by,
	updated_at
)
VALUES (1, $1::text, $2::bigint, $3::bigint, $4::bigint, $5::bigint, $6::text, NOW())
ON CONFLICT (id) DO UPDATE SET
	mode = EXCLUDED.mode,
	default_storage_quota_bytes = COALESCE(EXCLUDED.default_storage_quota_bytes, blossom_server_policy.default_storage_quota_bytes),
	default_egress_quota_bytes = COALESCE(EXCLUDED.default_egress_quota_bytes, blossom_server_policy.default_egress_quota_bytes),
	enabled_user_default_storage_quota_bytes = COALESCE(EXCLUDED.enabled_user_default_storage_quota_bytes, blossom_server_policy.enabled_user_default_storage_quota_bytes),
	enabled_user_default_egress_quota_bytes = COALESCE(EXCLUDED.enabled_user_default_egress_quota_bytes, blossom_server_policy.enabled_user_default_egress_quota_bytes),
	updated_by = EXCLUDED.updated_by,
	updated_at = NOW()`
	_, err := q.db.Exec(ctx, statement,
		policy.Mode,
		nullableInt64(policy.DefaultStorageQuotaBytes),
		nullableInt64(policy.DefaultEgressQuotaBytes),
		nullableInt64(policy.EnabledUserDefaultStorageQuotaBytes),
		nullableInt64(policy.EnabledUserDefaultEgressQuotaBytes),
		policy.UpdatedBy,
	)
	return err
}
