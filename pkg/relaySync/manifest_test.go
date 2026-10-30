package relaysync

import (
	"context"
	"testing"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

func TestManifestServicePaginatesSnapshot(t *testing.T) {
	storage := NewMemoryStorage(nil, DefaultLimits())
	event := testValidEvent(t)
	if _, err := storage.WriteEvents(context.Background(), EventBatch{Events: []Event{event}}); err != nil {
		t.Fatal(err)
	}
	authorizer := NewMemoryPeerAuthorizer()
	scope := Scope{Name: "public", Version: 1}
	authorizer.Grant(Principal{ID: "peer"}, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession, ActionPull}})
	sessions, err := NewSessionService(SessionServiceOptions{Store: NewMemorySessionStore(nil), Snapshots: storage, Authorizer: authorizer})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := sessions.Open(context.Background(), Principal{ID: "peer"}, &relaysyncv2.OpenSessionRequest{ProtocolVersion: 2, Scope: &relaysyncv2.Scope{Name: "public", Version: 1}})
	if err != nil {
		t.Fatal(err)
	}
	var id SessionID
	copy(id[:], opened.SessionId)
	manifests, err := NewManifestService(sessions, 1)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := manifests.Next(context.Background(), Principal{ID: "peer"}, id, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Partitions) != 1 || !manifest.Complete {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
}

func TestManifestServiceAcceptsSpeculativeTerminalContinuation(t *testing.T) {
	storage := NewMemoryStorage(nil, DefaultLimits())
	authorizer := NewMemoryPeerAuthorizer()
	scope := Scope{Name: "public", Version: 1}
	principal := Principal{ID: "peer"}
	authorizer.Grant(principal, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession, ActionPull}})
	sessions, err := NewSessionService(SessionServiceOptions{Store: NewMemorySessionStore(nil), Snapshots: storage, Authorizer: authorizer})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := sessions.Open(context.Background(), principal, &relaysyncv2.OpenSessionRequest{ProtocolVersion: 2, Scope: &relaysyncv2.Scope{Name: scope.Name, Version: scope.Version}})
	if err != nil {
		t.Fatal(err)
	}
	var id SessionID
	copy(id[:], opened.SessionId)
	manifests, err := NewManifestService(sessions, 1)
	if err != nil {
		t.Fatal(err)
	}
	continuation := make([]byte, 8)
	continuation[7] = 1 // an offset beyond the empty descriptor
	manifest, err := manifests.Next(context.Background(), principal, id, continuation, 1)
	if err != nil {
		t.Fatalf("speculative terminal continuation: %v", err)
	}
	if !manifest.GetComplete() || len(manifest.GetPartitions()) != 0 {
		t.Fatalf("unexpected terminal manifest: %+v", manifest)
	}
}

func TestManifestPartitionUsesOptionalSQLCount(t *testing.T) {
	partition := PartitionKey{AlgorithmVersion: 1, PrefixBits: 8}
	partition.Prefix[0] = 0x42
	snapshot := &countedManifestSnapshot{
		descriptor: SnapshotDescriptor{Partitions: []PartitionKey{partition}},
		metadata:   []EventMetadata{{CreatedAt: 1}},
		count:      1,
	}
	summary, err := manifestPartition(context.Background(), snapshot, partition)
	if err != nil {
		t.Fatal(err)
	}
	if summary.GetPartition().GetEventCount() != 1 || snapshot.metadataOpens != 1 {
		t.Fatalf("count=%d metadata opens=%d, want SQL count and one digest scan", summary.GetPartition().GetEventCount(), snapshot.metadataOpens)
	}
}

type countedManifestSnapshot struct {
	descriptor    SnapshotDescriptor
	metadata      []EventMetadata
	count         uint64
	metadataOpens int
}

func (s *countedManifestSnapshot) Descriptor() SnapshotDescriptor { return s.descriptor }
func (s *countedManifestSnapshot) Close() error                   { return nil }
func (s *countedManifestSnapshot) GetEvents(context.Context, []EventID) (EventIterator, error) {
	return &memoryEventIterator{}, nil
}
func (s *countedManifestSnapshot) PartitionCount(context.Context, PartitionKey) (uint64, error) {
	return s.count, nil
}
func (s *countedManifestSnapshot) OpenMetadata(context.Context, PartitionKey, *MetadataCursor) (MetadataIterator, error) {
	s.metadataOpens++
	return &memoryMetadataIterator{items: append([]EventMetadata(nil), s.metadata...)}, nil
}
