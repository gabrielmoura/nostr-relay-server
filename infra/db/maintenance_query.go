package db

import (
	"context"

	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
)

const analyzeTablesSQL = `ANALYZE event, profiles, banned_users, objects`
const vacuumAnalyzeTablesSQL = `VACUUM (ANALYZE) event, profiles, banned_users, objects`
const reindexEventTableSQL = `REINDEX TABLE CONCURRENTLY event`

func (q *Queries) AnalyzeTables(ctx context.Context) error {
	_, err := q.db.Exec(ctx, analyzeTablesSQL)
	if err == nil {
		_ = cache.InvalidateQueryCache()
	}
	return err
}

func (q *Queries) VacuumAnalyzeTables(ctx context.Context) error {
	_, err := q.db.Exec(ctx, vacuumAnalyzeTablesSQL)
	if err == nil {
		_ = cache.InvalidateQueryCache()
	}
	return err
}

func (q *Queries) ReindexEventTable(ctx context.Context) error {
	_, err := q.db.Exec(ctx, reindexEventTableSQL)
	if err == nil {
		_ = cache.InvalidateQueryCache()
	}
	return err
}
