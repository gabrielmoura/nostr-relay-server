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

func TestPrepareFiberListenersClosesInternalListenerWhenExternalBindFails(t *testing.T) {
	t.Parallel()

	internalProbe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve internal address: %v", err)
	}
	internalAddress := internalProbe.Addr().String()
	if err := internalProbe.Close(); err != nil {
		t.Fatalf("release internal address: %v", err)
	}

	externalProbe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy external address: %v", err)
	}
	t.Cleanup(func() { _ = externalProbe.Close() })

	internal, external, err := prepareFiberListeners(internalAddress, externalProbe.Addr().String())
	if err == nil {
		if internal != nil {
			_ = internal.Close()
		}
		if external != nil {
			_ = external.Close()
		}
		t.Fatal("prepareFiberListeners error = nil, want external bind error")
	}
	if internal != nil || external != nil {
		t.Fatal("failed listener preparation must not return open listeners")
	}

	rebound, err := net.Listen("tcp", internalAddress)
	if err != nil {
		t.Fatalf("internal listener was not closed after external bind failure: %v", err)
	}
	t.Cleanup(func() { _ = rebound.Close() })
}
