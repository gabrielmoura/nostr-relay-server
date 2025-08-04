package helper

import (
	"errors"
	"fmt"
	"github.com/nbd-wtf/go-nostr"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/stretchr/testify/assert"
)

var conf *config.RelayConfig

func init() {
	conf = &config.RelayConfig{
		QueryIDsLimit:     5,
		QueryAuthorsLimit: 10,
		QueryKindsLimit:   10,
		QueryTagsLimit:    10,
		QueryLimit:        5,
	}
}

func TestQueryEventsSql_Basic(t *testing.T) {

	filter := nostr.Filter{
		IDs:     []string{"id1", "id2"},
		Authors: []string{"author1"},
		Kinds:   []int{1},
		Tags: map[string][]string{
			"#p": {"val1", "val2"},
		},
		Search: "test",
		Limit:  10,
	}

	query, params, err := QueryEventsSql(conf, filter, false)
	assert.NoError(t, err)
	assert.Contains(t, query, `SELECT id, pubkey, created_at, kind, tags, content, sig FROM event WHERE`)
	assert.Equal(t, 8, len(params)) // 2 IDs + 1 author + 1 kind + 2 tags + limit

	var strParams string
	for _, str := range params {
		strParams += fmt.Sprintf("%s", str)
	}

	println("Params: ", strParams)
	println("Query: ", query)
}

func TestQueryEventsSql_TooManyIDs(t *testing.T) {
	filter := nostr.Filter{
		IDs: []string{"1", "2", "3", "4", "5", "6"}, // ultrapassa limite de 5
	}

	_, _, err := QueryEventsSql(conf, filter, false)
	assert.True(t, errors.Is(err, ErrTooManyIDs))
}

func TestQueryEventsSql_EmptyTagSet(t *testing.T) {
	filter := nostr.Filter{
		Tags: map[string][]string{
			"#e": {},
		},
	}

	_, _, err := QueryEventsSql(conf, filter, false)
	assert.True(t, errors.Is(err, ErrEmptyTagSet))
}

func TestQueryEventsSql_TooManyTagValues(t *testing.T) {
	filter := nostr.Filter{
		Tags: map[string][]string{
			"#e": {"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}, // 11 > limite de 10
		},
	}

	_, _, err := QueryEventsSql(conf, filter, false)
	assert.True(t, errors.Is(err, ErrTooManyTagValues))
}

func TestQueryEventsSql_DefaultLimit(t *testing.T) {
	filter := nostr.Filter{
		Limit: 0, // limite inválido
	}

	_, params, err := QueryEventsSql(conf, filter, false)
	assert.NoError(t, err)
	assert.Equal(t, conf.QueryLimit, params[len(params)-1])
}

func TestQueryEventsSql_DoCount(t *testing.T) {
	filter := nostr.Filter{
		IDs: []string{"id1"},
	}

	query, _, err := QueryEventsSql(conf, filter, true)
	assert.NoError(t, err)
	assert.Contains(t, query, "SELECT COUNT(*) FROM event")
}

// TestQueryEventsSql_DoCountWithComplexFilter tests the count query with a complex filter
// Garantir que a consulta COUNT(*) seja construída corretamente com múltiplos filtros.
func TestQueryEventsSql_DoCountWithComplexFilter(t *testing.T) {
	filter := nostr.Filter{
		Authors: []string{"author1", "author2"},
		Kinds:   []int{1, 7},
		Search:  "nostr",
	}

	query, params, err := QueryEventsSql(conf, filter, true)
	assert.NoError(t, err)

	assert.Contains(t, query, "SELECT COUNT(*) FROM event")
	assert.Contains(t, query, "pubkey IN ($1,$2)")
	assert.Contains(t, query, "kind IN ($3,$4)")
	assert.Contains(t, query, "content LIKE $5")
	assert.NotContains(t, query, "ORDER BY") // A contagem não precisa de ordenação
	assert.Equal(t, 6, len(params))          // 2 autores + 2 kinds + 1 search + 1 limit
}

// TestQueryEventsSql_WithFakeDeletionEnabled tests the SQL query generation
// when fake deletion is enabled, ensuring the query includes the deleted_by condition.
func TestQueryEventsSql_WithFakeDeletionDisabled(t *testing.T) {
	// Cria uma configuração local para este teste
	confWithoutDeletion := conf
	confWithoutDeletion.FakeDeletion = false

	filter := nostr.Filter{Authors: []string{"author1"}}
	query, _, err := QueryEventsSql(confWithoutDeletion, filter, false)
	assert.NoError(t, err)
	assert.NotContains(t, query, `deleted_by IS NULL`)
}
func TestQueryEventsSql_WithFakeDeletionEnabled(t *testing.T) {
	// Cria uma configuração local para este teste
	confWithDeletion := conf
	confWithDeletion.FakeDeletion = true

	filter := nostr.Filter{Authors: []string{"author1"}}
	query, _, err := QueryEventsSql(confWithDeletion, filter, false)
	assert.NoError(t, err)
	assert.Contains(t, query, `deleted_by IS NULL`)
}

func TestQueryEventsSql_LimitExceedsConfig(t *testing.T) {
	filter := nostr.Filter{
		Limit: conf.QueryLimit + 100, // Limite muito alto
	}

	_, params, err := QueryEventsSql(conf, filter, false)
	assert.NoError(t, err)
	assert.Equal(t, conf.QueryLimit, params[len(params)-1], "o limite deve ser reduzido para o QueryLimit da configuração")
}
func TestQueryEventsSql_TimeFilters(t *testing.T) {
	now := nostr.Now()
	sinceTime := now - 1000
	untilTime := now - 500

	filter := nostr.Filter{
		Since: &sinceTime,
		Until: &untilTime,
	}

	query, params, err := QueryEventsSql(conf, filter, false)
	assert.NoError(t, err)

	assert.Contains(t, query, `created_at >= $1`)
	assert.Contains(t, query, `created_at <= $2`)
	assert.Equal(t, sinceTime, *params[0].(*nostr.Timestamp))
	assert.Equal(t, untilTime, *params[1].(*nostr.Timestamp))
}
func TestQueryEventsSql_SearchWithWildcardCharacter(t *testing.T) {
	filter := nostr.Filter{
		Search: "uma string com 100% de certeza",
	}

	_, params, err := QueryEventsSql(conf, filter, false)
	assert.NoError(t, err)

	// O parâmetro para a busca deve conter a string de busca com '%' escapado e envolvido por '%' para o LIKE
	expectedSearchParam := `%uma string com 100\% de certeza%`
	assert.Equal(t, expectedSearchParam, params[0])
}
func TestQueryEventsSql_CumulativeTooManyTagValues(t *testing.T) {
	filter := nostr.Filter{
		Tags: map[string][]string{
			"#e": {"v1", "v2", "v3", "v4", "v5"},
			"#p": {"v6", "v7", "v8", "v9", "v10", "v11"}, // Total de valores (5+6) > limite de 10
		},
	}

	_, _, err := QueryEventsSql(conf, filter, false)
	assert.ErrorIs(t, err, ErrTooManyTagValues, "esperado erro ErrTooManyTagValues a partir de tags cumulativas")
}
func TestQueryEventsSql_TooManyKinds(t *testing.T) {
	filter := nostr.Filter{
		Kinds: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, // 11 > limite de 10
	}

	_, _, err := QueryEventsSql(conf, filter, false)
	assert.ErrorIs(t, err, ErrTooManyKinds, "esperado erro ErrTooManyKinds")
}
func TestQueryEventsSql_TooManyAuthors(t *testing.T) {
	filter := nostr.Filter{
		Authors: []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9", "a10", "a11"}, // 11 > limite de 10
	}

	_, _, err := QueryEventsSql(conf, filter, false)
	assert.ErrorIs(t, err, ErrTooManyAuthors, "esperado erro ErrTooManyAuthors")
}
