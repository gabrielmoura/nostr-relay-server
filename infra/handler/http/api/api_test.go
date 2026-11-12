package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

func TestAPIRejectsRequestsOverPerIPRateLimit(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{
		Enabled:  true,
		MaxLimit: 1,
		RateLimit: config.IPRequestRateLimitConfig{
			Enabled: true, RequestsPerSec: 1, Burst: 1, MaxClients: 10, IdleTTLSeconds: 60,
		},
	}}
	app := fiber.New()
	NewWithSource(cfg, func(context.Context, nostr.Filter, int) ([]*nostr.Event, error) { return nil, nil }).Register(app.Group("/api/v1"))

	for _, want := range []int{fiber.StatusOK, fiber.StatusTooManyRequests} {
		response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/query?limit=1", nil))
		if err != nil {
			t.Fatalf("app.Test() error = %v", err)
		}
		if response.StatusCode != want {
			t.Fatalf("request status = %d, want %d", response.StatusCode, want)
		}
		response.Body.Close()
	}
}

func TestScanRejectsResultsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := &config.Config{API: config.APIConfig{MaxLimit: 2, MaxScan: 10}}
	handler := NewWithSource(cfg, func(context.Context, nostr.Filter, int) ([]*nostr.Event, error) {
		cancel()
		return []*nostr.Event{{ID: "1", Kind: 1}}, nil
	})
	_, _, err := handler.scan(ctx, nostr.Filter{Limit: 1}, 0, 2, nil, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want cancellation", err)
	}
}

func TestAuthorSingletonRoutes(t *testing.T) {
	pubkey := strings.Repeat("a", 64)
	cfg := &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 10, MaxScan: 20}}
	app := fiber.New()
	NewWithSource(cfg, func(_ context.Context, filter nostr.Filter, offset int) ([]*nostr.Event, error) {
		if len(filter.Authors) != 1 || filter.Authors[0] != pubkey || len(filter.Kinds) != 1 {
			t.Fatalf("unexpected filter: %#v", filter)
		}
		if offset > 0 {
			return nil, nil
		}
		return []*nostr.Event{{ID: testID('1'), PubKey: pubkey, Kind: filter.Kinds[0]}}, nil
	}).Register(app.Group("/api/v1"))
	for _, tc := range []struct {
		path string
		kind int
	}{{"follows", 3}, {"relays", 10002}} {
		response, err := app.Test(httptest.NewRequest("GET", "/api/v1/"+pubkey+"/"+tc.path, nil))
		if err != nil {
			t.Fatal(err)
		}
		var body eventsResponse
		err = json.NewDecoder(response.Body).Decode(&body)
		response.Body.Close()
		if err != nil || response.StatusCode != fiber.StatusOK || body.Count != 1 || body.More || body.Events[0].Kind != tc.kind {
			t.Fatalf("%s: status=%d body=%#v err=%v", tc.path, response.StatusCode, body, err)
		}
		response, err = app.Test(httptest.NewRequest("GET", "/api/v1/"+pubkey+"/"+tc.path+"?offset=1", nil))
		if err != nil {
			t.Fatal(err)
		}
		err = json.NewDecoder(response.Body).Decode(&body)
		response.Body.Close()
		if err != nil || body.Count != 0 || len(body.Events) != 0 {
			t.Fatalf("offset: body=%#v err=%v", body, err)
		}
	}
}

func TestQueryHidesProtectedAndDirectMessagesBeforePagination(t *testing.T) {
	cfg := &config.Config{
		API:      config.APIConfig{Enabled: true, MaxLimit: 2, MaxScan: 20},
		Security: config.SecurityConfig{Limits: config.SecurityLimitsConfig{MaxLimit: 2}},
	}
	events := []*nostr.Event{
		{ID: testID('1'), Kind: 1},
		{ID: testID('2'), Kind: 1, Tags: nostr.Tags{{"-"}}},
		{ID: testID('3'), Kind: 1059},
		{ID: testID('4'), Kind: nostr.KindEncryptedDirectMessage},
		{ID: testID('5'), Kind: 1},
		{ID: testID('6'), Kind: 1},
		{ID: testID('7'), Kind: 1},
	}
	handler := NewWithSource(cfg, func(_ context.Context, filter nostr.Filter, offset int) ([]*nostr.Event, error) {
		end := offset + filter.Limit
		if end > len(events) {
			end = len(events)
		}
		if offset >= len(events) {
			return []*nostr.Event{}, nil
		}
		return events[offset:end], nil
	})
	app := fiber.New()
	handler.Register(app.Group("/api/v1"))
	request := httptest.NewRequest("GET", "/api/v1/query?limit=2", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	defer response.Body.Close()
	var body eventsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Count != 2 || body.Events[0].ID != testID('1') || body.Events[1].ID != testID('5') || !body.More {
		t.Fatalf("response = %#v", body)
	}
}

func TestQueryExcludesRequestedTagBeforePagination(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 2, MaxScan: 20}, Security: config.SecurityConfig{Limits: config.SecurityLimitsConfig{MaxLimit: 2}}}
	events := []*nostr.Event{{ID: testID('1'), Kind: 1, Tags: nostr.Tags{{"p", "hidden"}}}, {ID: testID('2'), Kind: 1}}
	app := fiber.New()
	NewWithSource(cfg, func(_ context.Context, filter nostr.Filter, offset int) ([]*nostr.Event, error) {
		if offset >= len(events) {
			return []*nostr.Event{}, nil
		}
		return events[offset:], nil
	}).Register(app.Group("/api/v1"))
	response, err := app.Test(httptest.NewRequest("GET", "/api/v1/query?limit=2&no_p=hidden", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer response.Body.Close()
	var body eventsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Count != 1 || body.Events[0].ID != testID('2') {
		t.Fatalf("response = %#v", body)
	}
}

func TestAPIMiddlewareRejectsDisabledMethodsAndUpgrade(t *testing.T) {
	tests := []struct {
		name   string
		config config.APIConfig
		method string
		header string
		want   int
	}{
		{name: "disabled", config: config.APIConfig{}, method: "GET", want: fiber.StatusNotFound},
		{name: "method", config: config.APIConfig{Enabled: true}, method: "POST", want: fiber.StatusMethodNotAllowed},
		{name: "upgrade", config: config.APIConfig{Enabled: true}, method: "GET", header: "Upgrade", want: fiber.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{API: test.config, Security: config.SecurityConfig{Limits: config.SecurityLimitsConfig{MaxLimit: 10}}}
			app := fiber.New()
			NewWithSource(cfg, func(context.Context, nostr.Filter, int) ([]*nostr.Event, error) { return []*nostr.Event{}, nil }).Register(app.Group("/api/v1"))
			request := httptest.NewRequest(test.method, "/api/v1/query", nil)
			if test.header != "" {
				request.Header.Set(test.header, "websocket")
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			if response.StatusCode != test.want {
				t.Fatalf("status = %d want %d", response.StatusCode, test.want)
			}
		})
	}
}

func TestIdentifierFilterRejectsInvalidIdentifier(t *testing.T) {
	if _, err := identifierFilter("invalid"); err == nil {
		t.Fatal("identifierFilter() error = nil")
	}
}

func TestQueryRejectsInvalidKind(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 10}, Security: config.SecurityConfig{Limits: config.SecurityLimitsConfig{MaxLimit: 10}}}
	app := fiber.New()
	NewWithSource(cfg, func(context.Context, nostr.Filter, int) ([]*nostr.Event, error) { return nil, nil }).Register(app.Group("/api/v1"))

	for _, raw := range []string{"-1", "abc", ""} {
		response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/query?kind="+raw, nil))
		if err != nil {
			t.Fatalf("app.Test() error = %v", err)
		}
		if response.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("kind=%q status = %d, want %d", raw, response.StatusCode, fiber.StatusBadRequest)
		}
		response.Body.Close()
	}
}

func TestQueryUsesConfiguredDefaultLimit(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 2, MaxScan: 10}}
	app := fiber.New()
	NewWithSource(cfg, func(_ context.Context, filter nostr.Filter, _ int) ([]*nostr.Event, error) {
		if filter.Limit != 3 {
			t.Fatalf("scan limit = %d, want 3", filter.Limit)
		}
		return nil, nil
	}).Register(app.Group("/api/v1"))
	response, err := app.Test(httptest.NewRequest("GET", "/api/v1/query", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

func TestQueryPropagatesSort(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 10, MaxScan: 20}}
	h := NewWithSource(cfg, func(context.Context, nostr.Filter, int) ([]*nostr.Event, error) {
		t.Fatal("unordered source used")
		return nil, nil
	})
	var ascending bool
	h.orderedSource = func(_ context.Context, _ nostr.Filter, _ int, asc bool) ([]*nostr.Event, error) {
		ascending = asc
		return nil, nil
	}
	app := fiber.New()
	h.Register(app.Group("/api/v1"))
	response, err := app.Test(httptest.NewRequest("GET", "/api/v1/query?sort=asc", nil))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != fiber.StatusOK || !ascending {
		t.Fatalf("status=%d ascending=%v", response.StatusCode, ascending)
	}
	response, err = app.Test(httptest.NewRequest("GET", "/api/v1/query?sort=invalid", nil))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("invalid sort status=%d", response.StatusCode)
	}
}

func TestQueryContinuesWhenDatabaseLimitIsLower(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 4, MaxScan: 10}, Relay: config.RelayConfig{QueryLimit: 2}}
	events := []*nostr.Event{{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}}
	app := fiber.New()
	NewWithSource(cfg, func(_ context.Context, filter nostr.Filter, offset int) ([]*nostr.Event, error) {
		if filter.Limit > 2 {
			t.Fatalf("limit = %d", filter.Limit)
		}
		if offset >= len(events) {
			return nil, nil
		}
		return events[offset:min(offset+filter.Limit, len(events))], nil
	}).Register(app.Group("/api/v1"))
	response, err := app.Test(httptest.NewRequest("GET", "/api/v1/query?limit=3", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var body eventsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Count != 3 || !body.More {
		t.Fatalf("response = %#v", body)
	}
}

func TestQueryDoesNotInventMoreAtScanLimit(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 2, MaxScan: 2}}
	app := fiber.New()
	NewWithSource(cfg, func(context.Context, nostr.Filter, int) ([]*nostr.Event, error) {
		return []*nostr.Event{{ID: "1", Tags: nostr.Tags{{"-"}}}, {ID: "2", Tags: nostr.Tags{{"-"}}}}, nil
	}).Register(app.Group("/api/v1"))
	response, err := app.Test(httptest.NewRequest("GET", "/api/v1/query?limit=1", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

func testID(char byte) string {
	return string(make([]byte, 0)) + string(bytesRepeat(char, 64))
}

func bytesRepeat(char byte, count int) []byte {
	result := make([]byte, count)
	for index := range result {
		result[index] = char
	}
	return result
}
