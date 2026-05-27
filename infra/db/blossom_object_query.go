package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type BlossomObjectFilters struct {
	SHA256        string
	MIMEType      string
	Extension     string
	ReviewState   string
	Pubkey        string
	UploaderQuery string
	Query         string
}

type BlossomObjectRow struct {
	Hash             string
	UploaderPubkey   string
	MIMEType         string
	Extension        string
	Size             int64
	CreatedAt        time.Time
	Width            sql.NullInt32
	Height           sql.NullInt32
	DurationMS       sql.NullInt64
	BitrateKbps      sql.NullInt32
	Blurhash         string
	ThumbnailHash    string
	OptimizedHash    string
	HLSManifestHash  string
	ProcessingStatus string
	ProcessingError  string
	ReviewState      string
	ExifStatus       string
	GPSDetected      bool
	DownloadCount    int64
	LastDownloadedAt sql.NullTime
	IngressBytes     int64
	EgressBytes      int64
	FlagReason       string
	NIP94Tags        []byte
	Mirrors          []byte
}

type BlossomOverviewStats struct {
	TotalObjects     int64
	FlaggedObjects   int64
	PendingReview    int64
	MonthlyIngress   int64
	MonthlyEgress    int64
	ActiveUsers      int64
	WhitelistedUsers int64
	UsedBytes        int64
}

type BlossomServerPolicy struct {
	ID                                  int16
	Mode                                string
	DefaultStorageQuotaBytes            sql.NullInt64
	DefaultEgressQuotaBytes             sql.NullInt64
	EnabledUserDefaultStorageQuotaBytes sql.NullInt64
	EnabledUserDefaultEgressQuotaBytes  sql.NullInt64
	UpdatedBy                           string
	UpdatedAt                           time.Time
}

type BlossomUserUsageRow struct {
	Pubkey            string
	ObjectCount       int64
	StorageUsedBytes  int64
	StorageQuotaBytes sql.NullInt64
	MonthlyEgress     int64
	EgressQuotaBytes  sql.NullInt64
	Enabled           bool
	LastUploadAt      sql.NullTime
	Notes             string
	Name              sql.NullString
	DisplayName       sql.NullString
	Picture           sql.NullString
}

type UpsertBlossomObjectAdminParams struct {
	Hash             string
	Extension        string
	Width            *int32
	Height           *int32
	DurationMS       *int64
	BitrateKbps      *int32
	Blurhash         string
	ThumbnailHash    string
	OptimizedHash    string
	HLSManifestHash  string
	ProcessingStatus string
	ProcessingError  string
	ExifStatus       string
	GPSDetected      bool
	IngressBytes     int64
	ReviewState      string
	FlagReason       string
	NIP94Tags        []byte
	Mirrors          []byte
}

type UpsertBlossomPubkeyQuotaParams struct {
	Pubkey            string
	Enabled           bool
	StorageQuotaBytes *int64
	EgressQuotaBytes  *int64
	Notes             string
	CreatedBy         string
}

func (q *Queries) UpsertBlossomObjectAdmin(ctx context.Context, arg UpsertBlossomObjectAdminParams) error {
	const statement = `
INSERT INTO blossom_objects_admin (
	hash, extension, width, height, duration_ms, bitrate_kbps, blurhash, thumbnail_hash, optimized_hash,
	hls_manifest_hash, processing_status, processing_error,
	exif_status, gps_detected, ingress_bytes, review_state, flag_reason, nip94_tags, mirrors, updated_at
) VALUES (
	$1::text, $2::text, $3::integer, $4::integer, $5::bigint, $6::integer, $7::text, $8::text, $9::text,
	$10::text, COALESCE(NULLIF($11::text, ''), 'pending'), $12::text,
	COALESCE(NULLIF($13::text, ''), 'pending'), $14::boolean, $15::bigint, COALESCE(NULLIF($16::text, ''), 'ready'), $17::text, $18::jsonb, $19::jsonb, NOW()
)
ON CONFLICT (hash) DO UPDATE SET
	extension = CASE WHEN NULLIF($2::text, '') IS NULL THEN blossom_objects_admin.extension ELSE EXCLUDED.extension END,
	width = COALESCE(EXCLUDED.width, blossom_objects_admin.width),
	height = COALESCE(EXCLUDED.height, blossom_objects_admin.height),
	duration_ms = COALESCE(EXCLUDED.duration_ms, blossom_objects_admin.duration_ms),
	bitrate_kbps = COALESCE(EXCLUDED.bitrate_kbps, blossom_objects_admin.bitrate_kbps),
	blurhash = COALESCE(NULLIF(EXCLUDED.blurhash, ''), blossom_objects_admin.blurhash),
	thumbnail_hash = COALESCE(NULLIF(EXCLUDED.thumbnail_hash, ''), blossom_objects_admin.thumbnail_hash),
	optimized_hash = COALESCE(NULLIF(EXCLUDED.optimized_hash, ''), blossom_objects_admin.optimized_hash),
	hls_manifest_hash = COALESCE(NULLIF(EXCLUDED.hls_manifest_hash, ''), blossom_objects_admin.hls_manifest_hash),
	processing_status = COALESCE(NULLIF($11::text, ''), blossom_objects_admin.processing_status, 'pending'),
	processing_error = CASE WHEN NULLIF($12::text, '') IS NULL THEN blossom_objects_admin.processing_error ELSE EXCLUDED.processing_error END,
	exif_status = COALESCE(NULLIF($13::text, ''), blossom_objects_admin.exif_status, 'pending'),
	gps_detected = EXCLUDED.gps_detected,
	ingress_bytes = CASE WHEN EXCLUDED.ingress_bytes > 0 THEN EXCLUDED.ingress_bytes ELSE blossom_objects_admin.ingress_bytes END,
	review_state = COALESCE(NULLIF($16::text, ''), blossom_objects_admin.review_state, 'ready'),
	flag_reason = CASE WHEN NULLIF($17::text, '') IS NULL THEN blossom_objects_admin.flag_reason ELSE EXCLUDED.flag_reason END,
	nip94_tags = CASE WHEN EXCLUDED.nip94_tags = '[]'::jsonb THEN blossom_objects_admin.nip94_tags ELSE EXCLUDED.nip94_tags END,
	mirrors = CASE WHEN EXCLUDED.mirrors = '[]'::jsonb THEN blossom_objects_admin.mirrors ELSE EXCLUDED.mirrors END,
	updated_at = NOW();`

	_, err := q.db.Exec(
		ctx,
		statement,
		arg.Hash,
		arg.Extension,
		arg.Width,
		arg.Height,
		arg.DurationMS,
		arg.BitrateKbps,
		arg.Blurhash,
		arg.ThumbnailHash,
		arg.OptimizedHash,
		arg.HLSManifestHash,
		firstNonEmptyString(arg.ProcessingStatus, "pending"),
		arg.ProcessingError,
		firstNonEmptyString(arg.ExifStatus, "pending"),
		arg.GPSDetected,
		arg.IngressBytes,
		firstNonEmptyString(arg.ReviewState, "ready"),
		arg.FlagReason,
		defaultJSONBytes(arg.NIP94Tags, []byte("[]")),
		defaultJSONBytes(arg.Mirrors, []byte("[]")),
	)
	return err
}

func (q *Queries) ListBlossomObjects(ctx context.Context, filters BlossomObjectFilters, limit, offset int) ([]BlossomObjectRow, int64, error) {
	whereSQL, args := buildBlossomObjectsWhere(filters)
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM objects o LEFT JOIN blossom_objects_admin ba ON ba.hash = o.hash LEFT JOIN profiles p ON p.public_key = o.public_key WHERE %s", whereSQL)
	var total int64
	if err := q.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)
	listSQL := fmt.Sprintf(`
SELECT
	o.hash,
	o.public_key,
	COALESCE(o.mime_type, ''),
	COALESCE(ba.extension, ''),
	COALESCE(o.size, 0),
	o.created_at,
	ba.width,
	ba.height,
	ba.duration_ms,
	ba.bitrate_kbps,
	COALESCE(ba.blurhash, ''),
	COALESCE(ba.thumbnail_hash, ''),
	COALESCE(ba.optimized_hash, ''),
	COALESCE(ba.hls_manifest_hash, ''),
	COALESCE(ba.processing_status, 'pending'),
	COALESCE(ba.processing_error, ''),
	COALESCE(ba.review_state, 'ready'),
	COALESCE(ba.exif_status, 'pending'),
	COALESCE(ba.gps_detected, false),
	COALESCE(ba.download_count, 0),
	ba.last_downloaded_at,
	COALESCE(ba.ingress_bytes, COALESCE(o.size, 0)),
	COALESCE(ba.egress_bytes, 0),
	COALESCE(ba.flag_reason, ''),
	COALESCE(ba.nip94_tags, '[]'::jsonb),
	COALESCE(ba.mirrors, '[]'::jsonb)
FROM objects o
LEFT JOIN blossom_objects_admin ba ON ba.hash = o.hash
LEFT JOIN profiles p ON p.public_key = o.public_key
WHERE %s
ORDER BY o.created_at DESC, o.hash DESC
LIMIT $%d OFFSET $%d`, whereSQL, len(listArgs)-1, len(listArgs))

	rows, err := q.db.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]BlossomObjectRow, 0, limit)
	for rows.Next() {
		var item BlossomObjectRow
		if err := rows.Scan(
			&item.Hash,
			&item.UploaderPubkey,
			&item.MIMEType,
			&item.Extension,
			&item.Size,
			&item.CreatedAt,
			&item.Width,
			&item.Height,
			&item.DurationMS,
			&item.BitrateKbps,
			&item.Blurhash,
			&item.ThumbnailHash,
			&item.OptimizedHash,
			&item.HLSManifestHash,
			&item.ProcessingStatus,
			&item.ProcessingError,
			&item.ReviewState,
			&item.ExifStatus,
			&item.GPSDetected,
			&item.DownloadCount,
			&item.LastDownloadedAt,
			&item.IngressBytes,
			&item.EgressBytes,
			&item.FlagReason,
			&item.NIP94Tags,
			&item.Mirrors,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	return items, total, rows.Err()

}

func (q *Queries) GetBlossomObject(ctx context.Context, hash string) (BlossomObjectRow, bool, error) {
	items, total, err := q.ListBlossomObjects(ctx, BlossomObjectFilters{SHA256: hash}, 1, 0)
	if err != nil {
		return BlossomObjectRow{}, false, err
	}
	if total == 0 || len(items) == 0 {
		return BlossomObjectRow{}, false, nil
	}
	return items[0], true, nil
}

func (q *Queries) GetBlossomOverviewStats(ctx context.Context) (BlossomOverviewStats, error) {
	const statement = `
SELECT
	COUNT(*) AS total_objects,
	COUNT(*) FILTER (WHERE COALESCE(ba.review_state, 'ready') = 'flagged') AS flagged_objects,
	COUNT(*) FILTER (WHERE COALESCE(ba.review_state, 'ready') = 'pending_review') AS pending_review,
	COALESCE(SUM(COALESCE(ba.ingress_bytes, o.size)), 0) AS monthly_ingress,
	COALESCE(SUM(COALESCE(ba.egress_bytes, 0)), 0) AS monthly_egress,
	COUNT(DISTINCT o.public_key) AS active_users,
	(SELECT COUNT(*) FROM blossom_pubkey_quotas WHERE enabled = TRUE) AS whitelisted_users,
	COALESCE(SUM(o.size), 0) AS used_bytes
FROM objects o
LEFT JOIN blossom_objects_admin ba ON ba.hash = o.hash;`

	var stats BlossomOverviewStats
	err := q.db.QueryRow(ctx, statement).Scan(
		&stats.TotalObjects,
		&stats.FlaggedObjects,
		&stats.PendingReview,
		&stats.MonthlyIngress,
		&stats.MonthlyEgress,
		&stats.ActiveUsers,
		&stats.WhitelistedUsers,
		&stats.UsedBytes,
	)
	return stats, err
}

func buildBlossomObjectsWhere(filters BlossomObjectFilters) (string, []any) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 8)
	if filters.SHA256 != "" {
		args = append(args, strings.TrimSpace(filters.SHA256))
		clauses = append(clauses, fmt.Sprintf("o.hash = $%d", len(args)))
	}
	if filters.MIMEType != "" {
		args = append(args, strings.TrimSpace(filters.MIMEType))
		clauses = append(clauses, fmt.Sprintf("o.mime_type = $%d", len(args)))
	}
	if filters.Extension != "" {
		args = append(args, strings.TrimSpace(filters.Extension))
		clauses = append(clauses, fmt.Sprintf("COALESCE(ba.extension, '') = $%d", len(args)))
	}
	if filters.ReviewState != "" {
		args = append(args, strings.TrimSpace(filters.ReviewState))
		clauses = append(clauses, fmt.Sprintf("COALESCE(ba.review_state, 'ready') = $%d", len(args)))
	}
	if filters.Pubkey != "" {
		args = append(args, strings.TrimSpace(filters.Pubkey))
		clauses = append(clauses, fmt.Sprintf("o.public_key = $%d", len(args)))
	}
	if filters.UploaderQuery != "" {
		args = append(args, "%"+strings.TrimSpace(filters.UploaderQuery)+"%")
		idx := len(args)
		clauses = append(clauses, fmt.Sprintf("(o.public_key ILIKE $%d OR COALESCE(p.name, '') ILIKE $%d OR COALESCE(p.display_name, '') ILIKE $%d OR COALESCE(p.nip05, '') ILIKE $%d)", idx, idx, idx, idx))
	}
	if filters.Query != "" {
		args = append(args, "%"+strings.TrimSpace(filters.Query)+"%")
		idx := len(args)
		clauses = append(clauses, fmt.Sprintf("(o.hash ILIKE $%d OR o.public_key ILIKE $%d OR COALESCE(o.mime_type, '') ILIKE $%d OR COALESCE(ba.review_state, 'ready') ILIKE $%d OR COALESCE(p.name, '') ILIKE $%d OR COALESCE(p.display_name, '') ILIKE $%d OR COALESCE(p.nip05, '') ILIKE $%d)", idx, idx, idx, idx, idx, idx, idx))
	}
	return strings.Join(clauses, " AND "), args
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func defaultJSONBytes(value []byte, fallback []byte) []byte {
	if len(value) == 0 {
		return fallback
	}
	return value
}

func decodeJSONMap(payload []byte) map[string]string {
	if len(payload) == 0 {
		return map[string]string{}
	}
	values := make(map[string]string)
	if err := json.Unmarshal(payload, &values); err != nil {
		return map[string]string{}
	}
	return values
}

func decodeJSONStringSlice(payload []byte) []string {
	if len(payload) == 0 {
		return []string{}
	}
	values := make([]string, 0)
	if err := json.Unmarshal(payload, &values); err != nil {
		return []string{}
	}
	return values
}
