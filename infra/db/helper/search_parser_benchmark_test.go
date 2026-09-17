package helper

import (
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

func BenchmarkParseSearchQuery(b *testing.B) {
	search := "best nostr relay DOMAIN:example.com language:pt include:spam 👩🏽‍💻"
	b.ReportAllocs()

	for b.Loop() {
		ParseSearchQuery(search)
	}
}

func BenchmarkQueryEventsSQL_NIP50WithArraysAndTags(b *testing.B) {
	cfg := testConfig()
	filter := nostr.Filter{
		IDs:     []string{"id-2", "id-1"},
		Authors: []string{"pubkey-2", "pubkey-1"},
		Kinds:   []int{1, 30023},
		Tags: nostr.TagMap{
			"p": {"pubkey-1", "pubkey-2"},
		},
		Search: "nostr relay",
		Limit:  20,
	}
	b.ReportAllocs()

	for b.Loop() {
		_, _, err := QueryEventsSql(cfg, filter, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}
