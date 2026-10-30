package relaysync

import "testing"

func TestFrameRoundTrip(t *testing.T) {
	frame, err := EncodeFrame([]byte("payload"), 42, true, FrameOptions{MaxPayload: 100})
	if err != nil {
		t.Fatal(err)
	}
	got, compressed, err := DecodeFrame(frame, FrameOptions{MaxPayload: 100})
	if err != nil || string(got) != "payload" || !compressed {
		t.Fatalf("frame round trip: %q %v %v", got, compressed, err)
	}
}

func TestFrameRejectsInvalidHeader(t *testing.T) {
	if _, _, err := DecodeFrame([]byte("bad"), FrameOptions{MaxPayload: 100}); err != ErrInvalidFrame {
		t.Fatalf("got %v", err)
	}
}
