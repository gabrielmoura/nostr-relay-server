package db

import (
	"context"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
)

// PurgeExpiredNIP40Events removes all currently expired NIP-40 events in
// configured chunks. It is the cron operation, not a live-ingestion path:
// cache invalidation happens once after the operation, never per chunk.
func (q *Queries) PurgeExpiredNIP40Events(ctx context.Context, nowUnix int64, batchSize int) (int64, error) {
	if batchSize <= 0 {
		return 0, fmt.Errorf("NIP-40 expiration batch size must be positive")
	}

	var totalDeleted int64
	for {
		deleted, err := q.deleteExpiredNIP40EventsChunk(ctx, nowUnix, batchSize)
		if err != nil {
			return totalDeleted, fmt.Errorf("purge expired NIP-40 events: %w", err)
		}

		totalDeleted += deleted
		if deleted < int64(batchSize) {
			break
		}
	}

	if totalDeleted > 0 {
		_ = cache.InvalidateQueryCache()
	}

	return totalDeleted, nil
}
