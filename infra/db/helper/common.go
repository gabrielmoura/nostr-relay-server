package helper

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
)

func QueryEventsSql(cfg *config.RelayConfig, filter nostr.Filter, doCount bool) (string, []any, error) {
	normalized, err := NormalizeAndValidateFilter(cfg, filter)
	if err != nil {
		return "", nil, err
	}
	return BuildQuery(normalized, cfg, doCount)
}

func NormalizeAndValidateFilter(cfg *config.RelayConfig, filter nostr.Filter) (nostr.Filter, error) {
	normalized := NormalizeFilter(cfg, filter)
	if err := ValidateFilterLimits(cfg, normalized); err != nil {
		return nostr.Filter{}, err
	}
	return normalized, nil
}
