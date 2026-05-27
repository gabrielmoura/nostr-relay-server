package db

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (q *Queries) InsertBlossomReviewReports(ctx context.Context, eventID string, reporterPubkey string, targetEventID string, targetPubkey string, reportType string, reason string, hashes []string, createdAt time.Time) error {
	if len(hashes) == 0 {
		return nil
	}
	const statement = `
INSERT INTO blossom_review_reports (event_id, object_hash, reporter_pubkey, target_event_id, target_pubkey, report_type, reason, status, created_at)
SELECT $1::text, unnest($2::text[]), $3::text, NULLIF($4::text, ''), NULLIF($5::text, ''), NULLIF($6::text, ''), NULLIF($7::text, ''), 'open', $8::timestamptz
ON CONFLICT DO NOTHING`
	_, err := q.db.Exec(ctx, statement, eventID, hashes, reporterPubkey, targetEventID, targetPubkey, reportType, reason, createdAt)
	return err
}

func blossomReportsWhere(filters BlossomReportFilters) (string, []any) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 4)
	if value := strings.TrimSpace(filters.ObjectHash); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("r.object_hash = $%d", len(args)))
	}
	if value := strings.TrimSpace(filters.ReportType); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("COALESCE(r.report_type, '') = $%d", len(args)))
	}
	if value := strings.TrimSpace(filters.Status); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("r.status = $%d", len(args)))
	}
	if value := strings.TrimSpace(filters.Query); value != "" {
		args = append(args, "%"+value+"%")
		idx := len(args)
		clauses = append(clauses, fmt.Sprintf("(r.object_hash ILIKE $%d OR r.reporter_pubkey ILIKE $%d OR COALESCE(r.target_event_id, '') ILIKE $%d OR COALESCE(r.target_pubkey, '') ILIKE $%d OR COALESCE(r.reason, '') ILIKE $%d)", idx, idx, idx, idx, idx))
	}
	return strings.Join(clauses, " AND "), args
}

func (q *Queries) ListBlossomReports(ctx context.Context, filters BlossomReportFilters, limit int, offset int) ([]BlossomReportRow, int64, error) {
	whereSQL, args := blossomReportsWhere(filters)
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM blossom_review_reports r WHERE %s", whereSQL)
	var total int64
	if err := q.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)
	listSQL := fmt.Sprintf(`SELECT r.id, r.event_id, r.object_hash, r.reporter_pubkey, COALESCE(r.target_event_id, ''), COALESCE(r.target_pubkey, ''), COALESCE(r.report_type, ''), COALESCE(r.reason, ''), r.status, COALESCE(r.resolved_by, ''), COALESCE(r.resolved_note, ''), r.created_at, r.resolved_at FROM blossom_review_reports r WHERE %s ORDER BY r.created_at DESC, r.id DESC LIMIT $%d OFFSET $%d`, whereSQL, len(listArgs)-1, len(listArgs))
	rows, err := q.db.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]BlossomReportRow, 0, limit)
	for rows.Next() {
		var item BlossomReportRow
		if err := rows.Scan(&item.ID, &item.EventID, &item.ObjectHash, &item.ReporterPubkey, &item.TargetEventID, &item.TargetPubkey, &item.ReportType, &item.Reason, &item.Status, &item.ResolvedBy, &item.ResolvedNote, &item.CreatedAt, &item.ResolvedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (q *Queries) ResolveBlossomReport(ctx context.Context, id int64, status string, resolvedBy string, note string) error {
	_, err := q.db.Exec(ctx, `UPDATE blossom_review_reports SET status = $2::text, resolved_by = NULLIF($3::text, ''), resolved_note = NULLIF($4::text, ''), resolved_at = NOW() WHERE id = $1::bigint`, id, status, resolvedBy, note)
	return err
}

func (q *Queries) GetBlossomReportSummary(ctx context.Context, filters BlossomReportFilters) (BlossomReportSummary, error) {
	whereSQL, args := blossomReportsWhere(filters)
	summary := BlossomReportSummary{ByType: []BlossomCountByValue{}, ByStatus: []BlossomCountByValue{}}
	if err := q.db.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'open'), COUNT(*) FILTER (WHERE status <> 'open') FROM blossom_review_reports r WHERE %s`, whereSQL), args...).Scan(&summary.TotalReports, &summary.OpenReports, &summary.ResolvedReports); err != nil {
		return BlossomReportSummary{}, err
	}
	typeRows, err := q.db.Query(ctx, fmt.Sprintf(`SELECT COALESCE(report_type, 'unknown') AS name, COUNT(*) AS count FROM blossom_review_reports r WHERE %s GROUP BY COALESCE(report_type, 'unknown') ORDER BY count DESC, name ASC LIMIT 8`, whereSQL), args...)
	if err != nil {
		return BlossomReportSummary{}, err
	}
	for typeRows.Next() {
		var item BlossomCountByValue
		if err := typeRows.Scan(&item.Name, &item.Count); err != nil {
			typeRows.Close()
			return BlossomReportSummary{}, err
		}
		summary.ByType = append(summary.ByType, item)
	}
	typeRows.Close()
	statusRows, err := q.db.Query(ctx, fmt.Sprintf(`SELECT status, COUNT(*) AS count FROM blossom_review_reports r WHERE %s GROUP BY status ORDER BY count DESC, status ASC`, whereSQL), args...)
	if err != nil {
		return BlossomReportSummary{}, err
	}
	for statusRows.Next() {
		var item BlossomCountByValue
		if err := statusRows.Scan(&item.Name, &item.Count); err != nil {
			statusRows.Close()
			return BlossomReportSummary{}, err
		}
		summary.ByStatus = append(summary.ByStatus, item)
	}
	statusRows.Close()
	return summary, nil
}

func (q *Queries) CountOpenBlossomReportsByHash(ctx context.Context, hash string) (int64, error) {
	var count int64
	err := q.db.QueryRow(ctx, `SELECT COUNT(*) FROM blossom_review_reports WHERE object_hash = $1::text AND status = 'open'`, hash).Scan(&count)
	return count, err
}
