package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

func TestAuthorAggregatesExcludeProtectedEvents(t *testing.T) {
	pubkey := strings.Repeat("a", 64)
	events := []*nostr.Event{{Kind: 1, CreatedAt: 3}, {Kind: 3, CreatedAt: 2, Tags: nostr.Tags{{"-"}}}, {Kind: 1, CreatedAt: 1}}
	cfg := &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 10, MaxScan: 20}}
	h := NewWithSource(cfg, nil)
	h.orderedSource = func(_ context.Context, filter nostr.Filter, offset int, ascending bool) ([]*nostr.Event, error) {
		ordered := append([]*nostr.Event(nil), events...)
		if ascending {
			ordered[0], ordered[2] = ordered[2], ordered[0]
		}
		if offset >= len(ordered) {
			return nil, nil
		}
		return ordered[offset:min(offset+filter.Limit, len(ordered))], nil
	}
	app := fiber.New()
	h.Register(app.Group("/api/v1"))
	for _, path := range []string{"kinds", "stats"} {
		response, err := app.Test(httptest.NewRequest("GET", "/api/v1/"+pubkey+"/"+path, nil))
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		err = json.NewDecoder(response.Body).Decode(&body)
		response.Body.Close()
		if err != nil || response.StatusCode != fiber.StatusOK {
			t.Fatalf("%s status=%d error=%v", path, response.StatusCode, err)
		}
		kinds := body["kinds"].([]any)
		if len(kinds) != 1 || kinds[0].(map[string]any)["count"] != float64(2) {
			t.Fatalf("%s body=%#v", path, body)
		}
		if path == "stats" && (body["total"] != float64(2) || body["first_seen"] != float64(1) || body["last_seen"] != float64(3)) {
			t.Fatalf("stats=%#v", body)
		}
	}
}

func TestRelayAggregatesFilterAndReportTruncation(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 10, MaxScan: 3}}
	events := []*nostr.Event{
		{ID: "1", Kind: 1, PubKey: "alice"},
		{ID: "2", Kind: 2, PubKey: "hidden", Tags: nostr.Tags{{"-"}}},
		{ID: "3", Kind: 1, PubKey: "alice"},
		{ID: "4", Kind: 3, PubKey: "bob"},
	}
	app := fiber.New()
	NewWithSource(cfg, func(_ context.Context, filter nostr.Filter, offset int) ([]*nostr.Event, error) {
		if offset >= len(events) {
			return nil, nil
		}
		return events[offset:min(offset+filter.Limit, len(events))], nil
	}).Register(app.Group("/api/v1"))
	for _, path := range []string{"kinds", "top-authors"} {
		response, err := app.Test(httptest.NewRequest("GET", "/api/v1/relay/"+path+"?limit=1", nil))
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		err = json.NewDecoder(response.Body).Decode(&body)
		response.Body.Close()
		if err != nil || response.StatusCode != fiber.StatusOK || body["approximate"] != true || body["filtered"] != true {
			t.Fatalf("%s status=%d body=%#v err=%v", path, response.StatusCode, body, err)
		}
		field := "kinds"
		if path == "top-authors" {
			field = "authors"
		}
		items := body[field].([]any)
		if len(items) != 1 || items[0].(map[string]any)["count"] != float64(2) {
			t.Fatalf("%s body=%#v", path, body)
		}
	}
}
