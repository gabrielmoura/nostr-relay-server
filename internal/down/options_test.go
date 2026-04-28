package down

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildOptions_FilterJSON(t *testing.T) {
	t.Parallel()

	t.Run("accepts valid object", func(t *testing.T) {
		t.Parallel()

		options, err := BuildOptions(CLIOptions{
			RelayURL: []string{"wss://relay.damus.io"},
			Kinds:    []int{1},
			Timeout:  30,
			Filter:   `{"search":"nostr","since":1700000000}`,
		})
		require.NoError(t, err)
		require.Equal(t, "nostr", options.Filter.Search)
		require.NotNil(t, options.Filter.Since)
	})

	t.Run("loads filter from file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		filterPath := filepath.Join(tmpDir, "filter.json")
		require.NoError(t, os.WriteFile(filterPath, []byte(`{"search":"from-file","since":1700000000}`), 0o600))

		options, err := BuildOptions(CLIOptions{
			RelayURL:   []string{"wss://relay.damus.io"},
			Kinds:      []int{1},
			Timeout:    30,
			FilterFile: filterPath,
		})
		require.NoError(t, err)
		require.Equal(t, "from-file", options.Filter.Search)
		require.NotNil(t, options.Filter.Since)
	})

	t.Run("rejects using filter and filter-file together", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		filterPath := filepath.Join(tmpDir, "filter.json")
		require.NoError(t, os.WriteFile(filterPath, []byte(`{"kinds":[1]}`), 0o600))

		_, err := BuildOptions(CLIOptions{
			RelayURL:   []string{"wss://relay.damus.io"},
			Kinds:      []int{1},
			Timeout:    30,
			Filter:     `{"kinds":[1]}`,
			FilterFile: filterPath,
		})
		require.ErrorContains(t, err, "use only one of --filter or --filter-file")
	})

	t.Run("rejects invalid json", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{
			RelayURL: []string{"wss://relay.damus.io"},
			Kinds:    []int{1},
			Timeout:  30,
			Filter:   `{"search":`,
		})
		require.ErrorContains(t, err, "invalid --filter JSON")
	})

	t.Run("rejects non-object filter", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{
			RelayURL: []string{"wss://relay.damus.io"},
			Kinds:    []int{1},
			Timeout:  30,
			Filter:   `[{"kinds":[1]}]`,
		})
		require.ErrorContains(t, err, "expected object")
	})

	t.Run("specific flags override filter fields", func(t *testing.T) {
		t.Parallel()

		options, err := BuildOptions(CLIOptions{
			RelayURL:  []string{"wss://relay.damus.io"},
			Kinds:     []int{1},
			Tags:      []string{"go", "nostr"},
			PublicKey: "abcd",
			Mentioned: true,
			Timeout:   30,
			Filter:    `{"authors":["ffff"],"kinds":[30023],"#t":["old"]}`,
		})
		require.NoError(t, err)
		require.Nil(t, options.Filter.Authors)
		require.Equal(t, []int{1}, options.Filter.Kinds)
		require.Equal(t, []string{"go", "nostr"}, options.Filter.Tags["t"])
		require.Equal(t, []string{"abcd"}, options.Filter.Tags["p"])
	})

	t.Run("strict-conflict rejects conflicting kinds", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{
			RelayURL: []string{"wss://relay.damus.io"},
			Kinds:    []int{1},
			Timeout:  30,
			Merge:    "strict-conflict",
			Filter:   `{"kinds":[30023]}`,
		})
		require.ErrorContains(t, err, "filter conflict on kinds")
	})

	t.Run("strict-conflict rejects conflicting author", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{
			RelayURL:  []string{"wss://relay.damus.io"},
			PublicKey: "abcd",
			Kinds:     []int{1},
			Timeout:   30,
			Merge:     "strict-conflict",
			Filter:    `{"authors":["ffff"]}`,
		})
		require.ErrorContains(t, err, "filter conflict on authors")
	})

	t.Run("strict-conflict accepts equal values", func(t *testing.T) {
		t.Parallel()

		options, err := BuildOptions(CLIOptions{
			RelayURL:  []string{"wss://relay.damus.io"},
			PublicKey: "abcd",
			Kinds:     []int{1, 30023},
			Tags:      []string{"go", "nostr"},
			Timeout:   30,
			Merge:     "strict-conflict",
			Filter:    `{"authors":["abcd"],"kinds":[30023,1],"#t":["nostr","go"]}`,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"abcd"}, options.Filter.Authors)
	})

	t.Run("rejects invalid merge strategy", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{
			RelayURL: []string{"wss://relay.damus.io"},
			Kinds:    []int{1},
			Timeout:  30,
			Merge:    "unknown",
		})
		require.ErrorContains(t, err, "invalid --filter-merge")
	})

	t.Run("mentioned requires public key", func(t *testing.T) {
		t.Parallel()

		_, err := BuildOptions(CLIOptions{
			RelayURL:  []string{"wss://relay.damus.io"},
			Kinds:     []int{1},
			Mentioned: true,
			Timeout:   30,
		})
		require.ErrorContains(t, err, "--mentioned requires --public-key")
	})
}
