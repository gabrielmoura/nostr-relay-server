// Package embed provides static assets bundled into the relay binary.
package embed

import _ "embed"

// NostrPNGSHA256 is the SHA-256 checksum of NostrPNG.
const NostrPNGSHA256 = "3f090aff4a230d54105705652b972d89b8a884b171bbbde825fe14221c897a0d"

// NostrPNG is the relay icon served from /nostr.png.
//
//go:embed nostr.png
var NostrPNG []byte
