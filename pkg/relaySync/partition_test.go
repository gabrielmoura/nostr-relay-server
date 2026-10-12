package relaysync

import (
	"context"
	"testing"
)

func TestPartitionForUsesFloorAndMaskedPrefix(t *testing.T) {
	var id EventID
	id[0] = 0xab
	id[1] = 0xcd
	key := PartitionFor(EventMetadata{ID: id, CreatedAt: -1}, 12)
	if key.Bucket != -1 || key.Prefix[0] != 0xab || key.Prefix[1] != 0xc0 {
		t.Fatalf("unexpected partition key: %+v", key)
	}
}

func TestDigestMetadataRequiresCanonicalOrder(t *testing.T) {
	items := []EventMetadata{{CreatedAt: 1}, {CreatedAt: 2}}
	first, err := DigestMetadata(items)
	if err != nil {
		t.Fatalf("DigestMetadata: %v", err)
	}
	second, err := DigestMetadata(append([]EventMetadata(nil), items...))
	if err != nil || first != second {
		t.Fatalf("digest is not deterministic: %x %x %v", first, second, err)
	}
	if _, err := DigestMetadata([]EventMetadata{items[1], items[0]}); err != ErrUnsortedMetadata {
		t.Fatalf("expected unsorted error, got %v", err)
	}
}

func TestDigestMetadataIteratorMatchesSliceDigest(t *testing.T) {
	var firstID, secondID EventID
	secondID[0] = 1
	items := []EventMetadata{{CreatedAt: 1, ID: firstID}, {CreatedAt: 2, ID: firstID}, {CreatedAt: 2, ID: secondID}}
	want, err := DigestMetadata(items)
	if err != nil {
		t.Fatal(err)
	}
	got, err := digestMetadataIterator(context.Background(), &memoryMetadataIterator{items: items}, uint64(len(items)))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("stream digest = %x, slice digest = %x", got, want)
	}
}

func TestVectorOwnsSortedMetadata(t *testing.T) {
	items := []EventMetadata{{CreatedAt: 1}, {CreatedAt: 2}}
	vector, err := NewVector(items)
	if err != nil {
		t.Fatalf("NewVector: %v", err)
	}
	items[0].CreatedAt = 99
	got, ok := vector.At(0)
	if !ok || got.CreatedAt != 1 {
		t.Fatalf("vector input was not copied: %+v", got)
	}
	if _, err := vector.Range(-1, 1); err == nil {
		t.Fatal("expected invalid range error")
	}
}
