package relaysync

import (
	"context"
	"errors"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/ingestion"
	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	"github.com/nbd-wtf/go-nostr"
)

type durableIngest func(context.Context, *nostr.Event) (ingestion.Receipt, bool)

type ingestionWriter struct {
	policy db.RelaySyncEventPolicy
	push   durableIngest
}

// NewIngestionWriter applies the relay's policy and writes through ingestion.
func NewIngestionWriter() core.EventWriter {
	return &ingestionWriter{policy: relaySyncWritePolicy, push: ingestion.PushDurable}
}

func (w *ingestionWriter) WriteEvents(ctx context.Context, batch core.EventBatch) ([]core.WriteResult, error) {
	limits := core.DefaultLimits()
	if batch.OperationID == "" || len(batch.Events) == 0 || len(batch.Events) > limits.MaxEventsPerBatch {
		return nil, core.ErrInvalidLimits
	}
	results := make([]core.WriteResult, len(batch.Events))
	seen := make(map[core.EventID]core.WriteResult, len(batch.Events))
	for i, event := range batch.Events {
		result := core.WriteResult{ID: event.ID}
		if err := event.Validate(limits); err != nil {
			result.Status, result.Code = core.WriteRejectedPermanent, "invalid_event"
			results[i] = result
			continue
		}
		if prior, ok := seen[event.ID]; ok {
			if prior.Status == core.WriteAccepted {
				prior.Status, prior.Code = core.WriteAlreadyExists, "duplicate_batch"
			}
			results[i] = prior
			continue
		}
		if err := w.policy(ctx, event); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if errors.Is(err, db.ErrRelaySyncPolicyUnavailable) {
				return nil, err
			}
			result.Status, result.Code = core.WriteRejectedPermanent, "policy_rejected"
			results[i] = result
			continue
		}
		native, err := db.RelaySyncEventToNostr(event)
		if err != nil {
			result.Status, result.Code = core.WriteRejectedPermanent, "storage_shape"
			results[i] = result
			continue
		}
		receipt, accepted := w.push(ctx, native)
		if !accepted && !receipt.Rejected {
			return nil, fmt.Errorf("relaysync: ingestion outcome unavailable: %s", receipt.Reason)
		}
		switch {
		case receipt.Inserted:
			result.Status, result.Code = core.WriteAccepted, "accepted"
		case receipt.Duplicate:
			result.Status, result.Code = core.WriteAlreadyExists, "already_exists"
		case receipt.Rejected:
			result.Status, result.Code = core.WriteRejectedPermanent, "policy_rejected"
		default:
			return nil, errors.New("relaysync: ingestion returned no outcome")
		}
		if receipt.Inserted || receipt.Duplicate {
			result.DecisionID = batch.OperationID + ":" + event.ID.String()
		}
		seen[event.ID] = result
		results[i] = result
	}
	return results, nil
}
