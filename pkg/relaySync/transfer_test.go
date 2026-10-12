package relaysync

import (
	"context"
	"errors"
	"testing"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

func TestTransferServicePushIdempotency(t *testing.T) {
	storage := NewMemoryStorage(nil, DefaultLimits())
	authorizer := NewMemoryPeerAuthorizer()
	scope := Scope{Name: "public", Version: 1}
	authorizer.Grant(Principal{ID: "peer"}, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession, ActionPush, ActionPull}})
	sessions, err := NewSessionService(SessionServiceOptions{Store: NewMemorySessionStore(nil), Authorizer: authorizer})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := sessions.Open(context.Background(), Principal{ID: "peer"}, &relaysyncv2.OpenSessionRequest{ProtocolVersion: 2, Scope: &relaysyncv2.Scope{Name: "public", Version: 1}})
	if err != nil {
		t.Fatal(err)
	}
	var id SessionID
	copy(id[:], opened.SessionId)
	transfer, err := NewTransferService(sessions, storage, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	event := testValidEvent(t)
	request := &relaysyncv2.PushEventsRequest{OperationId: "op-1", Events: []*relaysyncv2.Event{EventToProto(event)}}
	first, err := transfer.Push(context.Background(), Principal{ID: "peer"}, id, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := transfer.Push(context.Background(), Principal{ID: "peer"}, id, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Receipts) != 1 || first.Receipts[0].Status != relaysyncv2.WriteStatus_WRITE_ACCEPTED || second.Receipts[0].Status != relaysyncv2.WriteStatus_WRITE_ACCEPTED {
		t.Fatalf("receipts: %+v %+v", first, second)
	}
	if err := sessions.Complete(context.Background(), Principal{ID: "peer"}, id, opened.ResumeToken); err != nil {
		t.Fatal(err)
	}
	if _, err := transfer.Push(context.Background(), Principal{ID: "peer"}, id, request); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Push after completion = %v, want ErrNotFound", err)
	}
}

func TestTransferServiceRequiresOperationAction(t *testing.T) {
	storage := NewMemoryStorage(nil, DefaultLimits())
	principal := Principal{ID: "peer"}
	scope := Scope{Name: "public", Version: 1}
	authorizer := NewMemoryPeerAuthorizer()
	authorizer.Grant(principal, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession}})
	sessions, err := NewSessionService(SessionServiceOptions{Store: NewMemorySessionStore(nil), Authorizer: authorizer})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := sessions.Open(context.Background(), principal, &relaysyncv2.OpenSessionRequest{ProtocolVersion: 2, Scope: &relaysyncv2.Scope{Name: scope.Name, Version: scope.Version}})
	if err != nil {
		t.Fatal(err)
	}
	var id SessionID
	copy(id[:], opened.SessionId)
	transfer, err := NewTransferService(sessions, storage, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	_, err = transfer.Pull(context.Background(), principal, id, &relaysyncv2.PullEventsRequest{})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Pull error = %v, want ErrUnauthorized", err)
	}
}
