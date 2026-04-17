package seed

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildOptions(t *testing.T) {
	t.Parallel()

	t.Run("valid defaults", func(t *testing.T) {
		t.Parallel()

		options, err := BuildOptions(CLIOptions{Timeout: 30 * time.Second})
		require.NoError(t, err)
		require.False(t, options.Bootstrap)
	})

	t.Run("rejects skip migrate without bootstrap", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{SkipMigrate: true, Timeout: 30 * time.Second})
		require.ErrorContains(t, err, "--skip-migrate requires --bootstrap")
	})

	t.Run("rejects bootstrap idempotent without bootstrap", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{BootstrapIdempotent: true, Timeout: 30 * time.Second})
		require.ErrorContains(t, err, "--bootstrap-idempotent requires --bootstrap")
	})

	t.Run("rejects non-positive timeout", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{Timeout: 0})
		require.ErrorContains(t, err, "must be greater than 0")
	})
}
