// Package dmimport defines the narrow authorization boundary for browser DM imports.
package dmimport

import "github.com/nbd-wtf/go-nostr"

// Admission can only be created by Authorize. Its identity is intentionally
// private so worker callers cannot manufacture a policy bypass.
type Admission struct{ pubkey string }

func Authorize(authed string, evt *nostr.Event) (Admission, bool) {
	if !Allowed(authed, evt) {
		return Admission{}, false
	}
	return Admission{pubkey: authed}, true
}

func (a Admission) ValidFor(evt *nostr.Event) bool { return a.pubkey != "" && Allowed(a.pubkey, evt) }

// Allowed reports whether an authenticated NIP-42 identity is involved in an
// opaque DM envelope. It deliberately never decrypts or interprets content.
func Allowed(authed string, evt *nostr.Event) bool {
	if authed == "" || evt == nil {
		return false
	}
	switch evt.Kind {
	case 1059:
		return hasPTag(evt, authed)
	case nostr.KindEncryptedDirectMessage:
		return evt.PubKey == authed || hasPTag(evt, authed)
	case 10050:
		return evt.PubKey == authed
	default:
		return IsOwnedImportKind(evt.Kind) && evt.PubKey == authed
	}
}

func hasPTag(evt *nostr.Event, pubkey string) bool {
	for _, tag := range evt.Tags {
		if len(tag) >= 2 && tag[0] == "p" && tag[1] == pubkey {
			return true
		}
	}
	return false
}

func IsImportKind(kind int) bool {
	return kind == 1059 || kind == nostr.KindEncryptedDirectMessage || kind == 10050 || IsOwnedImportKind(kind)
}

// IsOwnedImportKind identifies non-DM events that can be copied only by their
// author after a NIP-42 authentication.
func IsOwnedImportKind(kind int) bool {
	switch kind {
	case nostr.KindProfileMetadata,
		3,
		1,
		7, 20, 21, 22,
		6, 16,
		1111, 1063,
		10000, 10002, 10003, 10063,
		30000, 30023, 30024, 30078:
		return true
	default:
		return false
	}
}

// VisibleTo applies recipient privacy at delivery time, including live events.
func VisibleTo(authed string, evt *nostr.Event) bool {
	return VisibleToWithConcord(authed, evt, false)
}

// VisibleToWithConcord applies recipient privacy to NIP-17 while letting the
// caller explicitly opt into the separately author-scoped Concord policy.
func VisibleToWithConcord(authed string, evt *nostr.Event, concordEnabled bool) bool {
	if evt == nil || (evt.Kind != 1059 && evt.Kind != nostr.KindEncryptedDirectMessage) {
		return true
	}
	// Concord wraps do not carry a recipient tag. Their author-scoped query
	// policy is enforced separately; applying NIP-17 recipient visibility here
	// would hide every valid Concord stream event.
	if evt.Kind == 1059 && concordEnabled && !hasAnyPTag(evt) {
		return true
	}
	return Allowed(authed, evt)
}

func hasAnyPTag(evt *nostr.Event) bool {
	for _, tag := range evt.Tags {
		if len(tag) >= 2 && tag[0] == "p" {
			return true
		}
	}
	return false
}
