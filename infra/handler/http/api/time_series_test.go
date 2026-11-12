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

func TestTimeSeriesCalendarAndValidation(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 10, MaxScan: 20}}
	app := fiber.New()
	NewWithSource(cfg, func(context.Context, nostr.Filter, int) ([]*nostr.Event, error) { return nil, nil }).Register(app.Group("/api/v1"))
	base := "/api/v1/" + strings.Repeat("a", 64) + "/1/"
	for _, tc := range []struct {
		path, field string
		size        int
	}{{"daily?year=2024&month=2", "days", 29}, {"hourly?year=2024&month=2&day=29", "hours", 24}} {
		response, err := app.Test(httptest.NewRequest("GET", base+tc.path, nil))
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		err = json.NewDecoder(response.Body).Decode(&body)
		response.Body.Close()
		if err != nil || response.StatusCode != fiber.StatusOK || len(body[tc.field].([]any)) != tc.size || body["total"] != float64(0) {
			t.Fatalf("%s status=%d body=%#v err=%v", tc.path, response.StatusCode, body, err)
		}
	}
	for _, path := range []string{"daily?year=2024&month=13", "hourly?year=2023&month=2&day=29", "daily?since=1"} {
		response, err := app.Test(httptest.NewRequest("GET", base+path, nil))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("%s status=%d", path, response.StatusCode)
		}
	}
}

func TestMonthlyRangeClipsBucketsAndBounds(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 10, MaxScan: 20}}
	var windows [][2]int64
	app := fiber.New()
	NewWithSource(cfg, func(_ context.Context, filter nostr.Filter, _ int) ([]*nostr.Event, error) {
		windows = append(windows, [2]int64{int64(*filter.Since), int64(*filter.Until)})
		return nil, nil
	}).Register(app.Group("/api/v1"))
	base := "/api/v1/" + strings.Repeat("a", 64) + "/1/monthly"
	response, err := app.Test(httptest.NewRequest("GET", base+"?since=1706745600&until=1709424000", nil))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	err = json.NewDecoder(response.Body).Decode(&body)
	response.Body.Close()
	if err != nil || response.StatusCode != fiber.StatusOK || len(body["months"].([]any)) != 2 {
		t.Fatalf("status=%d body=%#v err=%v", response.StatusCode, body, err)
	}
	if len(windows) != 2 || windows[0][0] != 1706745600 || windows[1][1] != 1709424000 {
		t.Fatalf("windows=%v", windows)
	}
	response, err = app.Test(httptest.NewRequest("GET", base+"?since=0&until=4102444800", nil))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("long range status=%d", response.StatusCode)
	}
}
