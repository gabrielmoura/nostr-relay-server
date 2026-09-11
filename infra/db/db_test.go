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
	queries := New(batchDBTX{rows: &batchRows{ids: []string{"event-1"}}})

	result, err := queries.InsertEventBatch(context.Background(), []*nostr.Event{
		{ID: "event-1", Tags: nostr.Tags{}},
		{ID: "event-2", Tags: nostr.Tags{}},
	})

	require.NoError(t, err)
	require.Equal(t, BatchInsertResult{
		Inserted:    1,
		Duplicates:  1,
		InsertedIDs: []string{"event-1"},
	}, result)
}

type batchDBTX struct {
	rows pgx.Rows
}

func (db batchDBTX) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (db batchDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return db.rows, nil
}

func (db batchDBTX) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func (db batchDBTX) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}

type batchRows struct {
	ids  []string
	next int
}

func (rows *batchRows) Close() {}

func (*batchRows) Err() error { return nil }

func (*batchRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (*batchRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (rows *batchRows) Next() bool {
	rows.next++
	return rows.next <= len(rows.ids)
}

func (rows *batchRows) Scan(dest ...any) error {
	*dest[0].(*string) = rows.ids[rows.next-1]
	return nil
}

func (rows *batchRows) Values() ([]any, error) { return []any{rows.ids[rows.next-1]}, nil }

func (*batchRows) RawValues() [][]byte { return nil }

func (*batchRows) Conn() *pgx.Conn { return nil }
