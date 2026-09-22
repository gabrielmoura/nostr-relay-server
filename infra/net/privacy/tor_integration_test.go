//go:build integration

package privacy

import (
	"context"
	"net"
	"os/exec"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"go.uber.org/zap"
)

func TestTorServicePersistentDataDirRestart(t *testing.T) {
	if _, err := exec.LookPath("tor"); err != nil {
		t.Skip("tor binary is not available")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	port := listener.Addr().(*net.TCPAddr).Port
	cfg := config.TorConfig{Mode: "native", DataDir: t.TempDir(), RemotePorts: []int{80}}
	for range 2 {
		svc := newTorService(cfg, zap.NewNop(), NewKeyStore(t.TempDir()))
		if err := svc.Start(context.Background(), port); err != nil {
			t.Fatalf("start Tor service: %v", err)
		}
		if err := svc.Close(); err != nil {
			t.Fatalf("close Tor service: %v", err)
		}
	}
}
