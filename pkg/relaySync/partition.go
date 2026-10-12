package relaysync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sort"
)

const partitionWindowSeconds int64 = 30 * 24 * 60 * 60

var (
	ErrUnsortedMetadata  = errors.New("metadata is not sorted")
	ErrDuplicateMetadata = errors.New("metadata contains duplicate event id")
)

// PartitionFor maps an event deterministically to the configured temporal
// bucket and event-ID prefix. Negative timestamps use mathematical floor.
func PartitionFor(metadata EventMetadata, prefixBits uint8) PartitionKey {
	bucket := metadata.CreatedAt / partitionWindowSeconds
	if metadata.CreatedAt < 0 && metadata.CreatedAt%partitionWindowSeconds != 0 {
		bucket--
	}
	key := PartitionKey{AlgorithmVersion: 1, Bucket: bucket, PrefixBits: prefixBits}
	copy(key.Prefix[:], metadata.ID[:])
	maskPrefix(&key.Prefix, prefixBits)
	return key
}

func maskPrefix(prefix *[32]byte, bits uint8) {
	bytesToKeep := int(bits / 8)
	if bits%8 != 0 {
		prefix[bytesToKeep] &= 0xff << (8 - bits%8)
		bytesToKeep++
	}
	for i := bytesToKeep; i < len(prefix); i++ {
		prefix[i] = 0
	}
}

// DigestMetadata returns a database-independent digest of a sorted metadata
// set. The canonical input is (created_at big-endian, event_id), with a
// domain separator and record count to prevent ambiguous concatenations.
func DigestMetadata(metadata []EventMetadata) ([32]byte, error) {
	var zero [32]byte
	for i := 1; i < len(metadata); i++ {
		cmp := compareMetadata(metadata[i-1], metadata[i])
		if cmp > 0 {
			return zero, ErrUnsortedMetadata
		}
		if cmp == 0 {
			return zero, ErrDuplicateMetadata
		}
	}
	h := sha256.New()
	h.Write([]byte("nostr-relaysync/metadata/v1\x00"))
	var count [8]byte
	binary.BigEndian.PutUint64(count[:], uint64(len(metadata)))
	h.Write(count[:])
	var timestamp [8]byte
	for _, item := range metadata {
		binary.BigEndian.PutUint64(timestamp[:], uint64(item.CreatedAt))
		h.Write(timestamp[:])
		h.Write(item.ID[:])
	}
	copy(zero[:], h.Sum(nil))
	return zero, nil
}

// digestMetadataIterator computes the same canonical digest as
// DigestMetadata without retaining the partition in memory. count must be the
// number of records in the iterator and is obtained by a separate pass.
func digestMetadataIterator(ctx context.Context, iterator MetadataIterator, count uint64) ([32]byte, error) {
	var zero [32]byte
	h := sha256.New()
	if _, err := h.Write([]byte("nostr-relaysync/metadata/v1\x00")); err != nil {
		return zero, err
	}
	var encodedCount [8]byte
	binary.BigEndian.PutUint64(encodedCount[:], count)
	if _, err := h.Write(encodedCount[:]); err != nil {
		return zero, err
	}
	var previous EventMetadata
	var havePrevious bool
	var timestamp [8]byte
	var seen uint64
	for {
		item, ok, err := iterator.Next(ctx)
		if err != nil {
			return zero, err
		}
		if !ok {
			break
		}
		if havePrevious {
			cmp := compareMetadata(previous, item)
			if cmp > 0 {
				return zero, ErrUnsortedMetadata
			}
			if cmp == 0 {
				return zero, ErrDuplicateMetadata
			}
		}
		binary.BigEndian.PutUint64(timestamp[:], uint64(item.CreatedAt))
		if _, err := h.Write(timestamp[:]); err != nil {
			return zero, err
		}
		if _, err := h.Write(item.ID[:]); err != nil {
			return zero, err
		}
		previous = item
		havePrevious = true
		seen++
	}
	if seen != count {
		return zero, ErrConflict
	}
	copy(zero[:], h.Sum(nil))
	return zero, nil
}

func compareMetadata(left, right EventMetadata) int {
	if left.CreatedAt < right.CreatedAt {
		return -1
	}
	if left.CreatedAt > right.CreatedAt {
		return 1
	}
	return bytes.Compare(left.ID[:], right.ID[:])
}

// Vector is an immutable, sorted metadata vector suitable for one bounded
// partition. It owns its input and exposes copies to prevent mutation after
// sealing.
type Vector struct {
	items  []EventMetadata
	sealed bool
}

func NewVector(items []EventMetadata) (*Vector, error) {
	copyItems := append([]EventMetadata(nil), items...)
	if !sort.SliceIsSorted(copyItems, func(i, j int) bool { return compareMetadata(copyItems[i], copyItems[j]) < 0 }) {
		return nil, ErrUnsortedMetadata
	}
	if _, err := DigestMetadata(copyItems); err != nil {
		return nil, err
	}
	return &Vector{items: copyItems, sealed: true}, nil
}

func (v *Vector) Len() int { return len(v.items) }

func (v *Vector) At(index int) (EventMetadata, bool) {
	if v == nil || index < 0 || index >= len(v.items) {
		return EventMetadata{}, false
	}
	return v.items[index], true
}

func (v *Vector) Range(start, end int) ([]EventMetadata, error) {
	if v == nil || start < 0 || end < start || end > len(v.items) {
		return nil, errors.New("invalid vector range")
	}
	return append([]EventMetadata(nil), v.items[start:end]...), nil
}

func (v *Vector) Digest() ([32]byte, error) {
	if v == nil || !v.sealed {
		return [32]byte{}, errors.New("vector is not sealed")
	}
	return DigestMetadata(v.items)
}
