package migratestrfry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunRejectsDirectLMDBBeforeLoadingTargetConfiguration(t *testing.T) {
	t.Parallel()

	_, err := Run(CLIOptions{
		StrfryDB:  "/source/strfry-db",
		Source:    SourceLMDB,
		BatchSize: 1,
	})

	require.ErrorContains(t, err, "direct LMDB reader is not available")
}
