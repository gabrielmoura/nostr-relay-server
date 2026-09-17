package helper

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSearchQuery(t *testing.T) {
	tests := []struct {
		name       string
		search     string
		terms      []string
		extensions map[string]string
		hasInput   bool
	}{
		{
			name:     "free text",
			search:   "best nostr apps",
			terms:    []string{"best", "nostr", "apps"},
			hasInput: true,
		},
		{
			name:   "recognized extensions are removed",
			search: "nostr DOMAIN:example.com language:pt include:spam",
			terms:  []string{"nostr"},
			extensions: map[string]string{
				"domain":   "example.com",
				"language": "pt",
				"include":  "spam",
			},
			hasInput: true,
		},
		{
			name:   "last repeated extension wins",
			search: "nsfw:false nsfw:true",
			extensions: map[string]string{
				"nsfw": "true",
			},
			hasInput: true,
		},
		{
			name:     "unknown extension remains text",
			search:   "topic:nostr",
			terms:    []string{"topic:nostr"},
			hasInput: true,
		},
		{
			name:       "empty extension value",
			search:     "domain:",
			extensions: map[string]string{"domain": ""},
			hasInput:   true,
		},
		{
			name:     "emoji only",
			search:   "👩🏽‍💻 🇧🇷",
			hasInput: true,
		},
		{
			name:     "punctuation is preserved for plainto tsquery",
			search:   "hello, world!",
			terms:    []string{"hello,", "world!"},
			hasInput: true,
		},
		{
			name:     "blank",
			search:   " \t ",
			hasInput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSearchQuery(tt.search)
			terms := tt.terms
			if terms == nil {
				terms = []string{}
			}
			extensions := tt.extensions
			if extensions == nil {
				extensions = map[string]string{}
			}
			require.Equal(t, terms, got.Terms)
			require.Equal(t, extensions, got.Extensions)
			require.Equal(t, tt.hasInput, got.HasInput)
		})
	}
}

func TestParseSearchQuery_LongInputDoesNotPanic(t *testing.T) {
	query := ParseSearchQuery(strings.Repeat("nostr ", maxSearchLength))
	require.NotEmpty(t, query.Terms)
}
