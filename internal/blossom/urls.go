package blossom

import (
	"net/url"
	"strings"
)

// BrowserURL returns a same-origin URL for an object hash that can be safely
// rendered by a browser. It intentionally points at the relay's public blob
// proxy instead of the configured media origin, which may be private (for
// example .onion or .i2p) or unreachable from the operator's browser.
func BrowserURL(hash string) string {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return ""
	}
	return "/blob/" + url.PathEscape(hash)
}
