package relaysync

import (
	"context"
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
}
