package croncmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildOptions(t *testing.T) {
	t.Parallel()

	t.Run("normalizes jobs and accepts valid config", func(t *testing.T) {
		t.Parallel()

		options, err := BuildOptions(Options{
			Jobs:    []string{"db_optimization, nip40_expiration_cleanup", "nip40"},
			Timeout: 30 * time.Minute,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"db_optimization", "nip40"}, options.Jobs)
	})

	t.Run("rejects list with run-once", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(Options{List: true, RunOnce: true, Timeout: 30 * time.Minute})
		require.ErrorContains(t, err, "--list cannot be used with --run-once")
	})

	t.Run("rejects non-positive timeout", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(Options{Timeout: 0})
		require.ErrorContains(t, err, "must be greater than 0")
	})
}
