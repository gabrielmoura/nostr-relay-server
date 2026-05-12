package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/db/helper"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

func (q *Queries) QueryEventsChan(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error) {
	events, err := q.QueryEvents(ctx, filter)
	if err != nil {
		return nil, err
	}

	ch := make(chan *nostr.Event)
	go func() {
		defer close(ch)
		for _, evt := range events {
			select {
			case ch <- evt:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

func (q *Queries) QueryEvents(ctx context.Context, filter nostr.Filter) ([]*nostr.Event, error) {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	cacheKey := helper.FilterHash(&config.Cfg.Relay, filter, false)
	if raw, ok := cache.GetQueryResult(cacheKey); ok {
		cache.QueryCacheHit(cacheKey)
		metrics.NostrRedisQueryCacheResult.WithLabelValues("hit").Inc()
		var events []*nostr.Event
		if err := json.Unmarshal([]byte(raw), &events); err == nil {
			return events, nil
		}
	}

	cache.QueryCacheMiss(cacheKey)
	metrics.NostrRedisQueryCacheResult.WithLabelValues("miss").Inc()
	query, params, err := queryEventsStatement(filter)
	if err != nil {
		return nil, err
	}

	rows, err := q.db.Query(ctx, query, params...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to fetch events using query %q: %w", query, err)
	}
	if err != nil {
		return nil, nil
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if payload, err := json.Marshal(events); err == nil {
		_ = cache.SetQueryResult(cacheKey, string(payload))
	}
	return events, nil
}

func (q *Queries) CountEvents(ctx context.Context, filter nostr.Filter) (int64, error) {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	cacheKey := helper.FilterHash(&config.Cfg.Relay, filter, true)
	if raw, ok := cache.GetQueryResult(cacheKey); ok {
		cache.QueryCacheHit(cacheKey)
		metrics.NostrRedisQueryCacheResult.WithLabelValues("hit").Inc()
		count, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			return count, nil
		}
	}

	cache.QueryCacheMiss(cacheKey)
	metrics.NostrRedisQueryCacheResult.WithLabelValues("miss").Inc()
	query, params, err := countEventsStatement(filter)
	if err != nil {
		return 0, err
	}

	var count int64
	if err := q.db.QueryRow(ctx, query, params...).Scan(&count); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("failed to fetch events using query %q: %w", query, err)
	}
	_ = cache.SetQueryResult(cacheKey, strconv.FormatInt(count, 10))
	return count, nil
}

func queryEventsStatement(filter nostr.Filter) (string, []any, error) {
	query, params, err := helper.QueryEventsSql(&config.Cfg.Relay, filter, false)
	if err != nil {
		return "", nil, err
	}
	if stmt, stmtParams, ok := preparedQueryForFilter(filter); ok {
		return stmt, stmtParams, nil
	}
	return query, params, nil
}

func countEventsStatement(filter nostr.Filter) (string, []any, error) {
	query, params, err := helper.QueryEventsSql(&config.Cfg.Relay, filter, true)
	if err != nil {
		return "", nil, err
	}
	if stmt, stmtParams, ok := preparedCountForFilter(filter); ok {
		return stmt, stmtParams, nil
	}
	return query, params, nil
}
