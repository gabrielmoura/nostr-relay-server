package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"github.com/nbd-wtf/go-nostr"
)

func TestRelaySyncHTTPURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "websocket", input: "ws://relay.example/relay", want: "http://relay.example/relay"},
		{name: "secure websocket", input: "wss://relay.example", want: "https://relay.example"},
		{name: "http", input: "http://relay.example", want: "http://relay.example"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := relaySyncHTTPURL(tt.input)
			if err != nil || got != tt.want {
				t.Fatalf("relaySyncHTTPURL(%q) = %q, %v; want %q", tt.input, got, err, tt.want)
			}
		})
	}
}

func TestRelaySyncCapabilityProbeFallsBackToLegacy(t *testing.T) {
	adapter, err := core.NewHTTPAdapter(&relaysyncv2.Capabilities{ProtocolVersions: []uint32{1}})
	if err != nil {
		t.Fatal(err)
	}
	mux := httpTestMux(t, adapter)
	server := httptest.NewServer(mux)
	defer server.Close()

	result, available, err := runRelaySyncIfAvailable(context.Background(), &ConfSync{Remote: server.URL, Direction: DirectionDown})
	if err != nil {
		t.Fatal(err)
	}
	if available || result != nil {
		t.Fatalf("legacy capability must select NIP-77 fallback: available=%v result=%v", available, result)
	}
}

func TestRelaySyncCapabilityProbeSelectsV2(t *testing.T) {
	adapter, err := core.NewHTTPAdapter(&relaysyncv2.Capabilities{ProtocolVersions: []uint32{2}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpTestMux(t, adapter))
	defer server.Close()

	_, available, err := runRelaySyncIfAvailable(context.Background(), &ConfSync{Remote: server.URL, Direction: DirectionDown, LocalFilter: []nostr.Filter{{}}})
	if !available {
		t.Fatalf("RelaySync v2 peer was not selected, err=%v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "database is not initialized") {
		t.Fatalf("v2 selection should enter RelaySync flow, got err=%v", err)
	}
}

func httpTestMux(t *testing.T, adapter *core.HTTPAdapter) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	adapter.Register(mux)
	return mux
}
