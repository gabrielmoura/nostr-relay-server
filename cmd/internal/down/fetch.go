package down

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

const pageSize = 500

type eventStore interface {
	InsertEvent(context.Context, *nostr.Event) error
}

type FetchStats struct {
	Received   int
	Persisted  int
	Duplicates int
	Pages      int
}

func fetchAndStoreEvents(
	ctx context.Context,
	client *nostr.Relay,
	baseFilter nostr.Filter,
	timeout time.Duration,
	store eventStore,
) (FetchStats, error) {

	currentUntil := initialUntil(baseFilter.Until)
	stats := FetchStats{}

	for {
		pageStart := time.Now()
		pageFilter := baseFilter.Clone()
		until := nostr.Timestamp(currentUntil)
		pageFilter.Until = &until
		pageFilter.Limit = pageSize
		pageFilter.LimitZero = false

		pageCtx, cancel := context.WithTimeout(ctx, timeout)
		sub, err := client.Subscribe(pageCtx, []nostr.Filter{pageFilter})
		if err != nil {
			cancel()
			return stats, fmt.Errorf("subscribe relay %q: %w", client.URL, err)
		}

		pageCount := 0
		lastTimestamp := currentUntil

		for evt := range sub.Events {
			stats.Received++
			eventTimestamp := evt.CreatedAt.Time().Unix()
			if eventTimestamp < lastTimestamp {
				lastTimestamp = eventTimestamp
			}

			pageCount++
			if err := store.InsertEvent(ctx, evt); err != nil {
				if errors.Is(err, dbmodel.ErrDupEvent) {
					stats.Duplicates++
					continue
				}

				log.Logger.Error("Failed to persist event", zap.Error(err), zap.String("id", evt.ID))
				continue
			}

			stats.Persisted++
		}

		sub.Unsub()
		cancel()
		stats.Pages++
		metrics.NostrDownloadPageLatencySeconds.WithLabelValues(client.URL).Observe(time.Since(pageStart).Seconds())

		if pageCount < pageSize {
			break
		}

		if lastTimestamp <= 0 || lastTimestamp >= currentUntil {
			break
		}

		currentUntil = lastTimestamp - 1
	}

	return stats, nil
}

func initialUntil(input *nostr.Timestamp) int64 {
	if input == nil {
		return nostr.Now().Time().Unix()
	}

	return input.Time().Unix()
}
