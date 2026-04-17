package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildOptions(t *testing.T) {
	t.Parallel()

	t.Run("accepts jsonl and tsv", func(t *testing.T) {
		t.Parallel()

		jsonl, err := BuildOptions(CLIOptions{Filename: "events.jsonl", BatchSize: 100, WriterWorkers: 2, Format: "jsonl", Limit: 0, SegmentSize: 0})
		require.NoError(t, err)
		require.Equal(t, FormatJSONL, jsonl.Format)

		tsv, err := BuildOptions(CLIOptions{Filename: "events.tsv", BatchSize: 100, WriterWorkers: 2, Format: "tsv", SegmentSize: 500, Overwrite: true})
		require.NoError(t, err)
		require.Equal(t, FormatTSV, tsv.Format)
	})

	t.Run("rejects invalid format", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{Filename: "events.txt", BatchSize: 100, WriterWorkers: 2, Format: "txt"})
		require.ErrorContains(t, err, "invalid --format")
	})

	t.Run("rejects invalid numeric options", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{Filename: "events.jsonl", BatchSize: 0, WriterWorkers: 2, Format: "jsonl"})
		require.ErrorContains(t, err, "--batch-size")

		_, err = BuildOptions(CLIOptions{Filename: "events.jsonl", BatchSize: 100, WriterWorkers: 0, Format: "jsonl"})
		require.ErrorContains(t, err, "--writer-workers")

		_, err = BuildOptions(CLIOptions{Filename: "events.jsonl", BatchSize: 100, WriterWorkers: 2, Format: "jsonl", Limit: -1})
		require.ErrorContains(t, err, "--limit")

		_, err = BuildOptions(CLIOptions{Filename: "events.jsonl", BatchSize: 100, WriterWorkers: 2, Format: "jsonl", SegmentSize: -1})
		require.ErrorContains(t, err, "--segment-size")

		validated, err := BuildOptions(CLIOptions{Filename: "events.jsonl", BatchSize: 100, WriterWorkers: 2, Format: "jsonl", Limit: 100, SegmentSize: 200})
		require.NoError(t, err)
		require.Equal(t, 200, validated.SegmentSize)
	})

	t.Run("accepts filter-file and rejects mixed filter sources", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()
		filterPath := filepath.Join(tmp, "filter.json")
		require.NoError(t, os.WriteFile(filterPath, []byte(`{"kinds":[1]}`), 0o600))

		opt, err := BuildOptions(CLIOptions{
			Filename:      "events.jsonl",
			BatchSize:     100,
			WriterWorkers: 2,
			Format:        "jsonl",
			FilterFile:    filterPath,
		})
		require.NoError(t, err)
		require.Equal(t, filterPath, opt.FilterFile)

		_, err = BuildOptions(CLIOptions{
			Filename:      "events.jsonl",
			BatchSize:     100,
			WriterWorkers: 2,
			Format:        "jsonl",
			Filter:        `{"authors":["x"]}`,
			FilterFile:    filterPath,
		})
		require.ErrorContains(t, err, "use only one of --filter or --filter-file")
	})
}
