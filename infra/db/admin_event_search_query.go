package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/db/helper"
	"github.com/nbd-wtf/go-nostr"
)

const countAllEvents = `-- name: CountAllEvents :one
SELECT COUNT(*) FROM event;
`

const countEventsSince = `-- name: CountEventsSince :one
SELECT COUNT(*)
FROM event
WHERE created_at >= $1::bigint;
`

func (q *Queries) CountAllEvents(ctx context.Context) (int64, error) {
	var count int64
	if err := q.db.QueryRow(ctx, countAllEvents).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) CountEventsSince(ctx context.Context, since int64) (int64, error) {
	var count int64
	if err := q.db.QueryRow(ctx, countEventsSince, since).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) QueryEventsWindow(ctx context.Context, filter nostr.Filter, offset int) ([]*nostr.Event, int64, error) {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	total, err := q.CountEvents(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	events, err := q.QueryEventsWindowOnly(ctx, filter, offset)
	return events, total, err
}

// QueryEventsWindowOnly reads a window without performing a separate total count.
func (q *Queries) QueryEventsWindowOnly(ctx context.Context, filter nostr.Filter, offset int) ([]*nostr.Event, error) {
	return q.queryEventsWindow(ctx, filter, offset, nil)
}

// QueryEventsWindowSorted reads a chronologically ordered window for the public API.
func (q *Queries) QueryEventsWindowSorted(ctx context.Context, filter nostr.Filter, offset int, ascending bool) ([]*nostr.Event, error) {
	return q.queryEventsWindow(ctx, filter, offset, &ascending)
}

func (q *Queries) queryEventsWindow(ctx context.Context, filter nostr.Filter, offset int, ascending *bool) ([]*nostr.Event, error) {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	var query string
	var params []any
	var err error
	if ascending == nil {
		query, params, err = helper.QueryEventsSql(&config.Cfg.Relay, filter, false)
	} else {
		filter, err = helper.NormalizeAndValidateFilter(&config.Cfg.Relay, filter)
		if err == nil {
			query, params, err = helper.BuildChronologicalQuery(filter, &config.Cfg.Relay, *ascending)
		}
	}
	if err != nil {
		return nil, err
	}
	query += fmt.Sprintf(" OFFSET $%d", len(params)+1)
	params = append(params, offset)

	rows, err := q.db.Query(ctx, query, params...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to fetch event page using query %q: %w", query, err)
	}
	if err != nil {
		return []*nostr.Event{}, nil
	}
	defer rows.Close()

	events := make([]*nostr.Event, 0, filter.Limit)
	for rows.Next() {
		evt, scanErr := scanNostrEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, evt)
	}

	return events, rows.Err()
}

func (q *Queries) GetEventAggregates(ctx context.Context, filter nostr.Filter) (EventAggregates, error) {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	whereClause, params := helper.BuildWhereClause(filter, &config.Cfg.Relay)
	total, err := q.CountEvents(ctx, filter)
	if err != nil {
		return EventAggregates{}, err
	}

	aggregates := EventAggregates{Total: total}
	if err := q.loadEventKindAggregates(ctx, whereClause, params, &aggregates); err != nil {
		return EventAggregates{}, err
	}
	if err := q.loadEventAuthorAggregates(ctx, whereClause, params, &aggregates); err != nil {
		return EventAggregates{}, err
	}
	if err := q.loadEventTagAggregates(ctx, whereClause, params, &aggregates); err != nil {
		return EventAggregates{}, err
	}
	if err := q.loadEventTrends(ctx, whereClause, params, &aggregates); err != nil {
		return EventAggregates{}, err
	}

	return aggregates, nil
}

func (q *Queries) GetEventTimeline(ctx context.Context, filter nostr.Filter, bucket string) ([]EventTimelinePoint, error) {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	whereClause, params := helper.BuildWhereClause(filter, &config.Cfg.Relay)
	step := int64(3600)
	if strings.EqualFold(strings.TrimSpace(bucket), "day") {
		step = 86400
	}

	timelineSQL := fmt.Sprintf(`
SELECT (created_at / %s) * %s AS ts, COUNT(*) AS count
FROM event
WHERE %s
GROUP BY ts
ORDER BY ts ASC;`, strconv.FormatInt(step, 10), strconv.FormatInt(step, 10), whereClause)

	rows, err := q.db.Query(ctx, timelineSQL, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]EventTimelinePoint, 0, 32)
	for rows.Next() {
		var item EventTimelinePoint
		if err := rows.Scan(&item.TS, &item.Count); err != nil {
			return nil, err
		}
		points = append(points, item)
	}

	return points, rows.Err()
}
