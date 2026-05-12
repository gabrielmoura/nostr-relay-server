package db

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

var reportTypeValidator = regexp.MustCompile(`^(nudity|malware|profanity|illegal|spam|impersonation|other)$`)

const getReportedEventsRawBase = `
SELECT e.id, e.content, e.created_at,
  (SELECT tag->>1 FROM jsonb_array_elements(e.tags) tag WHERE tag->>0 = 'e' LIMIT 1) AS target_event_id,
  (SELECT tag->>1 FROM jsonb_array_elements(e.tags) tag WHERE tag->>0 = 'p' LIMIT 1) AS target_pubkey,
  (SELECT tag->>2 FROM jsonb_array_elements(e.tags) tag WHERE tag->>0 IN ('e', 'p', 'x') AND COALESCE(tag->>2, '') <> '' LIMIT 1) AS report_type
FROM event e
WHERE e.kind = 1984
`

const getReportedEventsFilteredBase = `
SELECT id, content, created_at, target_event_id, target_pubkey, report_type
FROM (
` + getReportedEventsRawBase + `
) raw_reports
WHERE target_event_id IS NOT NULL
`

const getReportedEventsListBase = `
SELECT target_event_id,
  COALESCE(MAX(target_pubkey) FILTER (WHERE target_pubkey IS NOT NULL), '') AS target_pubkey,
  COUNT(*) AS report_count,
  MAX(created_at) AS last_reported,
  ARRAY_REMOVE(ARRAY_AGG(DISTINCT report_type), NULL) AS report_types
FROM (
` + getReportedEventsFilteredBase + `
) filtered_reports
WHERE target_event_id IS NOT NULL
`

func appendReportedEventsFilters(base string, args []any, filters ReportedEventsFilters) (string, []any) {
	if query := strings.TrimSpace(filters.Query); query != "" {
		args = append(args, "%"+query+"%")
		idx := len(args)
		base += fmt.Sprintf(" AND (target_event_id ILIKE $%d OR target_pubkey ILIKE $%d OR content ILIKE $%d)", idx, idx, idx)
	}
	if reportTypeValidator.MatchString(filters.ReportType) {
		args = append(args, filters.ReportType)
		base += fmt.Sprintf(" AND report_type = $%d", len(args))
	}
	if value := strings.TrimSpace(filters.TargetPubkey); value != "" {
		args = append(args, value)
		base += fmt.Sprintf(" AND target_pubkey = $%d", len(args))
	}
	if value := strings.TrimSpace(filters.TargetEventID); value != "" {
		args = append(args, value)
		base += fmt.Sprintf(" AND target_event_id = $%d", len(args))
	}
	if filters.Since > 0 {
		args = append(args, filters.Since)
		base += fmt.Sprintf(" AND created_at >= $%d", len(args))
	}
	if filters.Until > 0 {
		args = append(args, filters.Until)
		base += fmt.Sprintf(" AND created_at <= $%d", len(args))
	}
	return base, args
}

func (q *Queries) GetReportedEvents(ctx context.Context, filters ReportedEventsFilters, limit int, offset int) ([]ReportedEventSummary, int64, error) {
	countSQL, countArgs := appendReportedEventsFilters(getReportedEventsFilteredBase, []any{}, filters)
	countSQL = `SELECT COUNT(DISTINCT target_event_id) FROM (` + countSQL + `
) grouped_reports;`

	var total int64
	if err := q.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listSQL, listArgs := appendReportedEventsFilters(getReportedEventsListBase, []any{}, filters)
	listArgs = append(listArgs, limit, offset)
	listSQL += fmt.Sprintf(`
GROUP BY target_event_id
ORDER BY last_reported DESC
LIMIT $%d OFFSET $%d;`, len(listArgs)-1, len(listArgs))

	rows, err := q.db.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]ReportedEventSummary, 0, limit)
	for rows.Next() {
		var item ReportedEventSummary
		if err := rows.Scan(&item.TargetEventID, &item.TargetPubkey, &item.ReportCount, &item.LastReported, &item.ReportTypes); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	return items, total, rows.Err()
}

func (q *Queries) GetReportedEventsSummary(ctx context.Context, filters ReportedEventsFilters) (ReportedEventsSummary, error) {
	groupedSQL, groupedArgs := appendReportedEventsFilters(getReportedEventsListBase, []any{}, filters)
	rawSQL, rawArgs := appendReportedEventsFilters(getReportedEventsFilteredBase, []any{}, filters)
	summary := ReportedEventsSummary{}

	if err := q.loadReportedSummaryTotals(ctx, groupedSQL, groupedArgs, &summary); err != nil {
		return ReportedEventsSummary{}, err
	}
	if err := q.loadReportedSummaryTimeline(ctx, groupedSQL, groupedArgs, &summary); err != nil {
		return ReportedEventsSummary{}, err
	}
	if err := q.loadReportedSummaryTypes(ctx, rawSQL, rawArgs, &summary); err != nil {
		return ReportedEventsSummary{}, err
	}
	if err := q.loadReportedSummaryAuthors(ctx, rawSQL, rawArgs, &summary); err != nil {
		return ReportedEventsSummary{}, err
	}
	if err := q.loadReportedSummaryTargets(ctx, groupedSQL, groupedArgs, &summary); err != nil {
		return ReportedEventsSummary{}, err
	}

	return summary, nil
}
