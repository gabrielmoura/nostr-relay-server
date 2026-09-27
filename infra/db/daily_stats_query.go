package db

import (
	"context"
	"fmt"
)

const (
	dailyStatsTopKindsLimit   = 10
	dailyStatsTopAuthorsLimit = 10
	dailyStatsTopTagsLimit    = 20
)

type DailyStatsAggregate struct {
	TotalEvents        int64
	UniqueAuthors      int64
	UniqueKinds        int64
	EventsWithHashtags int64
	TopKinds           []DailyStatsKind
	TopAuthors         []DailyStatsAuthor
	TopTags            []DailyStatsTag
}

type DailyStatsKind struct {
	Kind  int   `json:"kind"`
	Count int64 `json:"count"`
}

type DailyStatsAuthor struct {
	Pubkey string `json:"pubkey"`
	Count  int64  `json:"count"`
}

type DailyStatsTag struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

const dailyStatsSummarySQL = `
SELECT
  COUNT(*) AS total_events,
  COUNT(DISTINCT pubkey) AS unique_authors,
  COUNT(DISTINCT kind) AS unique_kinds,
  COUNT(*) FILTER (
    WHERE EXISTS (
      SELECT 1
      FROM jsonb_array_elements(tags) AS tag
      WHERE jsonb_typeof(tag) = 'array'
        AND jsonb_array_length(tag) >= 2
        AND tag->>0 = 't'
        AND BTRIM(tag->>1) <> ''
    )
  ) AS events_with_hashtags
FROM event
WHERE created_at >= $1::bigint
  AND created_at < $2::bigint
  AND deleted_by IS NULL;`

const dailyStatsTopKindsSQL = `
SELECT kind, COUNT(*) AS count
FROM event
WHERE created_at >= $1::bigint
  AND created_at < $2::bigint
  AND deleted_by IS NULL
GROUP BY kind
ORDER BY count DESC, kind ASC
LIMIT 10;`

const dailyStatsTopAuthorsSQL = `
SELECT pubkey, COUNT(*) AS count
FROM event
WHERE created_at >= $1::bigint
  AND created_at < $2::bigint
  AND deleted_by IS NULL
GROUP BY pubkey
ORDER BY count DESC, pubkey ASC
LIMIT 10;`

const dailyStatsTopTagsSQL = `
WITH normalized_tags AS (
  SELECT DISTINCT event.id, LOWER(BTRIM(tag->>1)) AS tag
  FROM event
  CROSS JOIN LATERAL jsonb_array_elements(tags) AS tag
  WHERE created_at >= $1::bigint
    AND created_at < $2::bigint
    AND deleted_by IS NULL
    AND jsonb_typeof(tag) = 'array'
    AND jsonb_array_length(tag) >= 2
    AND tag->>0 = 't'
    AND BTRIM(tag->>1) <> ''
)
SELECT tag, COUNT(*) AS count
FROM normalized_tags
GROUP BY tag
ORDER BY count DESC, tag ASC
LIMIT 20;`

func (q *Queries) CollectDailyStats(ctx context.Context, startUnix, endUnix int64) (DailyStatsAggregate, error) {
	if endUnix <= startUnix {
		return DailyStatsAggregate{}, fmt.Errorf("daily stats window end must be after start")
	}

	aggregate := DailyStatsAggregate{
		TopKinds:   []DailyStatsKind{},
		TopAuthors: []DailyStatsAuthor{},
		TopTags:    []DailyStatsTag{},
	}
	if err := q.db.QueryRow(ctx, dailyStatsSummarySQL, startUnix, endUnix).Scan(
		&aggregate.TotalEvents,
		&aggregate.UniqueAuthors,
		&aggregate.UniqueKinds,
		&aggregate.EventsWithHashtags,
	); err != nil {
		return DailyStatsAggregate{}, fmt.Errorf("query daily stats summary: %w", err)
	}

	if err := q.collectDailyStatsKinds(ctx, startUnix, endUnix, &aggregate); err != nil {
		return DailyStatsAggregate{}, err
	}
	if err := q.collectDailyStatsAuthors(ctx, startUnix, endUnix, &aggregate); err != nil {
		return DailyStatsAggregate{}, err
	}
	if err := q.collectDailyStatsTags(ctx, startUnix, endUnix, &aggregate); err != nil {
		return DailyStatsAggregate{}, err
	}

	return aggregate, nil
}

func (q *Queries) collectDailyStatsKinds(ctx context.Context, startUnix, endUnix int64, aggregate *DailyStatsAggregate) error {
	rows, err := q.db.Query(ctx, dailyStatsTopKindsSQL, startUnix, endUnix)
	if err != nil {
		return fmt.Errorf("query daily stats top kinds: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item DailyStatsKind
		if err := rows.Scan(&item.Kind, &item.Count); err != nil {
			return fmt.Errorf("scan daily stats top kind: %w", err)
		}
		aggregate.TopKinds = append(aggregate.TopKinds, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate daily stats top kinds: %w", err)
	}
	return nil
}

func (q *Queries) collectDailyStatsAuthors(ctx context.Context, startUnix, endUnix int64, aggregate *DailyStatsAggregate) error {
	rows, err := q.db.Query(ctx, dailyStatsTopAuthorsSQL, startUnix, endUnix)
	if err != nil {
		return fmt.Errorf("query daily stats top authors: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item DailyStatsAuthor
		if err := rows.Scan(&item.Pubkey, &item.Count); err != nil {
			return fmt.Errorf("scan daily stats top author: %w", err)
		}
		aggregate.TopAuthors = append(aggregate.TopAuthors, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate daily stats top authors: %w", err)
	}
	return nil
}

func (q *Queries) collectDailyStatsTags(ctx context.Context, startUnix, endUnix int64, aggregate *DailyStatsAggregate) error {
	rows, err := q.db.Query(ctx, dailyStatsTopTagsSQL, startUnix, endUnix)
	if err != nil {
		return fmt.Errorf("query daily stats top tags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item DailyStatsTag
		if err := rows.Scan(&item.Tag, &item.Count); err != nil {
			return fmt.Errorf("scan daily stats top tag: %w", err)
		}
		aggregate.TopTags = append(aggregate.TopTags, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate daily stats top tags: %w", err)
	}
	return nil
}
