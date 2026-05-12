package db

import "github.com/nbd-wtf/go-nostr"

func preparedQueryForFilter(filter nostr.Filter) (string, []any, bool) {
	if filter.Search != "" || len(filter.Tags) > 0 || len(filter.IDs) > 1 || len(filter.Authors) > 1 || len(filter.Kinds) > 1 {
		return "", nil, false
	}
	if len(filter.IDs) == 1 {
		return "ps_event_by_id", []any{filter.IDs[0]}, true
	}
	if len(filter.Authors) == 1 && len(filter.Kinds) == 1 {
		return "ps_events_by_pubkey_kind", []any{filter.Authors[0], filter.Kinds[0], filter.Limit}, true
	}
	if len(filter.Authors) == 1 {
		return "ps_events_by_pubkey", []any{filter.Authors[0], filter.Limit}, true
	}
	if len(filter.Kinds) == 1 && filter.Since != nil {
		return "ps_events_by_kind", []any{filter.Kinds[0], *filter.Since, filter.Limit}, true
	}
	return "", nil, false
}

func preparedCountForFilter(filter nostr.Filter) (string, []any, bool) {
	if filter.Search != "" || len(filter.Tags) > 0 || len(filter.IDs) > 0 || len(filter.Kinds) > 0 || len(filter.Authors) != 1 {
		return "", nil, false
	}
	return "ps_count_by_filter", []any{filter.Authors[0]}, true
}
