package relaysync

import (
	"context"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

func TestMemoryStorageSnapshotAndIdempotentWriter(t *testing.T) {
	storage := NewMemoryStorage(nil, DefaultLimits())
	event := testValidEvent(t)
	results, err := storage.WriteEvents(context.Background(), EventBatch{OperationID: "op-1", Events: []Event{event}})
	if err != nil || results[0].Status != WriteAccepted {
		t.Fatalf("write: %+v %v", results, err)
	}
	results, err = storage.WriteEvents(context.Background(), EventBatch{OperationID: "op-1", Events: []Event{event}})
	if err != nil || results[0].Status != WriteAlreadyExists {
		t.Fatalf("retry: %+v %v", results, err)
	}
	snapshot, err := storage.OpenSnapshot(context.Background(), Scope{Name: "public", Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	descriptor := snapshot.Descriptor()
	partition := PartitionFor(EventMetadata{ID: event.ID, CreatedAt: event.CreatedAt}, 8)
	iterator, err := snapshot.OpenMetadata(context.Background(), partition, nil)
	if err != nil {
		t.Fatal(err)
	}
	metadata, ok, err := iterator.Next(context.Background())
	if err != nil || !ok || metadata.ID != event.ID {
		t.Fatalf("metadata: %+v %v %v", metadata, ok, err)
	}
	if descriptor.Watermark.Sequence == 0 {
		t.Fatal("snapshot watermark was not advanced")
	}
}

func testValidEvent(t *testing.T) Event {
	t.Helper()
	event := Event{CreatedAt: 1, Kind: 1, Content: "test"}
	privateKey, _ := btcec.PrivKeyFromBytes(bytesRepeat(1, 32))
	copy(event.PublicKey[:], schnorr.SerializePubKey(privateKey.PubKey()))
	computed, err := event.ComputeID(DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	event.ID = computed
	signature, err := schnorr.Sign(privateKey, event.ID[:])
	if err != nil {
		t.Fatal(err)
	}
	copy(event.Signature[:], signature.Serialize())
	return event
}

func bytesRepeat(value byte, count int) []byte {
	result := make([]byte, count)
	for i := range result {
		result[i] = value
	}
	return result
}
