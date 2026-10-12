package db

import (
	"testing"

	relaysync "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
)

func TestRelaySyncEventToNostrRejectsPostgresOverflow(t *testing.T) {
	event := relaysync.Event{CreatedAt: 1 << 40}
	if _, err := relaySyncEventToNostr(event); err == nil {
		t.Fatal("expected integer range error")
	}
}

func TestNewRelaySyncEventWriterRequiresQueries(t *testing.T) {
	if _, err := NewRelaySyncEventWriter(nil, relaysync.DefaultLimits(), nil); err == nil {
		t.Fatal("expected missing queries error")
	}
}
