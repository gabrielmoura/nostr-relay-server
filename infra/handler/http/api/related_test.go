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

func TestRelatedEventsMergeAndFilterBeforePagination(t *testing.T) {
	target := strings.Repeat("a", 64)
	events := []*nostr.Event{
		{ID: "1", CreatedAt: 5, Tags: nostr.Tags{{"e", target}, {"q", target}}},
		{ID: "2", CreatedAt: 4, Tags: nostr.Tags{{"q", target}, {"-"}}},
		{ID: "3", CreatedAt: 3, Tags: nostr.Tags{{"q", target}}},
		{ID: "4", CreatedAt: 2, Tags: nostr.Tags{{"e", target}}},
	}
	cfg := &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 2, MaxScan: 20}}
	app := fiber.New()
	NewWithSource(cfg, func(_ context.Context, filter nostr.Filter, offset int) ([]*nostr.Event, error) {
		var matched []*nostr.Event
		for _, event := range events {
			for key, values := range filter.Tags {
				for _, tag := range event.Tags {
					if len(tag) > 1 && tag[0] == key && tag[1] == values[0] {
						matched = append(matched, event)
					}
				}
			}
		}
		if offset >= len(matched) {
			return nil, nil
		}
		return matched[offset:min(offset+filter.Limit, len(matched))], nil
	}).Register(app.Group("/api/v1"))
	response, err := app.Test(httptest.NewRequest("GET", "/api/v1/ids/"+strings.ToUpper(target)+"/related?limit=1&offset=1", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var body eventsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusOK || body.Count != 1 || body.Events[0].ID != "3" || !body.More {
		t.Fatalf("status=%d body=%#v", response.StatusCode, body)
	}
}
