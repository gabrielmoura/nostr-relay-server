package migratestrfry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      CLIOptions
		contains string
	}{
		{
			name: "normalizes valid export options",
			raw: CLIOptions{
				StrfryDB:  " /var/lib/strfry/db ",
				StrfryBin: " strfry ",
				Source:    " EXPORT ",
				BatchSize: 100,
				Since:     42,
			},
		},
		{
			name:     "requires source database",
			raw:      CLIOptions{StrfryBin: "strfry", BatchSize: 1},
			contains: "--strfry-db",
		},
		{
			name:     "requires export binary after defaulting source",
			raw:      CLIOptions{StrfryDB: "db", BatchSize: 1},
			contains: "--strfry-bin",
		},
		{
			name:     "rejects unknown source",
			raw:      CLIOptions{StrfryDB: "db", StrfryBin: "strfry", Source: "file", BatchSize: 1},
			contains: "--source",
		},
		{
			name:     "rejects negative since",
			raw:      CLIOptions{StrfryDB: "db", StrfryBin: "strfry", BatchSize: 1, Since: -1},
			contains: "--since",
		},
		{
			name:     "rejects nonpositive batch size",
			raw:      CLIOptions{StrfryDB: "db", StrfryBin: "strfry"},
			contains: "--batch-size",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options, err := BuildOptions(test.raw)
			if test.contains != "" {
				require.ErrorContains(t, err, test.contains)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "/var/lib/strfry/db", options.StrfryDB)
			require.Equal(t, "strfry", options.StrfryBin)
			require.Equal(t, SourceExport, options.Source)
		})
	}
}

func TestBuildOptionsAcceptsLMDBSource(t *testing.T) {
	t.Parallel()

	options, err := BuildOptions(CLIOptions{
		StrfryDB:  "/source/db",
		Source:    SourceLMDB,
		BatchSize: 100,
	})

	require.NoError(t, err)
	require.Equal(t, SourceLMDB, options.Source)
}
