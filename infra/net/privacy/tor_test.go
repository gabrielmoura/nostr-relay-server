package privacy

import (
	"context"
	"encoding/json"
	"reflect"
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

func TestTorAuthURLsDerivesVirtualPorts(t *testing.T) {
	t.Parallel()

	got := torAuthURLs("publishedonion", []int{80, 8080, 443})
	want := []string{
		"ws://publishedonion.onion",
		"ws://publishedonion.onion:8080",
		"ws://publishedonion.onion:443",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("torAuthURLs() = %#v, want %#v", got, want)
	}
}

func TestTorServiceAuthURLsOnlyReturnsStartedNativeOnion(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.TorConfig
		started bool
		onionID string
		want    []string
	}{
		{
			name: "stopped native service",
			cfg:  config.TorConfig{Mode: "native"},
		},
		{
			name:    "started external service",
			cfg:     config.TorConfig{Mode: "external"},
			started: true,
			onionID: "publishedonion",
		},
		{
			name:    "started native service",
			cfg:     config.TorConfig{Mode: "native", RemotePorts: []int{80, 8080}},
			started: true,
			onionID: "publishedonion",
			want:    []string{"ws://publishedonion.onion", "ws://publishedonion.onion:8080"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &torService{cfg: tt.cfg, started: tt.started, onionID: tt.onionID}
			if got := service.AuthURLs(); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("AuthURLs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestTorServiceSaveNativeManifest(t *testing.T) {
	t.Parallel()

	store := NewKeyStore(t.TempDir())
	service := &torService{
		logger:  zap.NewNop(),
		store:   store,
		onionID: "publishedonion",
	}
	if err := service.saveNativeManifest([]int{80, 8448}, 4869); err != nil {
		t.Fatalf("saveNativeManifest: %v", err)
	}

	raw, err := store.Load("tor.json")
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	var manifest torManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if manifest.SchemaVersion != 1 || manifest.OnionAddress != "publishedonion.onion" || manifest.OnionServiceVersion != 3 {
		t.Fatalf("manifest identity = %#v, want published onion metadata", manifest)
	}
	if manifest.PublishedAt.IsZero() {
		t.Fatal("manifest PublishedAt is zero")
	}
	if !reflect.DeepEqual(manifest.RelayURLs, []string{"ws://publishedonion.onion", "ws://publishedonion.onion:8448"}) {
		t.Fatalf("manifest URLs = %#v", manifest.RelayURLs)
	}
	if !reflect.DeepEqual(manifest.RemotePorts, []int{80, 8448}) || manifest.Target != "127.0.0.1:4869" {
		t.Fatalf("manifest routing = %#v", manifest)
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
