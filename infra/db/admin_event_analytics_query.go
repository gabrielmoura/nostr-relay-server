package db

import (
	"context"
	"fmt"
)

func (q *Queries) loadEventKindAggregates(ctx context.Context, whereClause string, params []any, aggregates *EventAggregates) error {
	kindsSQL := fmt.Sprintf(`
SELECT kind, COUNT(*) AS count
FROM event
WHERE %s
GROUP BY kind
ORDER BY count DESC, kind ASC
LIMIT 8;`, whereClause)

	rows, err := q.db.Query(ctx, kindsSQL, params...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item EventKindAggregate
		if err := rows.Scan(&item.Kind, &item.Count); err != nil {
			return err
		}
		aggregates.Kinds = append(aggregates.Kinds, item)
	}

	return rows.Err()
}

func (q *Queries) loadEventAuthorAggregates(ctx context.Context, whereClause string, params []any, aggregates *EventAggregates) error {
	authorsSQL := fmt.Sprintf(`
SELECT event.pubkey,
       COALESCE(NULLIF(profiles.display_name, ''), NULLIF(profiles.name, ''), event.pubkey) AS display_name,
       COUNT(*) AS count
FROM event
LEFT JOIN profiles ON profiles.public_key = event.pubkey
WHERE %s
GROUP BY event.pubkey, profiles.display_name, profiles.name
ORDER BY count DESC, event.pubkey ASC
LIMIT 8;`, whereClause)

	rows, err := q.db.Query(ctx, authorsSQL, params...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item EventAuthorAggregate
		if err := rows.Scan(&item.Pubkey, &item.DisplayName, &item.Count); err != nil {
			return err
		}
		aggregates.TopAuthors = append(aggregates.TopAuthors, item)
	}

	return rows.Err()
}

func (q *Queries) loadEventTagAggregates(ctx context.Context, whereClause string, params []any, aggregates *EventAggregates) error {
	tagsSQL := fmt.Sprintf(`
SELECT LOWER(BTRIM(tag->>1)) AS tag, COUNT(*) AS count
FROM event
CROSS JOIN LATERAL jsonb_array_elements(tags) AS tag
WHERE %s
  AND jsonb_typeof(tag) = 'array'
  AND jsonb_array_length(tag) >= 2
  AND tag->>0 = 't'
  AND BTRIM(tag->>1) <> ''
GROUP BY LOWER(BTRIM(tag->>1))
ORDER BY count DESC, tag ASC
LIMIT 12;`, whereClause)

	rows, err := q.db.Query(ctx, tagsSQL, params...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item EventTagAggregate
		if err := rows.Scan(&item.Tag, &item.Count); err != nil {
			return err
		}
		aggregates.TopTags = append(aggregates.TopTags, item)
	}

	return rows.Err()
}

func (q *Queries) loadEventTrends(ctx context.Context, whereClause string, params []any, aggregates *EventAggregates) error {
	trendsSQL := fmt.Sprintf(`
WITH filtered AS (
    SELECT created_at, tags
    FROM event
    WHERE %s
),
month_tags AS (
    SELECT to_char(to_timestamp(created_at), 'YYYY-MM') AS period,
           LOWER(BTRIM(tag->>1)) AS tag,
           COUNT(*) AS count
    FROM filtered
    CROSS JOIN LATERAL jsonb_array_elements(tags) AS tag
    WHERE jsonb_typeof(tag) = 'array' AND jsonb_array_length(tag) >= 2 AND tag->>0 = 't' AND BTRIM(tag->>1) <> ''
    GROUP BY period, LOWER(BTRIM(tag->>1))
    ORDER BY count DESC, period ASC, tag ASC
    LIMIT 1
),
year_tags AS (
    SELECT to_char(to_timestamp(created_at), 'YYYY') AS period,
           LOWER(BTRIM(tag->>1)) AS tag,
           COUNT(*) AS count
    FROM filtered
    CROSS JOIN LATERAL jsonb_array_elements(tags) AS tag
    WHERE jsonb_typeof(tag) = 'array' AND jsonb_array_length(tag) >= 2 AND tag->>0 = 't' AND BTRIM(tag->>1) <> ''
    GROUP BY period, LOWER(BTRIM(tag->>1))
    ORDER BY count DESC, period ASC, tag ASC
    LIMIT 1
),
month_counts AS (
    SELECT to_char(to_timestamp(created_at), 'YYYY-MM') AS period, COUNT(*) AS count
    FROM filtered GROUP BY period ORDER BY count DESC, period ASC LIMIT 1
),
year_counts AS (
    SELECT to_char(to_timestamp(created_at), 'YYYY') AS period, COUNT(*) AS count
    FROM filtered GROUP BY period ORDER BY count DESC, period ASC LIMIT 1
)
SELECT COALESCE((SELECT period || ' · ' || tag FROM month_tags), ''),
       COALESCE((SELECT count FROM month_tags), 0),
       COALESCE((SELECT period || ' · ' || tag FROM year_tags), ''),
       COALESCE((SELECT count FROM year_tags), 0),
       COALESCE((SELECT period FROM month_counts), ''),
       COALESCE((SELECT count FROM month_counts), 0),
       COALESCE((SELECT period FROM year_counts), ''),
       COALESCE((SELECT count FROM year_counts), 0);`, whereClause)

	return q.db.QueryRow(ctx, trendsSQL, params...).Scan(
		&aggregates.Trends.TopTagMonth,
		&aggregates.Trends.TopTagMonthCount,
		&aggregates.Trends.TopTagYear,
		&aggregates.Trends.TopTagYearCount,
		&aggregates.Trends.PeakMonth,
		&aggregates.Trends.PeakMonthCount,
		&aggregates.Trends.PeakYear,
		&aggregates.Trends.PeakYearCount,
	)
}
