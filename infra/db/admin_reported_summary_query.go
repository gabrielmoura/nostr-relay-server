package db

import (
	"context"
)

func (q *Queries) loadReportedSummaryTotals(ctx context.Context, groupedSQL string, groupedArgs []any, summary *ReportedEventsSummary) error {
	summarySQL := `WITH grouped_reports AS (` + groupedSQL + `
GROUP BY target_event_id
)
SELECT COUNT(*) AS total_events,
  COALESCE(SUM(report_count), 0) AS total_reports,
  COUNT(DISTINCT NULLIF(target_pubkey, '')) AS unique_target_authors
FROM grouped_reports;`

	return q.db.QueryRow(ctx, summarySQL, groupedArgs...).Scan(&summary.TotalEvents, &summary.TotalReports, &summary.UniqueTargetAuthors)
}

func (q *Queries) loadReportedSummaryTimeline(ctx context.Context, groupedSQL string, groupedArgs []any, summary *ReportedEventsSummary) error {
	timelineSQL := `WITH grouped_reports AS (` + groupedSQL + `
GROUP BY target_event_id
)
SELECT to_char(to_timestamp(last_reported), 'YYYY-MM-DD') AS bucket, COALESCE(SUM(report_count), 0) AS count
FROM grouped_reports
GROUP BY bucket
ORDER BY bucket ASC;`

	rows, err := q.db.Query(ctx, timelineSQL, groupedArgs...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item ReportedTimelinePoint
		if err := rows.Scan(&item.Bucket, &item.Count); err != nil {
			return err
		}
		summary.Timeline = append(summary.Timeline, item)
	}

	return rows.Err()
}

func (q *Queries) loadReportedSummaryTypes(ctx context.Context, rawSQL string, rawArgs []any, summary *ReportedEventsSummary) error {
	typesSQL := `WITH filtered_reports AS (` + rawSQL + `
)
SELECT report_type AS name, COUNT(*) AS count
FROM filtered_reports
WHERE report_type IS NOT NULL AND report_type <> ''
GROUP BY report_type
ORDER BY count DESC, name ASC;`

	rows, err := q.db.Query(ctx, typesSQL, rawArgs...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item ReportedTypeCount
		if err := rows.Scan(&item.Name, &item.Count); err != nil {
			return err
		}
		summary.ReportTypes = append(summary.ReportTypes, item)
	}

	return rows.Err()
}

func (q *Queries) loadReportedSummaryAuthors(ctx context.Context, rawSQL string, rawArgs []any, summary *ReportedEventsSummary) error {
	authorsSQL := `WITH filtered_reports AS (` + rawSQL + `
)
SELECT target_pubkey AS pubkey, COUNT(*) AS count
FROM filtered_reports
WHERE target_pubkey IS NOT NULL AND target_pubkey <> ''
GROUP BY target_pubkey
ORDER BY count DESC, pubkey ASC
LIMIT 8;`

	rows, err := q.db.Query(ctx, authorsSQL, rawArgs...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item ReportedAuthorCount
		if err := rows.Scan(&item.Pubkey, &item.Count); err != nil {
			return err
		}
		summary.TopAuthors = append(summary.TopAuthors, item)
	}

	return rows.Err()
}

func (q *Queries) loadReportedSummaryTargets(ctx context.Context, groupedSQL string, groupedArgs []any, summary *ReportedEventsSummary) error {
	targetsSQL := `WITH grouped_reports AS (` + groupedSQL + `
GROUP BY target_event_id
)
SELECT target_event_id, report_count AS count
FROM grouped_reports
ORDER BY report_count DESC, target_event_id ASC
LIMIT 8;`

	rows, err := q.db.Query(ctx, targetsSQL, groupedArgs...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item ReportedTargetCount
		if err := rows.Scan(&item.TargetEventID, &item.Count); err != nil {
			return err
		}
		summary.TopTargets = append(summary.TopTargets, item)
	}

	return rows.Err()
}
