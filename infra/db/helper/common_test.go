package helper

import (
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func testConfig() *config.RelayConfig {
	return &config.RelayConfig{
		QueryIDsLimit:     5,
		QueryAuthorsLimit: 10,
		QueryKindsLimit:   10,
		QueryTagsLimit:    10,
		QueryLimit:        5,
	}
}

func TestNormalizeFilter_SortsAndClamps(t *testing.T) {
	cfg := testConfig()
	filter := nostr.Filter{
		IDs:     []string{"id2", "id1"},
		Authors: []string{"b", "a"},
		Kinds:   []int{7, 1},
		Tags: nostr.TagMap{
			"#p": {"v2", "v1"},
			"#e": {"x2", "x1"},
		},
		Limit: 999,
	}

	normalized := NormalizeFilter(cfg, filter)
	require.Equal(t, []string{"id1", "id2"}, normalized.IDs)
	require.Equal(t, []string{"a", "b"}, normalized.Authors)
	require.Equal(t, []int{1, 7}, normalized.Kinds)
	require.Equal(t, []string{"x1", "x2"}, normalized.Tags["#e"])
	require.Equal(t, []string{"v1", "v2"}, normalized.Tags["#p"])
	require.Equal(t, cfg.QueryLimit, normalized.Limit)
}

func TestQueryEventsSQL_BuildsEventQuery(t *testing.T) {
	cfg := testConfig()
	filter := nostr.Filter{
		IDs:     []string{"id2", "id1"},
		Authors: []string{"author1"},
		Kinds:   []int{1},
		Tags: nostr.TagMap{
			"#p": {"val2", "val1"},
		},
		Search: "nostr relay",
		Limit:  3,
	}

	query, params, err := QueryEventsSql(cfg, filter, false)
	require.NoError(t, err)
	require.NotContains(t, query, "?")
	require.Contains(t, query, "SELECT id, pubkey, created_at, kind, tags, content, sig FROM event WHERE")
	require.Contains(t, query, "id = ANY($1::text[])")
	require.Contains(t, query, "pubkey = ANY($2::text[])")
	require.Contains(t, query, "kind = ANY($3::integer[])")
	require.Contains(t, query, "(tagvalues && $4::text[] AND (tags @> $5::jsonb OR tags @> $6::jsonb))")
	require.Contains(t, query, "content_search @@ plainto_tsquery('simple', $7)")
	require.Contains(t, query, "ORDER BY ts_rank_cd(content_search, plainto_tsquery('simple', $7)) DESC, created_at DESC, id LIMIT $8")
	require.NotContains(t, query, "ILIKE")
	require.Equal(t, []any{[]string{"id1", "id2"}, []string{"author1"}, []int{1}, []string{"val1", "val2"}, `[["p","val1"]]`, `[["p","val2"]]`, "nostr relay", 3}, params)
}

func TestQueryEventsSQL_BuildsCountQuery(t *testing.T) {
	cfg := testConfig()
	query, params, err := QueryEventsSql(cfg, nostr.Filter{Authors: []string{"author1", "author2"}}, true)
	require.NoError(t, err)
	require.Contains(t, query, "SELECT COUNT(*) FROM event WHERE")
	require.NotContains(t, query, "ORDER BY")
	require.Equal(t, []any{[]string{"author1", "author2"}, cfg.QueryLimit}, params)
}

func TestQueryEventsSQL_ValidatesLimits(t *testing.T) {
	cfg := testConfig()
	_, _, err := QueryEventsSql(cfg, nostr.Filter{IDs: []string{"1", "2", "3", "4", "5", "6"}}, false)
	require.ErrorIs(t, err, ErrTooManyIDs)

	_, _, err = QueryEventsSql(cfg, nostr.Filter{Tags: nostr.TagMap{"#e": {}}}, false)
	require.ErrorIs(t, err, ErrEmptyTagSet)

	_, _, err = QueryEventsSql(cfg, nostr.Filter{Tags: nostr.TagMap{"#e": {"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}}}, false)
	require.ErrorIs(t, err, ErrTooManyTagValues)
}

func TestQueryEventsSQL_FakeDeletion(t *testing.T) {
	cfg := testConfig()
	cfg.FakeDeletion = true
	query, _, err := QueryEventsSql(cfg, nostr.Filter{Authors: []string{"author1"}}, false)
	require.NoError(t, err)
	require.Contains(t, query, "deleted_by IS NULL")
}

func TestQueryEventsSQL_SearchWithoutTermsReturnsNoRows(t *testing.T) {
	cfg := testConfig()

	for name, search := range map[string]string{
		"spaces":     "   ",
		"extensions": "domain:example.com include:spam",
		"emoji":      "👩🏽‍💻",
	} {
		t.Run(name, func(t *testing.T) {
			query, params, err := QueryEventsSql(cfg, nostr.Filter{Search: search}, false)
			require.NoError(t, err)
			require.Contains(t, query, "WHERE FALSE")
			require.Empty(t, params[:len(params)-1])
		})
	}
}

func TestQueryEventsSQL_RejectsTooLongSearch(t *testing.T) {
	cfg := testConfig()
	_, _, err := QueryEventsSql(cfg, nostr.Filter{Search: string(make([]byte, maxSearchLength+1))}, false)
	require.ErrorIs(t, err, ErrSearchTooLong)
}

func TestTagPayload_EscapesJSONStrings(t *testing.T) {
	require.Equal(t, `[["p","value\"with\\escape"]]`, tagPayload("p", `value"with\escape`))
}

func TestFilterHash_IsStable(t *testing.T) {
	cfg := testConfig()
	first := nostr.Filter{
		IDs:     []string{"b", "a"},
		Authors: []string{"pub2", "pub1"},
		Kinds:   []int{7, 1},
		Tags: nostr.TagMap{
			"#p": {"2", "1"},
		},
	}
	second := nostr.Filter{
		IDs:     []string{"a", "b"},
		Authors: []string{"pub1", "pub2"},
		Kinds:   []int{1, 7},
		Tags: nostr.TagMap{
			"#p": {"1", "2"},
		},
	}
	require.Equal(t, FilterHash(cfg, first, false), FilterHash(cfg, second, false))
}
