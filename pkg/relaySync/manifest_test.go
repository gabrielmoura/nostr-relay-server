package relaysync

import (
	"context"
	"testing"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

func TestManifestServicePaginatesSnapshot(t *testing.T) {
	storage := NewMemoryStorage(nil, DefaultLimits())
	event := testValidEvent(t)
	if _, err := storage.WriteEvents(context.Background(), EventBatch{Events: []Event{event}}); err != nil {
		t.Fatal(err)
	}
	authorizer := NewMemoryPeerAuthorizer()
	scope := Scope{Name: "public", Version: 1}
	authorizer.Grant(Principal{ID: "peer"}, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession}})
	sessions, err := NewSessionService(SessionServiceOptions{Store: NewMemorySessionStore(nil), Snapshots: storage, Authorizer: authorizer})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := sessions.Open(context.Background(), Principal{ID: "peer"}, &relaysyncv2.OpenSessionRequest{ProtocolVersion: 2, Scope: &relaysyncv2.Scope{Name: "public", Version: 1}})
	if err != nil {
		t.Fatal(err)
	}
	var id SessionID
	copy(id[:], opened.SessionId)
	manifests, err := NewManifestService(sessions, 1)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := manifests.Next(context.Background(), Principal{ID: "peer"}, id, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Partitions) != 1 || !manifest.Complete {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
}
