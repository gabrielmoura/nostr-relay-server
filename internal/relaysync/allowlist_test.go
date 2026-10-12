package relaysync

import (
	"context"
	"testing"

	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
)

func TestAllowlistAuthorizer(t *testing.T) {
	authorizer, err := NewAllowlistAuthorizer([]string{"127.0.0.1", "10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authorizer.Authorize(context.Background(), core.Principal{ID: "192.0.2.1"}, core.ActionOpenSession, core.Scope{}); err == nil {
		t.Fatal("expected denied peer")
	}
	if _, err := authorizer.Authorize(context.Background(), core.Principal{ID: "10.1.2.3"}, core.ActionOpenSession, core.Scope{}); err != nil {
		t.Fatal(err)
	}
}
