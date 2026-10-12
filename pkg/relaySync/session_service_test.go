package relaysync

import (
	"context"
	"errors"
	"testing"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

func TestSessionServiceIdempotencyAndTokenValidation(t *testing.T) {
	store := NewMemorySessionStore(nil)
	authorizer := NewMemoryPeerAuthorizer()
	scope := Scope{Name: "public", Version: 1}
	authorizer.Grant(Principal{ID: "peer-a"}, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession}})
	service, err := NewSessionService(SessionServiceOptions{Store: store, Authorizer: authorizer})
	if err != nil {
		t.Fatal(err)
	}
	request := &relaysyncv2.OpenSessionRequest{ProtocolVersion: 2, IdempotencyKey: []byte("same"), Scope: &relaysyncv2.Scope{Name: "public", Version: 1}}
	first, err := service.Open(context.Background(), Principal{ID: "peer-a"}, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Open(context.Background(), Principal{ID: "peer-a"}, request)
	if err != nil {
		t.Fatal(err)
	}
	if string(first.SessionId) != string(second.SessionId) || string(first.ResumeToken) != string(second.ResumeToken) {
		t.Fatal("idempotent retry created a different session")
	}
	var id SessionID
	copy(id[:], first.SessionId)
	if err := service.Delete(context.Background(), Principal{ID: "peer-a"}, id, []byte("wrong")); !errors.Is(err, ErrSessionToken) {
		t.Fatalf("wrong token error = %v", err)
	}
	if err := service.Delete(context.Background(), Principal{ID: "peer-a"}, id, first.ResumeToken); err != nil {
		t.Fatal(err)
	}
}

func TestSessionServiceCompleteDiscardsCycleState(t *testing.T) {
	store := NewMemorySessionStore(nil)
	authorizer := NewMemoryPeerAuthorizer()
	scope := Scope{Name: "public", Version: 1}
	authorizer.Grant(Principal{ID: "peer-a"}, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession}})
	storage := NewMemoryStorage(nil, DefaultLimits())
	service, err := NewSessionService(SessionServiceOptions{
		Store:      store,
		Snapshots:  storage,
		Authorizer: authorizer,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := &relaysyncv2.OpenSessionRequest{
		ProtocolVersion: 2,
		IdempotencyKey:  []byte("cycle-1"),
		Scope:           &relaysyncv2.Scope{Name: "public", Version: 1},
	}
	opened, err := service.Open(context.Background(), Principal{ID: "peer-a"}, request)
	if err != nil {
		t.Fatal(err)
	}
	var id SessionID
	copy(id[:], opened.SessionId)
	if _, ok := service.Snapshot(id); !ok {
		t.Fatal("expected snapshot to be retained during cycle")
	}
	if err := service.Complete(context.Background(), Principal{ID: "peer-a"}, id, opened.ResumeToken); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetSession(context.Background(), id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("session after completion = %v, want ErrNotFound", err)
	}
	if _, ok := service.Snapshot(id); ok {
		t.Fatal("snapshot retained after completion")
	}
	reopened, err := service.Open(context.Background(), Principal{ID: "peer-a"}, request)
	if err != nil {
		t.Fatal(err)
	}
	if string(reopened.SessionId) == string(opened.SessionId) {
		t.Fatal("completed cycle retained idempotency result")
	}
}

func TestSessionServiceCloseDiscardsSnapshots(t *testing.T) {
	store := NewMemorySessionStore(nil)
	authorizer := NewMemoryPeerAuthorizer()
	scope := Scope{Name: "public", Version: 1}
	principal := Principal{ID: "peer-a"}
	authorizer.Grant(principal, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession}})
	service, err := NewSessionService(SessionServiceOptions{Store: store, Snapshots: NewMemoryStorage(nil, DefaultLimits()), Authorizer: authorizer})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := service.Open(context.Background(), principal, &relaysyncv2.OpenSessionRequest{ProtocolVersion: 2, Scope: &relaysyncv2.Scope{Name: scope.Name, Version: scope.Version}})
	if err != nil {
		t.Fatal(err)
	}
	var id SessionID
	copy(id[:], opened.SessionId)
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := service.Snapshot(id); ok {
		t.Fatal("snapshot retained after service close")
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
}
