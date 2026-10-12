package nethttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	relaysync "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

func TestNewHandlerRequiresResolver(t *testing.T) {
	adapter, err := relaysync.NewHTTPAdapter(&relaysyncv2.Capabilities{ProtocolVersions: []uint32{2}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewHandler(adapter, Services{}); err != relaysync.ErrUnauthorized {
		t.Fatalf("NewHandler() error = %v, want unauthorized", err)
	}
}

func TestNewHandlerRegistersCapabilities(t *testing.T) {
	adapter, err := relaysync.NewHTTPAdapter(&relaysyncv2.Capabilities{ProtocolVersions: []uint32{2}})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(adapter, Services{Resolve: func(*http.Request) (relaysync.Principal, bool) { return relaysync.Principal{ID: "p"}, true }})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, relaysync.CapabilitiesPath, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("capabilities status = %d", response.Code)
	}
}
