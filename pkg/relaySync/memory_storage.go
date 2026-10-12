package relaysync

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"
)

// MemoryStorage is a bounded, process-local implementation useful for
// conformance and end-to-end tests. It is not a durable relay store.
type MemoryStorage struct {
	mu        sync.RWMutex
	clock     Clock
	limits    Limits
	events    map[EventID]Event
	watermark Cursor
}

func NewMemoryStorage(clock Clock, limits Limits) *MemoryStorage {
	if clock == nil {
		clock = systemClock{}
	}
	return &MemoryStorage{clock: clock, limits: limits.WithDefaults(), events: make(map[EventID]Event)}
}

func (s *MemoryStorage) OpenSnapshot(ctx context.Context, scope Scope) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	copyEvents := make(map[EventID]Event, len(s.events))
	metadata := make([]EventMetadata, 0, len(s.events))
	for id, event := range s.events {
		copyEvents[id] = cloneEvent(event)
		metadata = append(metadata, EventMetadata{ID: id, CreatedAt: event.CreatedAt})
	}
	sortMetadata(metadata)
	partitionSet := make(map[PartitionKey]struct{})
	for _, item := range metadata {
		partitionSet[PartitionFor(item, 8)] = struct{}{}
	}
	partitions := make([]PartitionKey, 0, len(partitionSet))
	for partition := range partitionSet {
		partitions = append(partitions, partition)
	}
	version := sha256.Sum256(append([]byte(scope.Name), byte(scope.Version)))
	return &memorySnapshot{events: copyEvents, metadata: metadata, descriptor: SnapshotDescriptor{Version: version[:], Scope: scope, Watermark: s.watermark, ExpiresAt: s.clock.Now().Add(30 * time.Minute), Partitions: partitions}}, nil
}

func (s *MemoryStorage) WriteEvents(ctx context.Context, batch EventBatch) ([]WriteResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	results := make([]WriteResult, len(batch.Events))
	for i, event := range batch.Events {
		results[i].ID = event.ID
		if err := event.Validate(s.limits); err != nil {
			results[i].Status = WriteRejectedPermanent
			results[i].Code = "INVALID_EVENT"
			continue
		}
		if _, exists := s.events[event.ID]; exists {
			results[i].Status = WriteAlreadyExists
			continue
		}
		s.events[event.ID] = cloneEvent(event)
		s.watermark.Sequence++
		if s.watermark.Epoch == 0 {
			s.watermark.Epoch = 1
		}
		results[i].Status = WriteAccepted
	}
	return results, nil
}

func sortMetadata(items []EventMetadata) {
	for i := 1; i < len(items); i++ {
		current := items[i]
		j := i
		for j > 0 && compareMetadata(items[j-1], current) > 0 {
			items[j] = items[j-1]
			j--
		}
		items[j] = current
	}
}

type memorySnapshot struct {
	mu         sync.Mutex
	events     map[EventID]Event
	metadata   []EventMetadata
	descriptor SnapshotDescriptor
	closed     bool
}

func (s *memorySnapshot) Descriptor() SnapshotDescriptor {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.descriptor
	d.Version = append([]byte(nil), d.Version...)
	d.Partitions = append([]PartitionKey(nil), d.Partitions...)
	return d
}
func (s *memorySnapshot) OpenMetadata(ctx context.Context, partition PartitionKey, after *MetadataCursor) (MetadataIterator, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrSnapshotExpired
	}
	items := make([]EventMetadata, 0)
	for _, item := range s.metadata {
		if PartitionFor(item, partition.PrefixBits) != partition {
			continue
		}
		if after != nil && compareMetadata(item, *after) <= 0 {
			continue
		}
		items = append(items, item)
	}
	return &memoryMetadataIterator{items: items}, nil
}
func (s *memorySnapshot) GetEvents(ctx context.Context, ids []EventID) (EventIterator, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrSnapshotExpired
	}
	items := make([]EventItem, len(ids))
	for i, id := range ids {
		items[i] = EventItem{ID: id}
		if event, ok := s.events[id]; ok {
			copy := cloneEvent(event)
			items[i].Event = &copy
		}
	}
	return &memoryEventIterator{items: items}, nil
}
func (s *memorySnapshot) Close() error { s.mu.Lock(); s.closed = true; s.mu.Unlock(); return nil }

type memoryMetadataIterator struct {
	items  []EventMetadata
	index  int
	closed bool
}

func (i *memoryMetadataIterator) Next(ctx context.Context) (EventMetadata, bool, error) {
	if err := ctx.Err(); err != nil {
		return EventMetadata{}, false, err
	}
	if i.closed || i.index >= len(i.items) {
		return EventMetadata{}, false, nil
	}
	item := i.items[i.index]
	i.index++
	return item, true, nil
}
func (i *memoryMetadataIterator) Close() error { i.closed = true; return nil }

type memoryEventIterator struct {
	items  []EventItem
	index  int
	closed bool
}

func (i *memoryEventIterator) Next(ctx context.Context) (EventItem, bool, error) {
	if err := ctx.Err(); err != nil {
		return EventItem{}, false, err
	}
	if i.closed || i.index >= len(i.items) {
		return EventItem{}, false, nil
	}
	item := i.items[i.index]
	i.index++
	return item, true, nil
}
func (i *memoryEventIterator) Close() error { i.closed = true; return nil }

func cloneEvent(event Event) Event {
	event.Tags = append([]Tag(nil), event.Tags...)
	for i := range event.Tags {
		event.Tags[i] = append(Tag(nil), event.Tags[i]...)
	}
	return event
}
