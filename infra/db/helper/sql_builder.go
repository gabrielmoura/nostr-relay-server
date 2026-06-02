package helper

import (
	"fmt"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
)

func BuildQuery(filter nostr.Filter, cfg *config.RelayConfig, doCount bool) (string, []any, error) {
	whereClause, params := BuildWhereClause(filter, cfg)

	var builder strings.Builder
	if doCount {
		builder.WriteString("SELECT COUNT(*) FROM event WHERE ")
	} else {
		builder.WriteString("SELECT id, pubkey, created_at, kind, tags, content, sig FROM event WHERE ")
	}
	builder.WriteString(whereClause)

	if !doCount {
		builder.WriteString(" ORDER BY created_at DESC, id")
	}

	builder.WriteString(" LIMIT ")
	builder.WriteString(addParam(&params, filter.Limit))

	return builder.String(), params, nil
}

func BuildWhereClause(filter nostr.Filter, cfg *config.RelayConfig) (string, []any) {
	conditions := make([]string, 0, 8)
	params := make([]any, 0, 8)

	addIDsCondition(&conditions, &params, filter.IDs)
	addAuthorsCondition(&conditions, &params, filter.Authors)
	addKindsCondition(&conditions, &params, filter.Kinds)
	addTagsCondition(&conditions, &params, filter.Tags)
	addTimeConditions(&conditions, &params, filter.Since, filter.Until)
	addSearchCondition(&conditions, &params, filter.Search)
	addDeletionCondition(&conditions, cfg.FakeDeletion)

	if len(conditions) == 0 {
		return "true", params
	}

	return strings.Join(conditions, " AND "), params
}

func addIDsCondition(conditions *[]string, params *[]any, ids []string) {
	if len(ids) == 0 {
		return
	}
	placeholders := make([]string, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, addParam(params, id))
	}
	*conditions = append(*conditions, fmt.Sprintf("id IN (%s)", strings.Join(placeholders, ",")))
}

func addAuthorsCondition(conditions *[]string, params *[]any, authors []string) {
	if len(authors) == 0 {
		return
	}
	placeholders := make([]string, 0, len(authors))
	for _, author := range authors {
		placeholders = append(placeholders, addParam(params, author))
	}
	*conditions = append(*conditions, fmt.Sprintf("pubkey IN (%s)", strings.Join(placeholders, ",")))
}

func addKindsCondition(conditions *[]string, params *[]any, kinds []int) {
	if len(kinds) == 0 {
		return
	}
	placeholders := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		placeholders = append(placeholders, addParam(params, kind))
	}
	*conditions = append(*conditions, fmt.Sprintf("kind IN (%s)", strings.Join(placeholders, ",")))
}

func addTagsCondition(conditions *[]string, params *[]any, tags nostr.TagMap) {
	for tagName, values := range tags {
		tagName = strings.TrimPrefix(tagName, "#")
		clauses := make([]string, 0, len(values))
		for _, value := range values {
			payload := fmt.Sprintf(`[[%q,%q]]`, tagName, value)
			clauses = append(clauses, "tags @> "+addParam(params, payload)+"::jsonb")
		}
		if len(clauses) == 1 {
			*conditions = append(*conditions, clauses[0])
			continue
		}
		*conditions = append(*conditions, "("+strings.Join(clauses, " OR ")+")")
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

func addSearchCondition(conditions *[]string, params *[]any, search string) {
	if search == "" {
		return
	}
	terms := strings.Fields(search)
	tsQuery := strings.Join(terms, " & ")
	tsPlaceholder := addParam(params, tsQuery)
	likePlaceholder := addParam(params, "%"+search+"%")
	*conditions = append(*conditions, `(
		content_search @@ to_tsquery('portuguese', `+tsPlaceholder+`)
		OR EXISTS (
			SELECT 1 FROM jsonb_array_elements(tags) tag
			WHERE lower(tag->>0) = 'description' AND tag->>1 ILIKE `+likePlaceholder+`
		)
	)`)
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
