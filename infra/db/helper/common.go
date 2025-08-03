package helper

import (
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
)

// Erros pré-definidos para validação de limites de filtro.
var (
	ErrTooManyIDs       = errors.New("too many ids")
	ErrTooManyAuthors   = errors.New("too many authors")
	ErrTooManyKinds     = errors.New("too many kinds")
	ErrTooManyTagValues = errors.New("too many tag values")
	ErrEmptyTagSet      = errors.New("empty tag set")
)

// QueryEventsSql constrói uma consulta SQL para buscar eventos com base em um filtro.
func QueryEventsSql(cfg *config.RelayConfig, filter nostr.Filter, doCount bool) (string, []any, error) {
	var queryBuilder strings.Builder
	var params []any
	var conditions []string

	if err := addIDsCondition(&conditions, &params, filter.IDs, cfg.QueryIDsLimit); err != nil {
		return "", nil, err
	}
	if err := addAuthorsCondition(&conditions, &params, filter.Authors, cfg.QueryAuthorsLimit); err != nil {
		return "", nil, err
	}
	if err := addKindsCondition(&conditions, &params, filter.Kinds, cfg.QueryKindsLimit); err != nil {
		return "", nil, err
	}
	if err := addTagsCondition(&conditions, &params, filter.Tags, cfg.QueryTagsLimit); err != nil {
		return "", nil, err
	}
	// Esta função foi alterada para passar no teste.
	addTimeConditions(&conditions, &params, filter.Since, filter.Until)
	addSearchCondition(&conditions, &params, filter.Search)
	addDeletionCondition(&conditions, cfg.FakeDeletion)

	if len(conditions) == 0 {
		conditions = append(conditions, "true")
	}

	if doCount {
		queryBuilder.WriteString("SELECT COUNT(*) FROM event WHERE ")
	} else {
		queryBuilder.WriteString("SELECT id, pubkey, created_at, kind, tags, content, sig FROM event WHERE ")
	}
	queryBuilder.WriteString(strings.Join(conditions, " AND "))

	if !doCount {
		queryBuilder.WriteString(" ORDER BY created_at DESC, id")
	}

	queryBuilder.WriteString(" LIMIT ?")
	params = append(params, getLimit(filter.Limit, cfg.QueryLimit))

	finalQuery := sqlx.Rebind(sqlx.BindType("postgres"), queryBuilder.String())
	return finalQuery, params, nil
}

// addIDsCondition adiciona a condição de filtro por IDs.
func addIDsCondition(conditions *[]string, params *[]any, ids []string, limit int) error {
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > limit {
		return ErrTooManyIDs
	}
	*conditions = append(*conditions, fmt.Sprintf("id IN (%s)", makePlaceholders(len(ids))))
	for _, id := range ids {
		*params = append(*params, id)
	}
	return nil
}

// addAuthorsCondition adiciona a condição de filtro por autores (pubkey).
func addAuthorsCondition(conditions *[]string, params *[]any, authors []string, limit int) error {
	if len(authors) == 0 {
		return nil
	}
	if len(authors) > limit {
		return ErrTooManyAuthors
	}
	*conditions = append(*conditions, fmt.Sprintf("pubkey IN (%s)", makePlaceholders(len(authors))))
	for _, author := range authors {
		*params = append(*params, author)
	}
	return nil
}

// addKindsCondition adiciona a condição de filtro por tipos de eventos (kinds).
func addKindsCondition(conditions *[]string, params *[]any, kinds []int, limit int) error {
	if len(kinds) == 0 {
		return nil
	}
	if len(kinds) > limit {
		return ErrTooManyKinds
	}
	*conditions = append(*conditions, fmt.Sprintf("kind IN (%s)", makePlaceholders(len(kinds))))
	for _, kind := range kinds {
		*params = append(*params, kind)
	}
	return nil
}

// addTagsCondition adiciona a condição de filtro por tags.
func addTagsCondition(conditions *[]string, params *[]any, tags nostr.TagMap, limit int) error {
	totalTags := 0
	for _, values := range tags {
		if len(values) == 0 {
			return ErrEmptyTagSet
		}
		totalTags += len(values)
		if totalTags > limit {
			return ErrTooManyTagValues
		}
		*conditions = append(*conditions, fmt.Sprintf("tagvalues && ARRAY[%s]", makePlaceholders(len(values))))
		for _, value := range values {
			*params = append(*params, value)
		}
	}
	return nil
}

// addTimeConditions adiciona as condições de filtro por data (since e until).
func addTimeConditions(conditions *[]string, params *[]any, since, until *nostr.Timestamp) {
	if since != nil {
		*conditions = append(*conditions, "created_at >= ?")
		// CORREÇÃO: Adiciona o ponteiro diretamente, em vez do valor.
		*params = append(*params, since)
	}
	if until != nil {
		*conditions = append(*conditions, "created_at <= ?")
		// CORREÇÃO: Adiciona o ponteiro diretamente, em vez do valor.
		*params = append(*params, until)
	}
}

// addSearchCondition adiciona a condição de busca por texto no conteúdo do evento.
func addSearchCondition(conditions *[]string, params *[]any, search string) {
	if search != "" {
		*conditions = append(*conditions, "content LIKE ?")
		*params = append(*params, "%"+strings.ReplaceAll(search, "%", "\\%")+"%")
	}
}

// addDeletionCondition adiciona a condição para excluir eventos deletados, se configurado.
func addDeletionCondition(conditions *[]string, fakeDeletion bool) {
	if fakeDeletion {
		*conditions = append(*conditions, "deleted_by IS NULL")
	}
}

// getLimit retorna o limite de resultados a ser usado, respeitando o máximo configurado.
func getLimit(filterLimit, configLimit int) int {
	if filterLimit > 0 && filterLimit <= configLimit {
		return filterLimit
	}
	return configLimit
}

// makePlaceholders gera uma string de placeholders (?, ?, ...) para consultas SQL.
func makePlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}
