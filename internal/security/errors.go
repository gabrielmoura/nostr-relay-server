package security

import (
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

type Prefix string

const (
	PrefixRateLimited Prefix = "rate-limited"
	PrefixBlocked     Prefix = "blocked"
	PrefixInvalid     Prefix = "invalid"
	PrefixRestricted  Prefix = "restricted"
	PrefixPoW         Prefix = "pow"
)

func Reason(prefix Prefix, message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return string(prefix)
	}
	return string(prefix) + ": " + message
}

func EventReject(eventID string, prefix Prefix, message string) nostr.OKEnvelope {
	return nostr.OKEnvelope{EventID: eventID, OK: false, Reason: Reason(prefix, message)}
}

func ClosedReject(subID string, prefix Prefix, message string) nostr.ClosedEnvelope {
	return nostr.ClosedEnvelope{SubscriptionID: subID, Reason: Reason(prefix, message)}
}

func ClosedRejectReason(subID, reason string) nostr.ClosedEnvelope {
	return nostr.ClosedEnvelope{SubscriptionID: subID, Reason: strings.TrimSpace(reason)}
}
