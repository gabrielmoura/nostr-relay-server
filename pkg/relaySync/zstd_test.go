package relaysync

import (
	"bytes"
	"testing"
)

func TestZstdCodecRoundTripAndThreshold(t *testing.T) {
	codec, err := NewZstdCodec(ZstdOptions{Level: 1, Threshold: 32, MaxOutput: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer codec.Close()
	small := []byte("small")
	encoded, compressed, err := codec.Encode(small)
	if err != nil || compressed || !bytes.Equal(encoded, small) {
		t.Fatalf("small payload: %v %v", compressed, err)
	}
	large := bytes.Repeat([]byte("relay-sync-event "), 256)
	encoded, compressed, err = codec.Encode(large)
	if err != nil || !compressed {
		t.Fatalf("large payload was not compressed: %v %v", compressed, err)
	}
	decoded, err := codec.Decode(encoded)
	if err != nil || !bytes.Equal(decoded, large) {
		t.Fatalf("round trip failed: %v", err)
	}
}

func TestZstdCodecRejectsOutputLimit(t *testing.T) {
	codec, err := NewZstdCodec(ZstdOptions{Threshold: 0, MaxOutput: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer codec.Close()
	encoded := codec.encoder.EncodeAll(bytes.Repeat([]byte("x"), 1024), nil)
	if _, err := codec.Decode(encoded); err != ErrDecompressedLimit {
		t.Fatalf("expected output limit, got %v", err)
	}
}
