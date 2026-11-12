package policies

import "strings"

// IsTemporaryRejection identifies dependency failures in the current Nostr
// reason contract. Callers that acknowledge durable writes must retry these
// failures instead of reporting them as permanent event rejections.
func IsTemporaryRejection(reason string) bool {
	return strings.HasPrefix(reason, "error:") || reason == "restricted: event moderation lookup failed"
}
