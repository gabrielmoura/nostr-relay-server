package privacy

import (
	"context"
	"strings"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"go.uber.org/zap"
)

func TestTorServiceStartNativeRejectsUnsupportedV2Configuration(t *testing.T) {
	t.Parallel()

	service := newTorService(config.TorConfig{Mode: "native", DataDir: t.TempDir()}, zap.NewNop(), nil).(*torService)
	err := service.startNative(context.Background(), 8080)
	if err == nil {
		t.Fatal("startNative error = nil, want unsupported v2 configuration error")
	}
	if !strings.Contains(err.Error(), "v3") {
		t.Fatalf("startNative error = %q, want v3 guidance", err)
	}
}
