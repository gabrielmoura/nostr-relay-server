package relaysync

import (
	"context"
	"errors"
	"testing"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

type recordingEventWriter struct {
	batches []EventBatch
}

func (w *recordingEventWriter) WriteEvents(_ context.Context, batch EventBatch) ([]WriteResult, error) {
	w.batches = append(w.batches, batch)
	results := make([]WriteResult, len(batch.Events))
	for index, event := range batch.Events {
		results[index] = WriteResult{ID: event.ID, Status: WriteAccepted}
	}
	return results, nil
}

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

func TestTransferServicePushAllowsNewOperationForDifferentBatch(t *testing.T) {
	storage := NewMemoryStorage(nil, DefaultLimits())
	principal := Principal{ID: "peer"}
	scope := Scope{Name: "public", Version: 1}
	authorizer := NewMemoryPeerAuthorizer()
	authorizer.Grant(principal, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession, ActionPush}})
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
	first := testValidEvent(t)
	second := first
	second.ID[0] ^= 0xff
	request := &relaysyncv2.PushEventsRequest{OperationId: "partition-a-0", Events: []*relaysyncv2.Event{EventToProto(first)}}
	if _, err := transfer.Push(context.Background(), principal, id, request); err != nil {
		t.Fatal(err)
	}
	if _, err := transfer.Push(context.Background(), principal, id, &relaysyncv2.PushEventsRequest{OperationId: "partition-b-0", Events: []*relaysyncv2.Event{EventToProto(second)}}); err != nil {
		t.Fatalf("new batch operation failed: %v", err)
	}
	if _, err := transfer.Push(context.Background(), principal, id, &relaysyncv2.PushEventsRequest{OperationId: request.OperationId, Events: []*relaysyncv2.Event{EventToProto(second)}}); !errors.Is(err, ErrConflict) {
		t.Fatalf("reused operation with different payload = %v, want ErrConflict", err)
	}
}

func TestTransferServicePushRejectsInvalidEventAndWritesRemainingEvents(t *testing.T) {
	principal := Principal{ID: "peer"}
	scope := Scope{Name: "public", Version: 1}
	authorizer := NewMemoryPeerAuthorizer()
	authorizer.Grant(principal, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession, ActionPush}})
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
	writer := &recordingEventWriter{}
	transfer, err := NewTransferService(sessions, writer, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}

	first := testValidEvent(t)
	invalid := EventToProto(testValidEvent(t))
	invalid.Tags = make([]*relaysyncv2.Tag, DefaultLimits().MaxTagCount+1)
	for index := range invalid.Tags {
		invalid.Tags[index] = &relaysyncv2.Tag{}
	}
	last := testValidEvent(t)
	last.ID[0] ^= 0xff
	request := &relaysyncv2.PushEventsRequest{
		OperationId: "mixed-events",
		Events: []*relaysyncv2.Event{
			EventToProto(first),
			invalid,
			EventToProto(last),
		},
	}

	response, err := transfer.Push(context.Background(), principal, id, request)
	if err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	if len(writer.batches) != 1 || len(writer.batches[0].Events) != 2 {
		t.Fatalf("writer batches = %#v, want one batch with two valid events", writer.batches)
	}
	if writer.batches[0].Events[0].ID != first.ID || writer.batches[0].Events[1].ID != last.ID {
		t.Fatalf("writer event order = %#v, want first and last valid events", writer.batches[0].Events)
	}
	if len(response.Receipts) != len(request.Events) {
		t.Fatalf("receipt count = %d, want %d", len(response.Receipts), len(request.Events))
	}
	if response.Receipts[0].Status != relaysyncv2.WriteStatus_WRITE_ACCEPTED || response.Receipts[2].Status != relaysyncv2.WriteStatus_WRITE_ACCEPTED {
		t.Fatalf("valid event receipts = %#v", response.Receipts)
	}
	rejected := response.Receipts[1]
	if rejected.Status != relaysyncv2.WriteStatus_WRITE_REJECTED_PERMANENT || rejected.Code != invalidEventReceiptCode {
		t.Fatalf("invalid event receipt = %#v", rejected)
	}
	if rejected.Message == "" {
		t.Fatal("invalid event receipt must preserve the validation reason")
	}
	if string(rejected.Id) != string(invalid.Id) {
		t.Fatalf("invalid event receipt ID = %x, want %x", rejected.Id, invalid.Id)
	}
}

func TestTransferServicePushUsesZeroIDForInvalidEventIdentifier(t *testing.T) {
	principal := Principal{ID: "peer"}
	scope := Scope{Name: "public", Version: 1}
	authorizer := NewMemoryPeerAuthorizer()
	authorizer.Grant(principal, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession, ActionPush}})
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
	writer := &recordingEventWriter{}
	transfer, err := NewTransferService(sessions, writer, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}

	invalid := EventToProto(testValidEvent(t))
	invalid.Id = []byte{1}
	response, err := transfer.Push(context.Background(), principal, id, &relaysyncv2.PushEventsRequest{OperationId: "invalid-id", Events: []*relaysyncv2.Event{invalid}})
	if err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	if len(writer.batches) != 0 {
		t.Fatalf("writer batches = %d, want 0", len(writer.batches))
	}
	if len(response.Receipts) != 1 || response.Receipts[0].Status != relaysyncv2.WriteStatus_WRITE_REJECTED_PERMANENT || response.Receipts[0].Code != invalidEventReceiptCode {
		t.Fatalf("invalid ID receipt = %#v", response.Receipts)
	}
	if len(response.Receipts[0].Id) != eventIDSize {
		t.Fatalf("invalid ID receipt length = %d, want %d", len(response.Receipts[0].Id), eventIDSize)
	}
	for _, value := range response.Receipts[0].Id {
		if value != 0 {
			t.Fatalf("invalid ID receipt = %x, want zero ID", response.Receipts[0].Id)
		}
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
