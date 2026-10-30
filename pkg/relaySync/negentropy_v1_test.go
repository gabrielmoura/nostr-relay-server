package relaysync

import (
	"errors"
	"testing"
)

func TestNegentropyFrameRoundTrip(t *testing.T) {
	var id EventID
	id[0] = 1
	want := NegentropyMessage{Version: NegentropyVersion, Ranges: []NegentropyRange{{UpperBound: NegentropyBound{Timestamp: 10, Prefix: []byte{1, 2}}, Mode: NegentropyIDs, IDs: []EventID{id}}}}
	encoded, err := EncodeNegentropy(want)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := DecodeNegentropy(encoded)
	if err != nil || len(got.Ranges) != 1 || got.Ranges[0].IDs[0] != id {
		t.Fatalf("round trip failed: %#v %v", got, err)
	}
}

func TestNegentropyFindsSymmetricDifference(t *testing.T) {
	var a, b, c EventID
	a[0], b[0], c[0] = 1, 2, 3
	left, err := NewNegentropy([]EventMetadata{{ID: a, CreatedAt: 1}, {ID: b, CreatedAt: 2}}, NegentropyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewNegentropy([]EventMetadata{{ID: a, CreatedAt: 1}, {ID: c, CreatedAt: 3}}, NegentropyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	first, err := left.Initiate()
	if err != nil {
		t.Fatal(err)
	}
	second, err := right.Respond(first)
	if err != nil {
		t.Fatal(err)
	}
	_, diff, _, err := left.Apply(second)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Have) != 1 || diff.Have[0] != b || len(diff.Need) != 1 || diff.Need[0] != c {
		t.Fatalf("unexpected diff: %#v", diff)
	}
}

func TestNegentropyRejectsMalformedFrame(t *testing.T) {
	if _, err := DecodeNegentropy([]byte{NegentropyVersion, 0x80}); err == nil {
		t.Fatal("expected malformed frame error")
	}
}

func TestNegentropyEnforcesFrameLimit(t *testing.T) {
	var first, second EventID
	first[0] = 1
	second[0] = 2
	engine, err := NewNegentropy([]EventMetadata{{ID: first}, {ID: second}}, NegentropyOptions{FrameSizeLimit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Initiate(); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("Initiate error = %v, want ErrFrameTooLarge", err)
	}
}
