package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

func TestQueryHidesProtectedAndGiftWrapBeforePagination(t *testing.T) {
	cfg := &config.Config{
		API:      config.APIConfig{Enabled: true, MaxLimit: 2, MaxScan: 20},
		Security: config.SecurityConfig{Limits: config.SecurityLimitsConfig{MaxLimit: 2}},
	}
	events := []*nostr.Event{
		{ID: testID('1'), Kind: 1},
		{ID: testID('2'), Kind: 1, Tags: nostr.Tags{{"-"}}},
		{ID: testID('3'), Kind: 1059},
		{ID: testID('4'), Kind: 1},
		{ID: testID('5'), Kind: 1},
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
	if body.Count != 2 || body.Events[0].ID != testID('1') || body.Events[1].ID != testID('4') || !body.More {
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
