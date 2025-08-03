package helper

import (
	"errors"
	"fmt"
	"github.com/nbd-wtf/go-nostr"
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	_ "github.com/nbd-wtf/go-nostr"
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

func ptrTime(t time.Time) *time.Time {
	return &t
}
