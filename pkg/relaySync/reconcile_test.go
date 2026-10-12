package relaysync

import (
	"context"
	"testing"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

func TestReconcileServiceRespondsWithinSessionPartition(t *testing.T) {
	storage := NewMemoryStorage(nil, DefaultLimits())
	event := testValidEvent(t)
	if _, err := storage.WriteEvents(context.Background(), EventBatch{Events: []Event{event}}); err != nil {
		t.Fatal(err)
	}
	principal := Principal{ID: "peer"}
	scope := Scope{Name: "public", Version: 1}
	authorizer := NewMemoryPeerAuthorizer()
	authorizer.Grant(principal, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession, ActionPull}})
	sessions, err := NewSessionService(SessionServiceOptions{Store: NewMemorySessionStore(nil), Snapshots: storage, Authorizer: authorizer})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := sessions.Open(context.Background(), principal, &relaysyncv2.OpenSessionRequest{ProtocolVersion: 2, Scope: &relaysyncv2.Scope{Name: scope.Name, Version: scope.Version}})
	if err != nil {
		t.Fatal(err)
	}
	var sessionID SessionID
	copy(sessionID[:], opened.SessionId)
	snapshot, ok := sessions.Snapshot(sessionID)
	if !ok {
		t.Fatal("session snapshot missing")
	}
	partitions := snapshot.Descriptor().Partitions
	if len(partitions) != 1 {
		t.Fatalf("partitions = %d, want 1", len(partitions))
	}
	remoteID := event.ID
	remoteID[0]++
	remote, err := NewNegentropy([]EventMetadata{{ID: remoteID, CreatedAt: event.CreatedAt}}, NegentropyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	frame, err := remote.Initiate()
	if err != nil {
		t.Fatal(err)
	}
	reconcile, err := NewReconcileService(sessions, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	partitionID := PartitionIdentity(partitions[0])
	response, err := reconcile.Reconcile(context.Background(), principal, sessionID, &relaysyncv2.ReconcileRequest{
		PartitionId:       partitionID[:],
		NegentropyPayload: frame,
		Round:             1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.NegentropyPayload) == 0 || response.Round != 1 {
		t.Fatalf("unexpected reconcile response: %+v", response)
	}
}
