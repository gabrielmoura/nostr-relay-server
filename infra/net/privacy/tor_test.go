package privacy

import (
	"context"
	"strings"
	"testing"

	"github.com/cretz/bine/control"

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

func TestNewTorAddOnionRequestMapsAllRemotePortsToFiberEndpoint(t *testing.T) {
	t.Parallel()

	key := control.GenKey(control.KeyAlgoED25519V3)
	request := newTorAddOnionRequest([]int{80, 443}, "127.0.0.1:9090", key)
	if request.Key != key {
		t.Fatal("Key must be preserved in the ADD_ONION request")
	}
	if len(request.Ports) != 2 {
		t.Fatalf("Ports = %#v, want two mappings", request.Ports)
	}
	for index, port := range request.Ports {
		if want := []string{"80", "443"}[index]; port.Key != want {
			t.Fatalf("virtual port = %q, want %q", port.Key, want)
		}
		if port.Val != "127.0.0.1:9090" {
			t.Fatalf("mapping target = %q, want Fiber endpoint", port.Val)
		}
	}
}
