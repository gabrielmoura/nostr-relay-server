package policies

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
)

// ContentSpamChecker evaluates an event for cross-account content spam.
type ContentSpamChecker interface {
	Check(ctx context.Context, evt *nostr.Event) (reject bool, flagged bool, reason string)
}
