package relaysync

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"google.golang.org/protobuf/proto"
)

type TransferService struct {
	sessions   *SessionService
	writer     EventWriter
	limits     Limits
	mu         sync.Mutex
	operations map[transferOperationKey]transferResult
}
type transferOperationKey struct {
	sessionID   SessionID
	operationID string
}
type transferResult struct {
	hash     [32]byte
	response *relaysyncv2.PushEventsResponse
}

const invalidEventReceiptCode = "INVALID_EVENT"

func NewTransferService(sessions *SessionService, writer EventWriter, limits Limits) (*TransferService, error) {
	if sessions == nil || writer == nil {
		return nil, ErrNotFound
	}
	service := &TransferService{sessions: sessions, writer: writer, limits: limits.WithDefaults(), operations: make(map[transferOperationKey]transferResult)}
	if err := sessions.RegisterFinishHook(service.ForgetSession); err != nil {
		return nil, err
	}
	return service, nil
}

func (t *TransferService) Pull(ctx context.Context, principal Principal, id SessionID, request *relaysyncv2.PullEventsRequest) (*relaysyncv2.PullEventsResponse, error) {
	if request == nil {
		return nil, ErrInvalidEvent
	}
	if _, err := t.sessions.RequireAction(ctx, principal, id, ActionPull); err != nil {
		return nil, err
	}
	if len(request.EventIds) > t.limits.MaxEventsPerBatch || proto.Size(request) > t.limits.MaxBatchBytes {
		return nil, ErrInvalidLimits
	}
	snapshot, ok := t.sessions.Snapshot(id)
	if !ok {
		return nil, ErrSnapshotExpired
	}
	ids := make([]EventID, len(request.EventIds))
	for i, value := range request.EventIds {
		if len(value) != 32 {
			return nil, ErrInvalidIdentifier
		}
		copy(ids[i][:], value)
	}
	iterator, err := snapshot.GetEvents(ctx, ids)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()
	response := &relaysyncv2.PullEventsResponse{}
	for {
		item, ok, err := iterator.Next(ctx)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		if item.Event != nil {
			response.Events = append(response.Events, EventToProto(*item.Event))
		} else {
			response.MissingEventIds = append(response.MissingEventIds, append([]byte(nil), item.ID[:]...))
		}
		if uint32(len(response.Events)) >= request.MaxEvents && request.MaxEvents > 0 {
			break
		}
	}
	return response, nil
}

func (t *TransferService) Push(ctx context.Context, principal Principal, id SessionID, request *relaysyncv2.PushEventsRequest) (*relaysyncv2.PushEventsResponse, error) {
	if request == nil {
		return nil, ErrInvalidEvent
	}
	if _, err := t.sessions.RequireAction(ctx, principal, id, ActionPush); err != nil {
		return nil, err
	}
	if request.OperationId == "" || len(request.Events) > t.limits.MaxEventsPerBatch || proto.Size(request) > t.limits.MaxBatchBytes {
		return nil, ErrInvalidLimits
	}
	events := make([]Event, 0, len(request.Events))
	validReceiptIndexes := make([]int, 0, len(request.Events))
	response := &relaysyncv2.PushEventsResponse{Receipts: make([]*relaysyncv2.EventReceipt, len(request.Events))}
	for index, message := range request.Events {
		event, err := EventFromProto(message, t.limits)
		if err != nil {
			response.Receipts[index] = invalidEventReceipt(message, err)
			continue
		}
		events = append(events, event)
		validReceiptIndexes = append(validReceiptIndexes, index)
	}
	hash := sha256.New()
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(request)
	if err != nil {
		return nil, err
	}
	hash.Write(encoded)
	var payloadHash [32]byte
	copy(payloadHash[:], hash.Sum(nil))
	key := transferOperationKey{sessionID: id, operationID: request.OperationId}
	t.mu.Lock()
	if cached, ok := t.operations[key]; ok {
		t.mu.Unlock()
		if cached.hash != payloadHash {
			return nil, ErrConflict
		}
		return proto.Clone(cached.response).(*relaysyncv2.PushEventsResponse), nil
	}
	t.mu.Unlock()
	if len(events) > 0 {
		results, err := t.writer.WriteEvents(ctx, EventBatch{OperationID: request.OperationId, Events: events})
		if err != nil {
			return nil, err
		}
		if len(results) != len(events) {
			return nil, fmt.Errorf("relaysync: event writer returned %d results for %d events", len(results), len(events))
		}
		for index, result := range results {
			response.Receipts[validReceiptIndexes[index]] = eventReceipt(result)
		}
	}
	t.mu.Lock()
	t.operations[key] = transferResult{hash: payloadHash, response: response}
	t.mu.Unlock()
	return response, nil
}

func invalidEventReceipt(message *relaysyncv2.Event, cause error) *relaysyncv2.EventReceipt {
	var id EventID
	if message != nil && len(message.Id) == len(id) {
		copy(id[:], message.Id)
	}
	return eventReceipt(WriteResult{
		ID:      id,
		Status:  WriteRejectedPermanent,
		Code:    invalidEventReceiptCode,
		Message: cause.Error(),
	})
}

func eventReceipt(result WriteResult) *relaysyncv2.EventReceipt {
	return &relaysyncv2.EventReceipt{
		Id:            append([]byte(nil), result.ID[:]...),
		Status:        writeStatusProto(result.Status),
		Code:          result.Code,
		Message:       result.Message,
		PolicyVersion: result.PolicyVersion,
		DecisionId:    result.DecisionID,
	}
}

// ForgetSession removes retry results associated with a completed cycle.
func (t *TransferService) ForgetSession(id SessionID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for key := range t.operations {
		if key.sessionID == id {
			delete(t.operations, key)
		}
	}
}
func writeStatusProto(status WriteStatus) relaysyncv2.WriteStatus {
	switch status {
	case WriteAccepted:
		return relaysyncv2.WriteStatus_WRITE_ACCEPTED
	case WriteAlreadyExists:
		return relaysyncv2.WriteStatus_WRITE_ALREADY_EXISTS
	case WriteRejectedPermanent:
		return relaysyncv2.WriteStatus_WRITE_REJECTED_PERMANENT
	case WriteFailedTemporary:
		return relaysyncv2.WriteStatus_WRITE_FAILED_TEMPORARY
	default:
		return relaysyncv2.WriteStatus_WRITE_STATUS_UNSPECIFIED
	}
}
