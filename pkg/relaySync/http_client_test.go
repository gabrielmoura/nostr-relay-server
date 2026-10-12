package relaysync

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"google.golang.org/protobuf/proto"
)

func TestHTTPClientCapabilities(t *testing.T) {
	want := &relaysyncv2.Capabilities{ProtocolVersions: []uint32{2}, Implementation: "peer"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != CapabilitiesPath {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != protobufMediaType {
			t.Fatalf("Accept = %q, want %q", got, protobufMediaType)
		}
		writeTestProto(t, w, want)
	}))
	defer server.Close()
	client := newTestHTTPClient(t, server.URL, nil)

	got, err := client.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("Capabilities = %v, want %v", got, want)
	}
}

func TestHTTPClientOpenSession(t *testing.T) {
	want := &relaysyncv2.OpenSessionResponse{SessionId: []byte("session"), ProtocolVersion: 2}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != sessionsPath {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != protobufMediaType {
			t.Fatalf("Content-Type = %q, want %q", got, protobufMediaType)
		}
		var request relaysyncv2.OpenSessionRequest
		if err := proto.Unmarshal(mustReadAll(t, r), &request); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if request.GetProtocolVersion() != 2 {
			t.Fatalf("protocol version = %d, want 2", request.GetProtocolVersion())
		}
		writeTestProto(t, w, want)
	}))
	defer server.Close()
	client := newTestHTTPClient(t, server.URL, nil)

	got, err := client.OpenSession(context.Background(), &relaysyncv2.OpenSessionRequest{ProtocolVersion: 2})
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("OpenSession = %v, want %v", got, want)
	}
}

func TestHTTPClientRejectsUnexpectedContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	client := newTestHTTPClient(t, server.URL, nil)

	_, err := client.Capabilities(context.Background())
	if !errors.Is(err, ErrUnexpectedContentType) {
		t.Fatalf("Capabilities error = %v, want ErrUnexpectedContentType", err)
	}
}

func TestHTTPClientDecodesZstdResponse(t *testing.T) {
	codec, err := NewZstdCodec(ZstdOptions{Threshold: 0, MaxOutput: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer codec.Close()
	want := &relaysyncv2.Capabilities{Implementation: "compressed"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept-Encoding"); got != "zstd" {
			t.Fatalf("Accept-Encoding = %q, want zstd", got)
		}
		encoded, err := proto.Marshal(want)
		if err != nil {
			t.Fatal(err)
		}
		compressed := codec.encoder.EncodeAll(encoded, nil)
		w.Header().Set("Content-Type", protobufMediaType)
		w.Header().Set("Content-Encoding", "zstd")
		_, _ = w.Write(compressed)
	}))
	defer server.Close()
	client := newTestHTTPClient(t, server.URL, codec)

	got, err := client.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("Capabilities = %v, want %v", got, want)
	}
}

func TestHTTPClientPropagatesContextCancellation(t *testing.T) {
	client, err := NewHTTPClient(HTTPClientOptions{
		BaseURL: "https://peer.example",
		Client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			<-request.Context().Done()
			return nil, request.Context().Err()
		})},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.Capabilities(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Capabilities error = %v, want context.Canceled", err)
	}
}

func newTestHTTPClient(t *testing.T, baseURL string, codec *ZstdCodec) *HTTPClient {
	t.Helper()
	client, err := NewHTTPClient(HTTPClientOptions{BaseURL: baseURL, Client: http.DefaultClient, Zstd: codec})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	return client
}

func writeTestProto(t *testing.T, writer http.ResponseWriter, message proto.Message) {
	t.Helper()
	encoded, err := proto.Marshal(message)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	writer.Header().Set("Content-Type", protobufMediaType)
	_, _ = writer.Write(encoded)
}

func mustReadAll(t *testing.T, request *http.Request) []byte {
	t.Helper()
	defer request.Body.Close()
	encoded, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("read request: %v", err)
	}
	return encoded
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
