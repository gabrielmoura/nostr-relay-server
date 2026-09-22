package cmd

import (
	"net"
	"testing"
)

func TestPrepareFiberListener(t *testing.T) {
	t.Parallel()

	listener, err := prepareFiberListener("127.0.0.1:0")
	if err != nil {
		t.Fatalf("prepare listener: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	if listener.Addr() == nil {
		t.Fatal("listener address is nil")
	}
}

func TestPrepareFiberListenerReturnsBindError(t *testing.T) {
	t.Parallel()

	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy listener: %v", err)
	}
	t.Cleanup(func() { _ = occupied.Close() })

	listener, err := prepareFiberListener(occupied.Addr().String())
	if err == nil {
		_ = listener.Close()
		t.Fatal("prepare listener error = nil, want bind error")
	}
	if listener != nil {
		t.Fatal("listener is not nil after bind error")
	}
}
