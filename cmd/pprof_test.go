//go:build diagnostics

package cmd

import (
	"net"
	"net/http"
	"testing"

	"go.uber.org/zap"
)

func TestStartDevelopmentPprofAtServesOnlyInDevelopment(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve loopback address: %v", err)
	}
	address := probe.Addr().String()
	if err := probe.Close(); err != nil {
		t.Fatalf("release loopback address: %v", err)
	}

	if server := startDevelopmentPprofAt("production", address, zap.NewNop()); server != nil {
		t.Fatal("production pprof server is not nil")
	}
	available, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("production pprof reserved loopback address: %v", err)
	}
	if err := available.Close(); err != nil {
		t.Fatalf("close availability probe: %v", err)
	}

	server := startDevelopmentPprofAt("development", "127.0.0.1:0", zap.NewNop())
	if server == nil {
		t.Fatal("development pprof server is nil")
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("close development pprof server: %v", err)
		}
	})

	response, err := http.Get("http://" + server.listener.Addr().String() + "/debug/pprof/")
	if err != nil {
		t.Fatalf("get pprof index: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("pprof index status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}
