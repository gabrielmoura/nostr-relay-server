package helper

import (
	"strings"
)

const maxSearchLength = 4096

// SearchQuery is the NIP-50 search input split into full-text terms and
// recognized extensions. Extensions are parsed for forward compatibility but
// are not SQL filters yet.
type SearchQuery struct {
	Terms      []string
	Extensions map[string]string
	HasInput   bool
}

// ParseSearchQuery parses the NIP-50 key:value extension syntax. Repeated
// supported extensions keep their final value.
func ParseSearchQuery(search string) SearchQuery {
	query := SearchQuery{
		Terms:      []string{},
		Extensions: map[string]string{},
		HasInput:   search != "",
	}

	for _, token := range strings.Fields(search) {
		key, value, hasColon := strings.Cut(token, ":")
		if hasColon && isSearchExtension(key) {
			query.Extensions[strings.ToLower(key)] = value
			continue
		}

		if term := removeEmoji(token); term != "" {
			query.Terms = append(query.Terms, term)
		}
	}

	return query
}

func isSearchExtension(key string) bool {
	switch strings.ToLower(key) {
	case "domain", "language", "sentiment", "nsfw", "include":
		return true
	default:
		return false
	}
}

func removeEmoji(value string) string {
	return strings.Map(func(r rune) rune {
		if isEmojiRune(r) {
			return -1
		}
		return r
	}, value)
}

func isEmojiRune(r rune) bool {
	return r >= 0x1F300 && r <= 0x1FAFF ||
		r >= 0x2600 && r <= 0x27BF ||
		r >= 0x2190 && r <= 0x21FF ||
		r >= 0x2B00 && r <= 0x2BFF ||
		r >= 0x1F1E6 && r <= 0x1F1FF ||
		r == 0xFE0F || r == 0x200D
}
