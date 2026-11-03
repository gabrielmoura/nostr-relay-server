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
	scope := core.Scope{Name: "public", Version: 1}
	if _, err := authorizer.Authorize(context.Background(), core.Principal{ID: "192.0.2.1"}, core.ActionOpenSession, scope); err == nil {
		t.Fatal("expected denied peer")
	}
	if _, err := authorizer.Authorize(context.Background(), core.Principal{ID: "10.1.2.3"}, core.ActionOpenSession, scope); err != nil {
		t.Fatal(err)
	}
	for _, denied := range []core.Scope{{Name: "private", Version: 1}, {Name: "public", Version: 2}, {Name: "public", Version: 1, Hash: [32]byte{1}}} {
		if _, err := authorizer.Authorize(context.Background(), core.Principal{ID: "10.1.2.3"}, core.ActionOpenSession, denied); err == nil {
			t.Fatalf("scope %#v was allowed", denied)
		}
	}
}

func TestAllowlistMappedIPv4(t *testing.T) {
	authorizer, err := NewAllowlistAuthorizer([]string{"::ffff:192.0.2.1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = authorizer.Authorize(context.Background(), core.Principal{ID: "192.0.2.1"}, core.ActionOpenSession, core.Scope{Name: "public", Version: 1})
	if err != nil {
		t.Fatalf("mapped IPv4 address was rejected: %v", err)
	}
}

func TestAllowlistMappedIPv4Prefix(t *testing.T) {
	authorizer, err := NewAllowlistAuthorizer([]string{"::ffff:192.0.2.0/120"})
	if err != nil {
		t.Fatal(err)
	}
	scope := core.Scope{Name: "public", Version: 1}
	if _, err := authorizer.Authorize(context.Background(), core.Principal{ID: "192.0.2.42"}, core.ActionOpenSession, scope); err != nil {
		t.Fatal(err)
	}
	if _, err := authorizer.Authorize(context.Background(), core.Principal{ID: "192.0.3.42"}, core.ActionOpenSession, scope); err == nil {
		t.Fatal("outside prefix was allowed")
	}
	if _, err := NewAllowlistAuthorizer([]string{"::ffff:192.0.2.0/95"}); err == nil {
		t.Fatal("ambiguous mapped prefix accepted")
	}
}
