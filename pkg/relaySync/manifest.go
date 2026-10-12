package relaysync

import (
	"context"
	"encoding/binary"
	"errors"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

var ErrInvalidContinuation = errors.New("relaysync: invalid continuation")

type ManifestService struct {
	sessions *SessionService
	pageSize int
}

func NewManifestService(sessions *SessionService, pageSize int) (*ManifestService, error) {
	if sessions == nil {
		return nil, ErrNotFound
	}
	if pageSize <= 0 {
		pageSize = 64
	}
	return &ManifestService{sessions: sessions, pageSize: pageSize}, nil
}

func (m *ManifestService) Next(ctx context.Context, principal Principal, id SessionID, continuation []byte, limit uint32) (*relaysyncv2.Manifest, error) {
	record, err := m.sessions.RequireAction(ctx, principal, id, ActionPull)
	if err != nil {
		return nil, err
	}
	snapshot, ok := m.sessions.Snapshot(id)
	if !ok {
		return nil, ErrSnapshotExpired
	}
	descriptor := snapshot.Descriptor()
	if string(descriptor.Version) != string(record.SnapshotVersion) {
		return nil, ErrSnapshotExpired
	}
	start := 0
	if len(continuation) > 0 {
		if len(continuation) != 8 {
			return nil, ErrInvalidContinuation
		}
		start = int(binary.BigEndian.Uint64(continuation))
	}
	page := m.pageSize
	if limit > 0 && int(limit) < page {
		page = int(limit)
	}
	// Parallel clients can speculatively request a page after the final page.
	// It is a valid empty terminal page, not a malformed continuation. Keeping
	// this distinction lets callers use bounded parallelism without retrying
	// the entire manifest when the final offset races another worker.
	if start >= len(descriptor.Partitions) {
		return &relaysyncv2.Manifest{
			SnapshotVersion: append([]byte(nil), descriptor.Version...),
			Watermark:       &relaysyncv2.Cursor{Epoch: descriptor.Watermark.Epoch, Sequence: descriptor.Watermark.Sequence},
			Complete:        true,
		}, nil
	}
	end := start + page
	if end > len(descriptor.Partitions) {
		end = len(descriptor.Partitions)
	}
	response := &relaysyncv2.Manifest{SnapshotVersion: append([]byte(nil), descriptor.Version...), Watermark: &relaysyncv2.Cursor{Epoch: descriptor.Watermark.Epoch, Sequence: descriptor.Watermark.Sequence}, Complete: end == len(descriptor.Partitions)}
	for _, partition := range descriptor.Partitions[start:end] {
		summary, err := manifestPartition(ctx, snapshot, partition)
		if err != nil {
			return nil, err
		}
		response.Partitions = append(response.Partitions, summary)
	}
	if end < len(descriptor.Partitions) {
		response.Continuation = make([]byte, 8)
		binary.BigEndian.PutUint64(response.Continuation, uint64(end))
	}
	return response, nil
}

func manifestPartition(ctx context.Context, snapshot Snapshot, partition PartitionKey) (*relaysyncv2.PartitionSummary, error) {
	var count uint64
	if counter, ok := snapshot.(partitionCounter); ok {
		var err error
		count, err = counter.PartitionCount(ctx, partition)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		count, err = countPartitionMetadata(ctx, snapshot, partition)
		if err != nil {
			return nil, err
		}
	}
	iterator, err := snapshot.OpenMetadata(ctx, partition, nil)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()
	digest, err := digestMetadataIterator(ctx, iterator, count)
	if err != nil {
		return nil, err
	}
	id := PartitionIdentity(partition)
	return &relaysyncv2.PartitionSummary{Partition: &relaysyncv2.Partition{Id: id[:], Bucket: partition.Bucket, PrefixBits: uint32(partition.PrefixBits), Prefix: append([]byte(nil), partition.Prefix[:]...), EventCount: count, AlgorithmVersion: partition.AlgorithmVersion}, Digest: digest[:]}, nil
}

// partitionCounter is an optional snapshot capability. SQL-backed adapters
// can obtain a grouped COUNT(*) while discovering partitions, so manifest
// construction needs only one ordered metadata scan for the canonical digest.
// The count never changes a PartitionKey: local cardinalities may differ.
type partitionCounter interface {
	PartitionCount(context.Context, PartitionKey) (uint64, error)
}

func countPartitionMetadata(ctx context.Context, snapshot Snapshot, partition PartitionKey) (uint64, error) {
	iterator, err := snapshot.OpenMetadata(ctx, partition, nil)
	if err != nil {
		return 0, err
	}
	defer iterator.Close()
	var count uint64
	for {
		_, ok, err := iterator.Next(ctx)
		if err != nil {
			return 0, err
		}
		if !ok {
			break
		}
		if count == ^uint64(0) {
			return 0, ErrInvalidLimits
		}
		count++
	}
	return count, nil
}
