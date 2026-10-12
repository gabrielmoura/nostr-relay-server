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
