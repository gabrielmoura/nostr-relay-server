package ws

import (
	"testing"
	"time"
)

func TestSendToWriterReturnsWhenWriterStopped(t *testing.T) {
	lifecycle := newConnectionLifecycle()
	lifecycle.stop()

	result := make(chan bool, 1)
	go func() {
		result <- sendToWriter(make(chan any), any("notice"), lifecycle.done)
	}()

	select {
	case sent := <-result:
		if sent {
			t.Fatal("sendToWriter = true, want false after writer stop")
		}
	case <-time.After(time.Second):
		t.Fatal("sendToWriter blocked after writer stop")
	}
}
