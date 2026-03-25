package helper

import (
	"sort"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
)

func NormalizeFilter(cfg *config.RelayConfig, filter nostr.Filter) nostr.Filter {
	if filter.Limit == 0 || filter.Limit > cfg.QueryLimit {
		filter.Limit = cfg.QueryLimit
	}

	sort.Strings(filter.IDs)
	sort.Strings(filter.Authors)
	sort.Ints(filter.Kinds)
	filter.Tags = normalizeTags(filter.Tags)

	return filter
}

func normalizeTags(tags nostr.TagMap) nostr.TagMap {
	if len(tags) == 0 {
		return tags
	}

	normalized := make(nostr.TagMap, len(tags))
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		values := append([]string(nil), tags[key]...)
		sort.Strings(values)
		normalized[key] = values
	}

	return normalized
}
