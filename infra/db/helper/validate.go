package helper

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
)

func ValidateFilterLimits(cfg *config.RelayConfig, filter nostr.Filter) error {
	if len(filter.IDs) > cfg.QueryIDsLimit {
		return ErrTooManyIDs
	}
	if len(filter.Authors) > cfg.QueryAuthorsLimit {
		return ErrTooManyAuthors
	}
	if len(filter.Kinds) > cfg.QueryKindsLimit {
		return ErrTooManyKinds
	}
	return validateTags(cfg.QueryTagsLimit, filter.Tags)
}

func validateTags(limit int, tags nostr.TagMap) error {
	totalValues := 0
	for _, values := range tags {
		if len(values) == 0 {
			return ErrEmptyTagSet
		}
		totalValues += len(values)
		if totalValues > limit {
			return ErrTooManyTagValues
		}
	}
	return nil
}
