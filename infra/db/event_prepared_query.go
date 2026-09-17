package db

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
)

func preparedQueryForFilter(cfg *config.RelayConfig, filter nostr.Filter) (string, []any, bool) {
	if cfg.FakeDeletion || filter.Search != "" || len(filter.Tags) > 0 {
		return "", nil, false
	}
	if len(filter.IDs) == 1 && len(filter.Authors) == 0 && len(filter.Kinds) == 0 && filter.Since == nil && filter.Until == nil {
		return "ps_event_by_id", []any{filter.IDs[0]}, true
	}
	if len(filter.IDs) == 0 && len(filter.Authors) == 1 && len(filter.Kinds) == 1 && filter.Since == nil && filter.Until == nil {
		return "ps_events_by_pubkey_kind", []any{filter.Authors[0], filter.Kinds[0], filter.Limit}, true
	}
	if len(filter.IDs) == 0 && len(filter.Authors) == 1 && len(filter.Kinds) == 0 && filter.Since == nil && filter.Until == nil {
		return "ps_events_by_pubkey", []any{filter.Authors[0], filter.Limit}, true
	}
	if len(filter.IDs) == 0 && len(filter.Authors) == 0 && len(filter.Kinds) == 1 && filter.Since != nil && filter.Until == nil {
		return "ps_events_by_kind", []any{filter.Kinds[0], *filter.Since, filter.Limit}, true
	}
	return "", nil, false
}

func preparedCountForFilter(cfg *config.RelayConfig, filter nostr.Filter) (string, []any, bool) {
	if cfg.FakeDeletion || filter.Search != "" || len(filter.Tags) > 0 || len(filter.IDs) > 0 || len(filter.Kinds) > 0 || len(filter.Authors) != 1 || filter.Since != nil || filter.Until != nil {
		return "", nil, false
	}
	return "ps_count_by_filter", []any{filter.Authors[0]}, true
}
