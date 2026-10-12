package relaysync

import (
	"context"
	"crypto/sha256"
	"sync"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"google.golang.org/protobuf/proto"
)

type TransferService struct {
	sessions   *SessionService
	writer     EventWriter
	limits     Limits
	mu         sync.Mutex
	operations map[string]transferResult
}
type transferResult struct {
	hash     [32]byte
	response *relaysyncv2.PushEventsResponse
}

func NewTransferService(sessions *SessionService, writer EventWriter, limits Limits) (*TransferService, error) {
	if sessions == nil || writer == nil {
		return nil, ErrNotFound
	}
	return &TransferService{sessions: sessions, writer: writer, limits: limits.WithDefaults(), operations: make(map[string]transferResult)}, nil
}

func (t *TransferService) Pull(ctx context.Context, principal Principal, id SessionID, request *relaysyncv2.PullEventsRequest) (*relaysyncv2.PullEventsResponse, error) {
	if request == nil {
		return nil, ErrInvalidEvent
	}
	if _, err := t.sessions.Session(ctx, principal, id); err != nil {
		return nil, err
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
	if _, err := t.sessions.Session(ctx, principal, id); err != nil {
		return nil, err
	}
	events := make([]Event, len(request.Events))
	for i, message := range request.Events {
		event, err := EventFromProto(message, t.limits)
		if err != nil {
			return nil, err
		}
		events[i] = event
	}
	hash := sha256.New()
	encoded, _ := proto.Marshal(request)
	hash.Write(encoded)
	var payloadHash [32]byte
	copy(payloadHash[:], hash.Sum(nil))
	t.mu.Lock()
	if cached, ok := t.operations[request.OperationId]; ok {
		t.mu.Unlock()
		if cached.hash != payloadHash {
			return nil, ErrConflict
		}
		return proto.Clone(cached.response).(*relaysyncv2.PushEventsResponse), nil
	}
	t.mu.Unlock()
	results, err := t.writer.WriteEvents(ctx, EventBatch{OperationID: request.OperationId, Events: events})
	if err != nil {
		return nil, err
	}
	response := &relaysyncv2.PushEventsResponse{}
	for _, result := range results {
		response.Receipts = append(response.Receipts, &relaysyncv2.EventReceipt{Id: append([]byte(nil), result.ID[:]...), Status: writeStatusProto(result.Status), Code: result.Code, Message: result.Message, PolicyVersion: result.PolicyVersion, DecisionId: result.DecisionID})
	}
	t.mu.Lock()
	t.operations[request.OperationId] = transferResult{hash: payloadHash, response: response}
	t.mu.Unlock()
	return response, nil
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
