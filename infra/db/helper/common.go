package helper

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
)

func QueryEventsSql(cfg *config.RelayConfig, filter nostr.Filter, doCount bool) (string, []any, error) {
	normalized := NormalizeFilter(cfg, filter)
	if err := ValidateFilterLimits(cfg, normalized); err != nil {
		return "", nil, err
	}
	return BuildQuery(normalized, cfg, doCount)
}
