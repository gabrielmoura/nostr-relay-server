package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestSplitSQLStatementsKeepsDollarQuotedBlocksIntact(t *testing.T) {
	input := `
CREATE OR REPLACE FUNCTION public.example()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM 1;
    RETURN NEW;
END;
$$;

DO $$
BEGIN
    PERFORM public.example();
END;
$$;

CREATE INDEX IF NOT EXISTS idx_example ON public.event (created_at);
`

	statements := splitSQLStatements(input)
	require.Len(t, statements, 3)
	require.Contains(t, statements[0], "PERFORM 1;")
	require.Contains(t, statements[1], "PERFORM public.example();")
	require.Equal(t, "CREATE INDEX IF NOT EXISTS idx_example ON public.event (created_at);", statements[2])
}

func TestInsertEventBatchCountsInsertedAndDuplicateRows(t *testing.T) {
	queries := New(batchDBTX{results: &batchResults{tags: []pgconn.CommandTag{
		pgconn.NewCommandTag("INSERT 0 1"),
		pgconn.NewCommandTag("INSERT 0 0"),
	}}})

	result, err := queries.InsertEventBatch(context.Background(), []*nostr.Event{
		{ID: "event-1"},
		{ID: "event-2"},
	})

	require.NoError(t, err)
	require.Equal(t, BatchInsertResult{Inserted: 1, Duplicates: 1}, result)
}

type batchDBTX struct {
	results pgx.BatchResults
}

func (db batchDBTX) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (db batchDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (db batchDBTX) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func (db batchDBTX) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return db.results
}

type batchResults struct {
	tags []pgconn.CommandTag
	next int
}

func (results *batchResults) Exec() (pgconn.CommandTag, error) {
	tag := results.tags[results.next]
	results.next++
	return tag, nil
}

func (*batchResults) Query() (pgx.Rows, error) {
	return nil, nil
}

func (*batchResults) QueryRow() pgx.Row {
	return nil
}

func (*batchResults) Close() error {
	return nil
}
