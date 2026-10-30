package db

import (
	"context"
	"errors"
	"fmt"
	"math"

	relaysync "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	"github.com/nbd-wtf/go-nostr"
)

// RelaySyncEventPolicy is the host's final acceptance hook. It should reuse
// the relay's existing authorization, retention and moderation decisions.
type RelaySyncEventPolicy func(context.Context, relaysync.Event) error

// RelaySyncEventWriter adapts the existing PostgreSQL batch writer to the
// framework-independent EventWriter contract. It does not bypass host policy:
// callers may provide a policy hook, while NIP-01 shape/signature validation
// is always performed here.
type RelaySyncEventWriter struct {
	queries *Queries
	limits  relaysync.Limits
	policy  RelaySyncEventPolicy
}

func NewRelaySyncEventWriter(queries *Queries, limits relaysync.Limits, policy RelaySyncEventPolicy) (*RelaySyncEventWriter, error) {
	if queries == nil {
		return nil, errors.New("relaysync: event writer requires queries")
	}
	limits = limits.WithDefaults()
	if err := limits.Validate(); err != nil {
		return nil, err
	}
	return &RelaySyncEventWriter{queries: queries, limits: limits, policy: policy}, nil
}

func (w *RelaySyncEventWriter) WriteEvents(ctx context.Context, batch relaysync.EventBatch) ([]relaysync.WriteResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if batch.OperationID == "" || len(batch.Events) == 0 || len(batch.Events) > w.limits.MaxEventsPerBatch {
		return nil, relaysync.ErrInvalidLimits
	}
	results := make([]relaysync.WriteResult, len(batch.Events))
	accepted := make([]*nostr.Event, 0, len(batch.Events))
	acceptedIndexes := make(map[string]int, len(batch.Events))
	seen := make(map[string]struct{}, len(batch.Events))
	for i := range batch.Events {
		event := batch.Events[i]
		results[i] = relaysync.WriteResult{ID: event.ID}
		if err := event.Validate(w.limits); err != nil {
			results[i].Status = relaysync.WriteRejectedPermanent
			results[i].Code = "invalid_event"
			results[i].Message = err.Error()
			continue
		}
		if _, exists := seen[event.ID.String()]; exists {
			results[i].Status = relaysync.WriteAlreadyExists
			results[i].Code = "duplicate_batch"
			continue
		}
		seen[event.ID.String()] = struct{}{}
		if w.policy != nil {
			if err := w.policy(ctx, event); err != nil {
				results[i].Status = relaysync.WriteRejectedPermanent
				results[i].Code = "policy_rejected"
				results[i].Message = err.Error()
				continue
			}
		}
		native, err := relaySyncEventToNostr(event)
		if err != nil {
			results[i].Status = relaysync.WriteRejectedPermanent
			results[i].Code = "storage_shape"
			results[i].Message = err.Error()
			continue
		}
		acceptedIndexes[event.ID.String()] = i
		accepted = append(accepted, native)
	}
	if len(accepted) == 0 {
		return results, nil
	}
	inserted, err := w.queries.InsertEventBatch(ctx, accepted)
	if err != nil {
		return nil, fmt.Errorf("relaysync: persist event batch: %w", err)
	}
	for _, id := range inserted.InsertedIDs {
		if i, ok := acceptedIndexes[id]; ok {
			results[i].Status = relaysync.WriteAccepted
			results[i].Code = "accepted"
			results[i].DecisionID = batch.OperationID + ":" + id
		}
	}
	rejected := make(map[string]struct{}, len(inserted.RejectedEventIDs))
	for _, id := range inserted.RejectedEventIDs {
		rejected[id] = struct{}{}
		if i, ok := acceptedIndexes[id]; ok {
			results[i].Status = relaysync.WriteRejectedPermanent
			results[i].Code = "storage_rejected"
		}
	}
	for _, event := range accepted {
		i := acceptedIndexes[event.ID]
		if results[i].Status == relaysync.WriteStatusUnspecified {
			if _, ok := rejected[event.ID]; ok {
				continue
			}
			results[i].Status = relaysync.WriteAlreadyExists
			results[i].Code = "already_exists"
			results[i].DecisionID = batch.OperationID + ":" + event.ID
		}
	}
	for i := range results {
		if results[i].Status == relaysync.WriteStatusUnspecified {
			results[i].Status = relaysync.WriteRejectedPermanent
			results[i].Code = "rejected"
		}
	}
	return results, nil
}

func relaySyncEventToNostr(event relaysync.Event) (*nostr.Event, error) {
	if event.CreatedAt < math.MinInt32 || event.CreatedAt > math.MaxInt32 || uint64(event.Kind) > math.MaxInt32 {
		return nil, errors.New("event integer fields exceed PostgreSQL range")
	}
	tags := make(nostr.Tags, len(event.Tags))
	for i := range event.Tags {
		tags[i] = append(nostr.Tag(nil), event.Tags[i]...)
	}
	return &nostr.Event{ID: event.ID.String(), PubKey: event.PublicKey.String(), CreatedAt: nostr.Timestamp(event.CreatedAt), Kind: int(event.Kind), Tags: tags, Content: event.Content, Sig: event.Signature.String()}, nil
}
