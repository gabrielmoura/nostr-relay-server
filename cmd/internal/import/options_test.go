package _import

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildOptions(t *testing.T) {
	t.Parallel()

	t.Run("accepts valid options", func(t *testing.T) {
		t.Parallel()

		options, err := BuildOptions(CLIOptions{
			Filename:      "events.jsonl",
			BatchSize:     100,
			NumWorkers:    2,
			StatsInterval: 5 * time.Second,
		})
		require.NoError(t, err)
		require.Equal(t, "events.jsonl", options.Filename)
	})

	t.Run("rejects invalid worker and batch settings", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{Filename: "events.jsonl", BatchSize: -1, NumWorkers: 1})
		require.ErrorContains(t, err, "--batch-size")

		_, err = BuildOptions(CLIOptions{Filename: "events.jsonl", BatchSize: 10, NumWorkers: 0})
		require.ErrorContains(t, err, "--num-workers")
	})

	t.Run("rejects negative stats interval", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{Filename: "events.jsonl", BatchSize: 10, NumWorkers: 1, StatsInterval: -1 * time.Second})
		require.ErrorContains(t, err, "--stats-interval")
	})
}
