package relaysync

import (
	"bytes"
	"context"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

// ReconcileService handles one bounded Negentropy round. The local vector is
// rebuilt from the immutable snapshot for each request, so no per-peer
// Negentropy state survives a finite cycle.
type ReconcileService struct {
	sessions *SessionService
	limits   Limits
}

func NewReconcileService(sessions *SessionService, limits Limits) (*ReconcileService, error) {
	if sessions == nil {
		return nil, ErrNotFound
	}
	limits = limits.WithDefaults()
	if err := limits.Validate(); err != nil {
		return nil, err
	}
	return &ReconcileService{sessions: sessions, limits: limits}, nil
}

// Reconcile responds to a peer's Negentropy frame for one snapshot partition.
// A partition that exceeds the bounded metadata budget must be subdivided by
// the manifest algorithm before reconciliation.
func (r *ReconcileService) Reconcile(ctx context.Context, principal Principal, id SessionID, request *relaysyncv2.ReconcileRequest) (*relaysyncv2.ReconcileResponse, error) {
	if request == nil || len(request.PartitionId) != 32 || len(request.NegentropyPayload) == 0 {
		return nil, ErrInvalidEvent
	}
	if request.Round == 0 || request.Round > uint32(r.limits.MaxConcurrentRounds) {
		return nil, ErrConflict
	}
	if len(request.NegentropyPayload) > r.limits.MaxFrameBytes {
		return nil, ErrFrameTooLarge
	}
	if _, err := r.sessions.RequireAction(ctx, principal, id, ActionPull); err != nil {
		return nil, err
	}
	snapshot, ok := r.sessions.Snapshot(id)
	if !ok {
		return nil, ErrSnapshotExpired
	}
	partition, ok := findPartition(snapshot.Descriptor().Partitions, request.PartitionId)
	if !ok {
		return nil, ErrNotFound
	}
	metadata, err := readPartitionMetadata(ctx, snapshot, partition, r.limits.MaxBatchBytes/(len(EventID{})+8))
	if err != nil {
		return nil, err
	}
	engine, err := NewNegentropy(metadata, NegentropyOptions{FrameSizeLimit: r.limits.MaxFrameBytes})
	if err != nil {
		return nil, err
	}
	responseFrame, err := engine.Respond(request.NegentropyPayload)
	if err != nil {
		return nil, err
	}
	if len(responseFrame) > r.limits.MaxFrameBytes {
		return nil, ErrFrameTooLarge
	}
	return &relaysyncv2.ReconcileResponse{
		PartitionId:       append([]byte(nil), request.PartitionId...),
		NegentropyPayload: responseFrame,
		Round:             request.Round,
		Complete:          len(responseFrame) == 1,
	}, nil
}

func findPartition(partitions []PartitionKey, id []byte) (PartitionKey, bool) {
	for _, partition := range partitions {
		identity := PartitionIdentity(partition)
		if bytes.Equal(identity[:], id) {
			return partition, true
		}
	}
	return PartitionKey{}, false
}

func readPartitionMetadata(ctx context.Context, snapshot Snapshot, partition PartitionKey, maxItems int) ([]EventMetadata, error) {
	if maxItems <= 0 {
		return nil, ErrInvalidLimits
	}
	iterator, err := snapshot.OpenMetadata(ctx, partition, nil)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()
	metadata := make([]EventMetadata, 0, minInt(maxItems, 1024))
	for {
		item, ok, err := iterator.Next(ctx)
		if err != nil {
			return nil, err
		}
		if !ok {
			return metadata, nil
		}
		if len(metadata) == maxItems {
			return nil, ErrInvalidLimits
		}
		metadata = append(metadata, item)
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
