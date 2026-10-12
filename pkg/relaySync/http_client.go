package relaysync

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"google.golang.org/protobuf/proto"
)

var (
	ErrNilHTTPClient           = errors.New("relaysync: HTTP client is required")
	ErrInvalidHTTPBaseURL      = errors.New("relaysync: invalid HTTP base URL")
	ErrUnexpectedContentType   = errors.New("relaysync: unexpected HTTP content type")
	ErrUnexpectedContentCoding = errors.New("relaysync: unexpected HTTP content encoding")
)

// HTTPClientOptions configures a reusable RelaySync HTTP client. The caller
// owns ZstdCodec and must keep it open while the client is in use.
type HTTPClientOptions struct {
	BaseURL          string
	Client           *http.Client
	Zstd             *ZstdCodec
	MaxResponseBytes int
}

// HTTPClient consumes RelaySync's protobuf HTTP endpoints. It is independent
// of a router, server implementation, and authentication scheme.
type HTTPClient struct {
	baseURL          *url.URL
	client           *http.Client
	zstd             *ZstdCodec
	maxResponseBytes int
}

// NewHTTPClient creates a RelaySync HTTP client. A supplied http.Client is
// used as-is, allowing callers to control transport, timeouts, and tracing.
func NewHTTPClient(options HTTPClientOptions) (*HTTPClient, error) {
	if options.Client == nil {
		return nil, ErrNilHTTPClient
	}
	baseURL, err := url.Parse(options.BaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("%w: %q", ErrInvalidHTTPBaseURL, options.BaseURL)
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, fmt.Errorf("%w: unsupported scheme %q", ErrInvalidHTTPBaseURL, baseURL.Scheme)
	}
	if options.MaxResponseBytes == 0 {
		options.MaxResponseBytes = DefaultLimits().MaxBatchBytes
	}
	if options.MaxResponseBytes < 0 {
		return nil, fmt.Errorf("%w: max response bytes must be positive", ErrInvalidLimits)
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/")
	return &HTTPClient{
		baseURL:          baseURL,
		client:           options.Client,
		zstd:             options.Zstd,
		maxResponseBytes: options.MaxResponseBytes,
	}, nil
}

// Capabilities retrieves the peer's RelaySync v2 capabilities.
func (c *HTTPClient) Capabilities(ctx context.Context) (*relaysyncv2.Capabilities, error) {
	response := &relaysyncv2.Capabilities{}
	if err := c.doProto(ctx, http.MethodGet, CapabilitiesPath, nil, response); err != nil {
		return nil, err
	}
	return response, nil
}

// OpenSession opens a RelaySync session with the peer.
func (c *HTTPClient) OpenSession(ctx context.Context, request *relaysyncv2.OpenSessionRequest) (*relaysyncv2.OpenSessionResponse, error) {
	if request == nil {
		return nil, ErrInvalidEvent
	}
	response := &relaysyncv2.OpenSessionResponse{}
	if err := c.doProto(ctx, http.MethodPost, sessionsPath, request, response); err != nil {
		return nil, err
	}
	return response, nil
}

// CompleteSession releases all server-side state for a successfully finished
// finite synchronization cycle. The token is sent only in the Authorization
// header and is never included in the URL.
func (c *HTTPClient) CompleteSession(ctx context.Context, id SessionID, token []byte) error {
	request := &relaysyncv2.CompleteSessionRequest{SessionId: append([]byte(nil), id[:]...)}
	response := &relaysyncv2.CompleteSessionResponse{}
	path := sessionsPath + "/" + base64.RawURLEncoding.EncodeToString(id[:]) + "/complete"
	headers := make(http.Header)
	headers.Set("Authorization", "Bearer "+base64.RawURLEncoding.EncodeToString(token))
	if err := c.doProtoWithHeaders(ctx, http.MethodPost, path, request, response, headers); err != nil {
		return err
	}
	if !response.GetComplete() {
		return ErrConflict
	}
	return nil
}

// ManifestNext requests the next bounded manifest page for a session.
func (c *HTTPClient) ManifestNext(ctx context.Context, id SessionID, token []byte, request *relaysyncv2.ManifestRequest) (*relaysyncv2.Manifest, error) {
	if request == nil {
		return nil, ErrInvalidEvent
	}
	response := &relaysyncv2.Manifest{}
	if err := c.doSessionProto(ctx, http.MethodPost, id, "/manifest:next", token, request, response); err != nil {
		return nil, err
	}
	return response, nil
}

// PullEvents retrieves event payloads from the immutable session snapshot.
func (c *HTTPClient) PullEvents(ctx context.Context, id SessionID, token []byte, request *relaysyncv2.PullEventsRequest) (*relaysyncv2.PullEventsResponse, error) {
	if request == nil {
		return nil, ErrInvalidEvent
	}
	response := &relaysyncv2.PullEventsResponse{}
	if err := c.doSessionProto(ctx, http.MethodPost, id, "/events:pull", token, request, response); err != nil {
		return nil, err
	}
	return response, nil
}

// PushEvents sends a retry-safe batch to the peer's EventWriter.
func (c *HTTPClient) PushEvents(ctx context.Context, id SessionID, token []byte, request *relaysyncv2.PushEventsRequest) (*relaysyncv2.PushEventsResponse, error) {
	if request == nil {
		return nil, ErrInvalidEvent
	}
	response := &relaysyncv2.PushEventsResponse{}
	if err := c.doSessionProto(ctx, http.MethodPost, id, "/events:push", token, request, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *HTTPClient) doSessionProto(ctx context.Context, method string, id SessionID, suffix string, token []byte, request, response proto.Message) error {
	path := sessionsPath + "/" + base64.RawURLEncoding.EncodeToString(id[:]) + suffix
	headers := make(http.Header)
	headers.Set("Authorization", "Bearer "+base64.RawURLEncoding.EncodeToString(token))
	return c.doProtoWithHeaders(ctx, method, path, request, response, headers)
}

func (c *HTTPClient) doProto(ctx context.Context, method, path string, request, response proto.Message) error {
	return c.doProtoWithHeaders(ctx, method, path, request, response, nil)
}

func (c *HTTPClient) doProtoWithHeaders(ctx context.Context, method, path string, request, response proto.Message, headers http.Header) error {
	if c == nil || c.client == nil || c.baseURL == nil {
		return ErrNilHTTPClient
	}
	endpoint := c.baseURL.JoinPath(path)
	var body io.Reader
	if request != nil {
		encoded, err := proto.Marshal(request)
		if err != nil {
			return fmt.Errorf("marshal protobuf request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return fmt.Errorf("create HTTP request: %w", err)
	}
	httpRequest.Header.Set("Accept", protobufMediaType)
	for key, values := range headers {
		for _, value := range values {
			httpRequest.Header.Add(key, value)
		}
	}
	if request != nil {
		httpRequest.Header.Set("Content-Type", protobufMediaType)
	}
	if c.zstd != nil {
		httpRequest.Header.Set("Accept-Encoding", "zstd")
	}
	httpResponse, err := c.client.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("perform HTTP request: %w", err)
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return &HTTPResponseError{StatusCode: httpResponse.StatusCode, Status: httpResponse.Status}
	}
	if !isProtobufContentType(httpResponse.Header.Get("Content-Type")) {
		return fmt.Errorf("%w: %q", ErrUnexpectedContentType, httpResponse.Header.Get("Content-Type"))
	}
	payload, err := c.readResponse(httpResponse)
	if err != nil {
		return err
	}
	if err := proto.Unmarshal(payload, response); err != nil {
		return fmt.Errorf("unmarshal protobuf response: %w", err)
	}
	return nil
}

func (c *HTTPClient) readResponse(response *http.Response) ([]byte, error) {
	payload, err := io.ReadAll(io.LimitReader(response.Body, int64(c.maxResponseBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("read HTTP response: %w", err)
	}
	if len(payload) > c.maxResponseBytes {
		return nil, ErrDecompressedLimit
	}
	contentEncoding := strings.TrimSpace(response.Header.Get("Content-Encoding"))
	if contentEncoding == "" || strings.EqualFold(contentEncoding, "identity") {
		return payload, nil
	}
	if !strings.EqualFold(contentEncoding, "zstd") || c.zstd == nil {
		return nil, fmt.Errorf("%w: %q", ErrUnexpectedContentCoding, contentEncoding)
	}
	decoded, err := c.zstd.Decode(payload)
	if err != nil {
		return nil, fmt.Errorf("decode zstd HTTP response: %w", err)
	}
	if len(decoded) > c.maxResponseBytes {
		return nil, ErrDecompressedLimit
	}
	return decoded, nil
}

func isProtobufContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && mediaType == protobufMediaType
}

// HTTPResponseError reports a non-successful HTTP response without attempting
// to interpret the peer's error body.
type HTTPResponseError struct {
	StatusCode int
	Status     string
}

func (e *HTTPResponseError) Error() string {
	return fmt.Sprintf("relaysync: HTTP request failed: %s", e.Status)
}
