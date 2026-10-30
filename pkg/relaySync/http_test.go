package relaysync

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"google.golang.org/protobuf/proto"
)

func TestHTTPAdapterCapabilities(t *testing.T) {
	capabilities := &relaysyncv2.Capabilities{
		ProtocolVersions: []uint32{2},
		Compressions:     []relaysyncv2.Compression{relaysyncv2.Compression_COMPRESSION_NONE},
		Scopes:           []string{"public"},
		LiveSync:         true,
		Implementation:   "nrserver",
	}
	adapter, err := NewHTTPAdapter(capabilities)
	if err != nil {
		t.Fatalf("NewHTTPAdapter: %v", err)
	}
	capabilities.Implementation = "mutated"
	mux := http.NewServeMux()
	adapter.Register(mux)

	request := httptest.NewRequest(http.MethodGet, CapabilitiesPath, nil)
	request.Header.Set("Accept", protobufMediaType)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != protobufMediaType {
		t.Fatalf("Content-Type = %q, want %q", contentType, protobufMediaType)
	}
	if vary := response.Header().Get("Vary"); vary != "Accept" {
		t.Fatalf("Vary = %q, want %q", vary, "Accept")
	}
	var got relaysyncv2.Capabilities
	if err := proto.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.GetImplementation() != "nrserver" || !got.GetLiveSync() || got.GetProtocolVersions()[0] != 2 {
		t.Fatalf("unexpected capabilities: implementation=%q live=%v versions=%v", got.GetImplementation(), got.GetLiveSync(), got.GetProtocolVersions())
	}
}

func TestHTTPAdapterRejectsUnsupportedAccept(t *testing.T) {
	adapter, err := NewHTTPAdapter(&relaysyncv2.Capabilities{})
	if err != nil {
		t.Fatalf("NewHTTPAdapter: %v", err)
	}
	mux := http.NewServeMux()
	adapter.Register(mux)

	tests := []struct {
		name   string
		accept string
		status int
	}{
		{name: "missing", status: http.StatusOK},
		{name: "exact", accept: protobufMediaType, status: http.StatusOK},
		{name: "application wildcard", accept: "application/*", status: http.StatusOK},
		{name: "any wildcard", accept: "*/*", status: http.StatusOK},
		{name: "weighted supported", accept: "application/json, application/x-protobuf;q=0.5", status: http.StatusOK},
		{name: "unsupported", accept: "application/json", status: http.StatusNotAcceptable},
		{name: "zero quality", accept: "application/x-protobuf;q=0", status: http.StatusNotAcceptable},
		{name: "invalid quality", accept: "application/x-protobuf;q=invalid", status: http.StatusNotAcceptable},
		{name: "malformed", accept: "application/x-protobuf;", status: http.StatusNotAcceptable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, CapabilitiesPath, nil)
			if test.accept != "" {
				request.Header.Set("Accept", test.accept)
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
		})
	}
}

func TestHTTPAdapterRejectsNonGET(t *testing.T) {
	adapter, err := NewHTTPAdapter(&relaysyncv2.Capabilities{})
	if err != nil {
		t.Fatalf("NewHTTPAdapter: %v", err)
	}
	mux := http.NewServeMux()
	adapter.Register(mux)

	request := httptest.NewRequest(http.MethodPost, CapabilitiesPath, nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if allow := response.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", allow, http.MethodGet)
	}
}

func TestNewHTTPAdapterRejectsNilCapabilities(t *testing.T) {
	if _, err := NewHTTPAdapter(nil); err != ErrNilCapabilities {
		t.Fatalf("NewHTTPAdapter(nil) error = %v, want %v", err, ErrNilCapabilities)
	}
}

func TestHTTPAdapterCompletesFiniteSession(t *testing.T) {
	store := NewMemorySessionStore(nil)
	authorizer := NewMemoryPeerAuthorizer()
	scope := Scope{Name: "public", Version: 1}
	principal := Principal{ID: "peer-a"}
	authorizer.Grant(principal, AuthorizedScope{Scope: scope, Actions: []Action{ActionOpenSession}})
	service, err := NewSessionService(SessionServiceOptions{Store: store, Authorizer: authorizer})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := service.Open(context.Background(), principal, &relaysyncv2.OpenSessionRequest{
		ProtocolVersion: 2,
		Scope:           &relaysyncv2.Scope{Name: scope.Name, Version: scope.Version},
	})
	if err != nil {
		t.Fatal(err)
	}
	var id SessionID
	copy(id[:], opened.SessionId)
	mux := http.NewServeMux()
	adapter, err := NewHTTPAdapter(&relaysyncv2.Capabilities{ProtocolVersions: []uint32{2}})
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.RegisterSessionService(mux, service, func(*http.Request) (Principal, bool) {
		return principal, true
	}); err != nil {
		t.Fatal(err)
	}
	body, err := proto.Marshal(&relaysyncv2.CompleteSessionRequest{SessionId: id[:]})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, sessionsPath+"/"+base64.RawURLEncoding.EncodeToString(id[:])+"/complete", bytes.NewReader(body))
	request.Header.Set("Content-Type", protobufMediaType)
	request.Header.Set("Authorization", "Bearer "+base64.RawURLEncoding.EncodeToString(opened.ResumeToken))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var completed relaysyncv2.CompleteSessionResponse
	if err := proto.Unmarshal(response.Body.Bytes(), &completed); err != nil {
		t.Fatal(err)
	}
	if !completed.GetComplete() {
		t.Fatal("complete response was false")
	}
	if _, err := store.GetSession(context.Background(), id); err != ErrNotFound {
		t.Fatalf("session after complete = %v, want ErrNotFound", err)
	}
}
