package helper

import (
	"strings"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
)

func TestBuildChronologicalQuery(t *testing.T) {
	for _, tc := range []struct {
		asc   bool
		order string
	}{{true, "ORDER BY created_at ASC, id ASC"}, {false, "ORDER BY created_at DESC, id ASC"}} {
		query, _, err := BuildChronologicalQuery(nostr.Filter{Limit: 10, Search: "nostr"}, &config.RelayConfig{}, tc.asc)
		if err != nil || !strings.Contains(query, tc.order) {
			t.Fatalf("query=%q err=%v", query, err)
		}
	}
}
