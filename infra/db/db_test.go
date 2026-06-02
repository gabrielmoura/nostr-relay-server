package db

import (
	"testing"

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
