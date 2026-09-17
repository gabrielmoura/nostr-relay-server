package helper

import (
	"fmt"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
)

func BuildQuery(filter nostr.Filter, cfg *config.RelayConfig, doCount bool) (string, []any, error) {
	search := ParseSearchQuery(filter.Search)
	whereClause, params, searchPlaceholder := buildWhereClause(filter, cfg, search)

	var builder strings.Builder
	if doCount {
		builder.WriteString("SELECT COUNT(*) FROM event WHERE ")
	} else {
		builder.WriteString("SELECT id, pubkey, created_at, kind, tags, content, sig FROM event WHERE ")
	}
	builder.WriteString(whereClause)

	if !doCount {
		if searchPlaceholder != "" {
			builder.WriteString(" ORDER BY ts_rank_cd(content_search, plainto_tsquery('simple', ")
			builder.WriteString(searchPlaceholder)
			builder.WriteString(")) DESC, created_at DESC, id")
		} else {
			builder.WriteString(" ORDER BY created_at DESC, id")
		}
	}

	builder.WriteString(" LIMIT ")
	builder.WriteString(addParam(&params, filter.Limit))

	return builder.String(), params, nil
}

func BuildWhereClause(filter nostr.Filter, cfg *config.RelayConfig) (string, []any) {
	whereClause, params, _ := buildWhereClause(filter, cfg, ParseSearchQuery(filter.Search))
	return whereClause, params
}

func buildWhereClause(filter nostr.Filter, cfg *config.RelayConfig, search SearchQuery) (string, []any, string) {
	conditions := make([]string, 0, 8)
	params := make([]any, 0, 8)

	addIDsCondition(&conditions, &params, filter.IDs)
	addAuthorsCondition(&conditions, &params, filter.Authors)
	addKindsCondition(&conditions, &params, filter.Kinds)
	addTagsCondition(&conditions, &params, filter.Tags)
	addTimeConditions(&conditions, &params, filter.Since, filter.Until)
	searchPlaceholder := addSearchCondition(&conditions, &params, search)
	addDeletionCondition(&conditions, cfg.FakeDeletion)

	if len(conditions) == 0 {
		return "true", params, searchPlaceholder
	}

	return strings.Join(conditions, " AND "), params, searchPlaceholder
}

func addIDsCondition(conditions *[]string, params *[]any, ids []string) {
	if len(ids) == 0 {
		return
	}
	*conditions = append(*conditions, "id = ANY("+addParam(params, ids)+"::text[])")
}

func addAuthorsCondition(conditions *[]string, params *[]any, authors []string) {
	if len(authors) == 0 {
		return
	}
	*conditions = append(*conditions, "pubkey = ANY("+addParam(params, authors)+"::text[])")
}

func addKindsCondition(conditions *[]string, params *[]any, kinds []int) {
	if len(kinds) == 0 {
		return
	}
	*conditions = append(*conditions, "kind = ANY("+addParam(params, kinds)+"::integer[])")
}

func addTagsCondition(conditions *[]string, params *[]any, tags nostr.TagMap) {
	for tagName, values := range tags {
		tagName = strings.TrimPrefix(tagName, "#")
		clauses := make([]string, 0, len(values))
		valuesPlaceholder := addParam(params, values)
		for _, value := range values {
			payload := fmt.Sprintf(`[[%q,%q]]`, tagName, value)
			clauses = append(clauses, "tags @> "+addParam(params, payload)+"::jsonb")
		}
		exactCondition := clauses[0]
		if len(clauses) > 1 {
			exactCondition = "(" + strings.Join(clauses, " OR ") + ")"
		}
		*conditions = append(*conditions, "(tagvalues && "+valuesPlaceholder+"::text[] AND "+exactCondition+")")
	}
}

func addTimeConditions(conditions *[]string, params *[]any, since, until *nostr.Timestamp) {
	if since != nil {
		*conditions = append(*conditions, "created_at >= "+addParam(params, since))
	}
	if until != nil {
		*conditions = append(*conditions, "created_at <= "+addParam(params, until))
	}
}

func addSearchCondition(conditions *[]string, params *[]any, search SearchQuery) string {
	if len(search.Terms) == 0 {
		if search.HasInput {
			*conditions = append(*conditions, "FALSE")
		}
		return ""
	}

	placeholder := addParam(params, strings.Join(search.Terms, " "))
	*conditions = append(*conditions, "content_search @@ plainto_tsquery('simple', "+placeholder+")")
	return placeholder
}

func addDeletionCondition(conditions *[]string, fakeDeletion bool) {
	if fakeDeletion {
		*conditions = append(*conditions, "deleted_by IS NULL")
	}
}

func addParam(params *[]any, value any) string {
	*params = append(*params, value)
	return fmt.Sprintf("$%d", len(*params))
}
