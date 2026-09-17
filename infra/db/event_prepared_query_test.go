package db

import (
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestPreparedQueryForFilter_OnlySelectsEquivalentStatements(t *testing.T) {
	cfg := &config.RelayConfig{}
	since := nostr.Timestamp(100)

	tests := []struct {
		name       string
		filter     nostr.Filter
		fakeDelete bool
		statement  string
		params     []any
		ok         bool
	}{
		{
			name:      "id only",
			filter:    nostr.Filter{IDs: []string{"id"}},
			statement: "ps_event_by_id",
			params:    []any{"id"},
			ok:        true,
		},
		{
			name:      "author and kind",
			filter:    nostr.Filter{Authors: []string{"author"}, Kinds: []int{1}, Limit: 4},
			statement: "ps_events_by_pubkey_kind",
			params:    []any{"author", 1, 4},
			ok:        true,
		},
		{
			name:      "kind and since",
			filter:    nostr.Filter{Kinds: []int{1}, Since: &since, Limit: 4},
			statement: "ps_events_by_kind",
			params:    []any{1, since, 4},
			ok:        true,
		},
		{
			name:   "id with author is dynamic",
			filter: nostr.Filter{IDs: []string{"id"}, Authors: []string{"author"}},
		},
		{
			name:   "author with time range is dynamic",
			filter: nostr.Filter{Authors: []string{"author"}, Since: &since},
		},
		{
			name:       "fake deletion is dynamic",
			filter:     nostr.Filter{Authors: []string{"author"}},
			fakeDelete: true,
		},
		{
			name:   "search is dynamic",
			filter: nostr.Filter{Authors: []string{"author"}, Search: "nostr"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.FakeDeletion = tt.fakeDelete
			statement, params, ok := preparedQueryForFilter(cfg, tt.filter)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.statement, statement)
			require.Equal(t, tt.params, params)
		})
	}
}

func TestPreparedCountForFilter_OnlySelectsExactAuthorFilter(t *testing.T) {
	cfg := &config.RelayConfig{}

	statement, params, ok := preparedCountForFilter(cfg, nostr.Filter{Authors: []string{"author"}})
	require.True(t, ok)
	require.Equal(t, "ps_count_by_filter", statement)
	require.Equal(t, []any{"author"}, params)

	since := nostr.Timestamp(100)
	_, _, ok = preparedCountForFilter(cfg, nostr.Filter{Authors: []string{"author"}, Since: &since})
	require.False(t, ok)
}
