// Package embed provides static assets bundled into the relay binary.
package embed

import _ "embed"

// NostrPNG is the relay icon served from /nostr.png.
//
//go:embed nostr.png
var NostrPNG []byte
