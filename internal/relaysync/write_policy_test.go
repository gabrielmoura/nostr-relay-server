package relaysync

import (
	"context"
	"errors"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/policies"
	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
)

func TestRelaySyncWritePolicyFailsWhenPolicyUnavailable(t *testing.T) {
	previous := policies.P
	policies.P = nil
	t.Cleanup(func() { policies.P = previous })
	if err := relaySyncWritePolicy(context.Background(), core.Event{}); !errors.Is(err, db.ErrRelaySyncPolicyUnavailable) {
		t.Fatalf("error = %v", err)
	}
}

func TestRelaySyncWritePolicyRejectsProtectedEvent(t *testing.T) {
	previous := policies.P
	policies.P = &policies.Policies{Config: &config.Config{NIP70: config.NIP70Config{Enabled: true}}}
	t.Cleanup(func() { policies.P = previous })
	err := relaySyncWritePolicy(context.Background(), core.Event{Tags: []core.Tag{{"-"}}})
	if err == nil || errors.Is(err, db.ErrRelaySyncPolicyUnavailable) {
		t.Fatalf("protected event error = %v", err)
	}
}

func TestRelaySyncPolicyRejectionDistinguishesDependencyFailures(t *testing.T) {
	for _, tc := range []struct {
		reason    string
		retryable bool
	}{
		{"error: failed to load group state", true},
		{"error: database unavailable", true},
		{"restricted: event moderation lookup failed", true},
		{"restricted: group access denied", false},
		{"blocked: banned", false},
	} {
		err := relaySyncPolicyRejection(tc.reason)
		if got := errors.Is(err, db.ErrRelaySyncPolicyUnavailable); got != tc.retryable {
			t.Errorf("reason=%q retryable=%v want=%v", tc.reason, got, tc.retryable)
		}
	}
}
