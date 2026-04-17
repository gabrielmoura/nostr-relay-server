package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/db/helper"
	"github.com/nbd-wtf/go-nostr"
)

func (q *Queries) QueryEventsPage(ctx context.Context, filter nostr.Filter, offset int) ([]*nostr.Event, error) {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	query, params, err := helper.QueryEventsSql(&config.Cfg.Relay, filter, false)
	if err != nil {
		return nil, err
	}

	query = query + fmt.Sprintf(" OFFSET $%d", len(params)+1)
	params = append(params, offset)

	rows, err := q.db.Query(ctx, query, params...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to fetch export events using query %q: %w", query, err)
	}
	if err != nil {
		return []*nostr.Event{}, nil
	}
	defer rows.Close()

	events := make([]*nostr.Event, 0, filter.Limit)
	for rows.Next() {
		var evt nostr.Event
		var timestamp int64
		if err := rows.Scan(&evt.ID, &evt.PubKey, &timestamp, &evt.Kind, &evt.Tags, &evt.Content, &evt.Sig); err != nil {
			return nil, err
		}
		evt.CreatedAt = nostr.Timestamp(timestamp)
		events = append(events, &evt)
	}

	return events, nil
}
