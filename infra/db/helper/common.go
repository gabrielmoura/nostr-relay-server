package helper

import (
	"errors"
	"github.com/gabrielmoura/nostr-relay-server/config"

	"github.com/jmoiron/sqlx"
	"github.com/nbd-wtf/go-nostr"
	"strings"
)

var (
	ErrTooManyIDs       = errors.New("too many ids")
	ErrTooManyAuthors   = errors.New("too many authors")
	ErrTooManyKinds     = errors.New("too many kinds")
	ErrTooManyTagValues = errors.New("too many tag values")
	ErrEmptyTagSet      = errors.New("empty tag set")
)

func QueryEventsSql(cfg *config.RelayConfig, filter nostr.Filter, doCount bool) (string, []any, error) {
	conditions := make([]string, 0, 7)
	params := make([]any, 0, 20)

	if len(filter.IDs) > 0 {
		if len(filter.IDs) > cfg.QueryIDsLimit {
			return "", nil, ErrTooManyIDs
		}
		for _, v := range filter.IDs {
			params = append(params, v)
		}
		conditions = append(conditions, `id IN (`+makePlaceHolders(len(filter.IDs))+`)`)
	}

	if len(filter.Authors) > 0 {
		if len(filter.Authors) > cfg.QueryAuthorsLimit {
			return "", nil, ErrTooManyAuthors
		}
		for _, v := range filter.Authors {
			params = append(params, v)
		}
		conditions = append(conditions, `pubkey IN (`+makePlaceHolders(len(filter.Authors))+`)`)
	}

	if len(filter.Kinds) > 0 {
		if len(filter.Kinds) > cfg.QueryKindsLimit {
			return "", nil, ErrTooManyKinds
		}
		for _, v := range filter.Kinds {
			params = append(params, v)
		}
		conditions = append(conditions, `kind IN (`+makePlaceHolders(len(filter.Kinds))+`)`)
	}

	totalTags := 0
	for _, values := range filter.Tags {
		if len(values) == 0 {
			return "", nil, ErrEmptyTagSet
		}
		for _, tagValue := range values {
			params = append(params, tagValue)
		}
		conditions = append(conditions, `tagvalues && ARRAY[`+makePlaceHolders(len(values))+`]`)
		totalTags += len(values)
		if totalTags > cfg.QueryTagsLimit {
			return "", nil, ErrTooManyTagValues
		}
	}

	if filter.Since != nil {
		conditions = append(conditions, `created_at >= ?`)
		params = append(params, filter.Since)
	}
	if filter.Until != nil {
		conditions = append(conditions, `created_at <= ?`)
		params = append(params, filter.Until)
	}
	if filter.Search != "" {
		conditions = append(conditions, `content LIKE ?`)
		params = append(params, `%`+strings.ReplaceAll(filter.Search, `%`, `\%`)+`%`)
	}

	if len(conditions) == 0 {
		conditions = append(conditions, `true`)
	}

	if filter.Limit < 1 || filter.Limit > cfg.QueryLimit {
		params = append(params, cfg.QueryLimit)
	} else {
		params = append(params, filter.Limit)
	}

	if cfg.FakeDeletion {
		conditions = append(conditions, `deleted_by IS NULL`)
	}

	var query string
	if doCount {
		query = sqlx.Rebind(sqlx.BindType("postgres"), `SELECT COUNT(*) FROM event WHERE `+
			strings.Join(conditions, " AND ")+" LIMIT ?")
	} else {
		query = sqlx.Rebind(sqlx.BindType("postgres"), `SELECT id, pubkey, created_at, kind, tags, content, sig FROM event WHERE `+
			strings.Join(conditions, " AND ")+" ORDER BY created_at DESC, id LIMIT ?")
	}

	return query, params, nil
}
func makePlaceHolders(n int) string {
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}
