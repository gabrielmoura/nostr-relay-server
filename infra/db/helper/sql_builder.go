package helper

import (
	"fmt"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/jmoiron/sqlx"
	"github.com/nbd-wtf/go-nostr"
)

func BuildQuery(filter nostr.Filter, cfg *config.RelayConfig, doCount bool) (string, []any, error) {
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
		conditions = append(conditions, "true")
	}

	var builder strings.Builder
	if doCount {
		builder.WriteString("SELECT COUNT(*) FROM event WHERE ")
	} else {
		builder.WriteString("SELECT id, pubkey, created_at, kind, tags, content, sig FROM event WHERE ")
	}
	builder.WriteString(strings.Join(conditions, " AND "))

	if !doCount {
		builder.WriteString(" ORDER BY created_at DESC, id")
	}

	builder.WriteString(" LIMIT ?")
	params = append(params, filter.Limit)

	return sqlx.Rebind(sqlx.BindType("postgres"), builder.String()), params, nil
}

func addIDsCondition(conditions *[]string, params *[]any, ids []string) {
	if len(ids) == 0 {
		return
	}
	*conditions = append(*conditions, fmt.Sprintf("id IN (%s)", makePlaceholders(len(ids))))
	for _, id := range ids {
		*params = append(*params, id)
	}
}

func addAuthorsCondition(conditions *[]string, params *[]any, authors []string) {
	if len(authors) == 0 {
		return
	}
	*conditions = append(*conditions, fmt.Sprintf("pubkey IN (%s)", makePlaceholders(len(authors))))
	for _, author := range authors {
		*params = append(*params, author)
	}
}

func addKindsCondition(conditions *[]string, params *[]any, kinds []int) {
	if len(kinds) == 0 {
		return
	}
	*conditions = append(*conditions, fmt.Sprintf("kind IN (%s)", makePlaceholders(len(kinds))))
	for _, kind := range kinds {
		*params = append(*params, kind)
	}
}

func addTagsCondition(conditions *[]string, params *[]any, tags nostr.TagMap) {
	for _, values := range tags {
		*conditions = append(*conditions, fmt.Sprintf("tagvalues && ARRAY[%s]", makePlaceholders(len(values))))
		for _, value := range values {
			*params = append(*params, value)
		}
	}
}

func addTimeConditions(conditions *[]string, params *[]any, since, until *nostr.Timestamp) {
	if since != nil {
		*conditions = append(*conditions, "created_at >= ?")
		*params = append(*params, since)
	}
	if until != nil {
		*conditions = append(*conditions, "created_at <= ?")
		*params = append(*params, until)
	}
}

func addSearchCondition(conditions *[]string, params *[]any, search string) {
	if search == "" {
		return
	}
	terms := strings.Fields(search)
	tsQuery := strings.Join(terms, " & ")
	*conditions = append(*conditions, "content_search @@ to_tsquery('portuguese', ?)")
	*params = append(*params, tsQuery)
}

func addDeletionCondition(conditions *[]string, fakeDeletion bool) {
	if fakeDeletion {
		*conditions = append(*conditions, "deleted_by IS NULL")
	}
}

func makePlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}
