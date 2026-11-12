package api

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

func TestNIPFERejectsMultipleAndUnsupportedFrames(t *testing.T) {
	app := fiber.New()
	app.Post("/", NIPFE(&config.Config{API: config.APIConfig{NIPFE: config.NIPFEConfig{Enabled: true, MaxBodyBytes: 1024}}}))
	for _, payload := range []string{"[\"AUTH\"]", "[\"REQ\"] [\"REQ\"]"} {
		request := httptest.NewRequest("POST", "/", strings.NewReader(payload))
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("app.Test() error = %v", err)
		}
		if response.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("payload %q status = %d", payload, response.StatusCode)
		}
		if !strings.Contains(response.Header.Get(fiber.HeaderContentType), "application/x-ndjson") {
			t.Fatalf("payload %q content type = %q", payload, response.Header.Get(fiber.HeaderContentType))
		}
	}
}

func TestNIPFEBoundsFiltersAndResponseBytes(t *testing.T) {
	cfg := &config.Config{Security: config.SecurityConfig{Limits: config.SecurityLimitsConfig{MaxFiltersPerReq: 1}}, API: config.APIConfig{NIPFE: config.NIPFEConfig{MaxResponseBytes: 64}}}
	for _, command := range []string{"REQ", "COUNT"} {
		body := []byte(fmt.Sprintf(`[%q,"id",{},{}]`, command))
		_, status := executeNIPFEWithConfig(context.Background(), body, "", cfg)
		if status != fiber.StatusBadRequest {
			t.Fatalf("%s filter limit status=%d", command, status)
		}
	}
	previous := findNIPFEEvents
	t.Cleanup(func() { findNIPFEEvents = previous })
	findNIPFEEvents = func(context.Context, string, nostr.Filter) ([]*nostr.Event, error) {
		return []*nostr.Event{{ID: "event", Kind: 1, Content: strings.Repeat("x", 128)}}, nil
	}
	frames, status := executeNIPFEWithConfig(context.Background(), []byte(`["REQ","id",{}]`), "", cfg)
	if status != fiber.StatusRequestEntityTooLarge || len(frames) != 1 {
		t.Fatalf("status=%d frames=%#v", status, frames)
	}
}

func TestNIPFELimitsResponseEvents(t *testing.T) {
	previous := findNIPFEEvents
	t.Cleanup(func() { findNIPFEEvents = previous })
	findNIPFEEvents = func(context.Context, string, nostr.Filter) ([]*nostr.Event, error) {
		return []*nostr.Event{{ID: "one", Kind: nostr.KindTextNote}, {ID: "two", Kind: nostr.KindTextNote}}, nil
	}
	cfg := &config.Config{API: config.APIConfig{NIPFE: config.NIPFEConfig{
		MaxResponseBytes:  1024,
		MaxResponseEvents: 1,
	}}}
	frames, status := executeNIPFEWithConfig(context.Background(), []byte(`["REQ","id",{}]`), "", cfg)
	if status != fiber.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d frames=%#v", status, frames)
	}
	closed := frames[0].([]any)
	if closed[2] != "error: response exceeds event limit" {
		t.Fatalf("close reason=%#v", closed)
	}
}

func TestNIPFECountSharesScanBudgetAcrossFilters(t *testing.T) {
	previous := findNIPFECountPage
	t.Cleanup(func() { findNIPFECountPage = previous })
	findNIPFECountPage = func(_ context.Context, _ string, filter nostr.Filter, _ int) ([]*nostr.Event, int, error) {
		if filter.Authors[0] == "first" {
			return []*nostr.Event{{ID: "first", Kind: nostr.KindTextNote}}, 1, nil
		}
		events := make([]*nostr.Event, filter.Limit)
		for i := range events {
			events[i] = &nostr.Event{ID: fmt.Sprintf("second-%d", i), Kind: nostr.KindTextNote, Tags: nostr.Tags{{"-"}}}
		}
		return events, len(events), nil
	}
	cfg := &config.Config{API: config.APIConfig{MaxScan: 3}, Relay: config.RelayConfig{QueryLimit: 2}}
	frames, status := executeNIPFEWithConfig(context.Background(), []byte(`["COUNT","count",{"authors":["first"]},{"authors":["second"]}]`), "", cfg)
	if status != fiber.StatusServiceUnavailable {
		t.Fatalf("status=%d frames=%#v", status, frames)
	}
	closed := frames[0].([]any)
	if closed[2] != "error: request work limit exceeded" {
		t.Fatalf("close reason=%#v", closed)
	}
}

func TestNIPFEReqSharesScanBudgetAcrossFilters(t *testing.T) {
	previousEvents, previousPage := findNIPFEEvents, findNIPFECountPage
	t.Cleanup(func() {
		findNIPFEEvents, findNIPFECountPage = previousEvents, previousPage
	})
	findNIPFEEvents = oneShotEvents
	findNIPFECountPage = func(_ context.Context, _ string, filter nostr.Filter, _ int) ([]*nostr.Event, int, error) {
		if filter.Authors[0] == "first" {
			return []*nostr.Event{{ID: "first", Kind: nostr.KindTextNote}}, 1, nil
		}
		events := make([]*nostr.Event, filter.Limit)
		for i := range events {
			events[i] = &nostr.Event{ID: fmt.Sprintf("second-%d", i), Kind: nostr.KindTextNote, Tags: nostr.Tags{{"-"}}}
		}
		return events, len(events), nil
	}
	cfg := &config.Config{API: config.APIConfig{
		MaxScan: 3,
		NIPFE:   config.NIPFEConfig{MaxResponseBytes: 1024},
	}, Relay: config.RelayConfig{QueryLimit: 2}}
	frames, status := executeNIPFEWithConfig(context.Background(), []byte(`["REQ","req",{"authors":["first"]},{"authors":["second"]}]`), "", cfg)
	if status != fiber.StatusServiceUnavailable {
		t.Fatalf("status=%d frames=%#v", status, frames)
	}
	closed := frames[0].([]any)
	if closed[2] != "error: request work limit exceeded" {
		t.Fatalf("close reason=%#v", closed)
	}
}

func TestNIPFECountScansPastRequestLimit(t *testing.T) {
	previous := findNIPFECountPage
	t.Cleanup(func() { findNIPFECountPage = previous })
	findNIPFECountPage = func(_ context.Context, _ string, filter nostr.Filter, offset int) ([]*nostr.Event, int, error) {
		var events []*nostr.Event
		for i := offset; i < 105 && len(events) < filter.Limit; i++ {
			events = append(events, &nostr.Event{ID: fmt.Sprint(i), Kind: nostr.KindTextNote})
		}
		return events, len(events), nil
	}
	frames, status := executeNIPFE(context.Background(), []byte(`["COUNT","count",{"limit":1}]`), "")
	if status != fiber.StatusOK {
		t.Fatalf("status = %d, frames = %#v", status, frames)
	}
	payload := frames[0].([]any)[2].(map[string]int64)
	if payload["count"] != 105 {
		t.Fatalf("count = %d, want 105", payload["count"])
	}
}

func TestNIPFEReqFillsLimitAfterVisibilityFiltering(t *testing.T) {
	previousSource, previousConfig := findNIPFECountPage, config.Cfg
	t.Cleanup(func() { findNIPFECountPage, config.Cfg = previousSource, previousConfig })
	config.Cfg = &config.Config{API: config.APIConfig{MaxScan: 5}, Relay: config.RelayConfig{QueryLimit: 2}}
	findNIPFECountPage = func(_ context.Context, _ string, filter nostr.Filter, offset int) ([]*nostr.Event, int, error) {
		switch offset {
		case 0:
			return []*nostr.Event{{ID: "hidden", Tags: nostr.Tags{{"-"}}}, {ID: "one"}}, 2, nil
		case 2:
			return []*nostr.Event{{ID: "two"}}, 1, nil
		default:
			t.Fatalf("unexpected offset %d with limit %d", offset, filter.Limit)
			return nil, 0, nil
		}
	}
	events, err := oneShotEvents(context.Background(), "", nostr.Filter{Limit: 2})
	if err != nil || len(events) != 2 || events[0].ID != "one" || events[1].ID != "two" {
		t.Fatalf("events=%#v err=%v", events, err)
	}
}

func TestNIPFECountRejectsIncompleteScan(t *testing.T) {
	previousSource, previousConfig := findNIPFECountPage, config.Cfg
	t.Cleanup(func() { findNIPFECountPage, config.Cfg = previousSource, previousConfig })
	config.Cfg = &config.Config{API: config.APIConfig{MaxScan: 2}, Relay: config.RelayConfig{QueryLimit: 2}}
	findNIPFECountPage = func(_ context.Context, _ string, filter nostr.Filter, offset int) ([]*nostr.Event, int, error) {
		events := make([]*nostr.Event, filter.Limit)
		for i := range events {
			events[i] = &nostr.Event{ID: fmt.Sprint(offset + i), Kind: nostr.KindTextNote}
		}
		return events, len(events), nil
	}
	frames, status := executeNIPFE(context.Background(), []byte(`["COUNT","count",{}]`), "")
	if status != fiber.StatusServiceUnavailable {
		t.Fatalf("status = %d, frames = %#v", status, frames)
	}
}

func TestVisibleEventsRestrictsPrivateDMsForNIPFE(t *testing.T) {
	owner := "owner"
	other := "other"
	events := []*nostr.Event{
		{ID: "private-four", Kind: nostr.KindEncryptedDirectMessage, PubKey: owner},
		{ID: "private-wrap", Kind: 1059, PubKey: "ephemeral", Tags: nostr.Tags{{"p", owner}}},
		{ID: "public-note", Kind: nostr.KindTextNote, PubKey: other},
	}

	visibleWithoutAuth := filterNIPFEVisibleEvents(events, "")
	if len(visibleWithoutAuth) != 1 || visibleWithoutAuth[0].ID != "public-note" {
		t.Fatalf("unauthenticated NIP-FE events = %#v", visibleWithoutAuth)
	}

	visibleForOwner := filterNIPFEVisibleEvents(events, owner)
	if len(visibleForOwner) != len(events) {
		t.Fatalf("owner NIP-FE events = %#v, want all owner-visible events", visibleForOwner)
	}

	visibleForOther := filterNIPFEVisibleEvents(events, other)
	if len(visibleForOther) != 1 || visibleForOther[0].ID != "public-note" {
		t.Fatalf("other NIP-FE events = %#v", visibleForOther)
	}
}

func TestNIPFEExcludesProtectedEvents(t *testing.T) {
	events := []*nostr.Event{
		{ID: "protected", Kind: nostr.KindTextNote, PubKey: "owner", Tags: nostr.Tags{{"-"}}},
		{ID: "public", Kind: nostr.KindTextNote},
	}
	for _, authed := range []string{"", "owner", "other"} {
		visible := filterNIPFEVisibleEvents(events, authed)
		if len(visible) != 1 || visible[0].ID != "public" {
			t.Fatalf("authed %q: visible events = %#v", authed, visible)
		}
	}
}

func TestNIPFECountFiltersPrivateDMs(t *testing.T) {
	previous := findNIPFECountPage
	t.Cleanup(func() { findNIPFECountPage = previous })
	findNIPFECountPage = func(context.Context, string, nostr.Filter, int) ([]*nostr.Event, int, error) {
		return []*nostr.Event{
			{ID: "private-four", Kind: nostr.KindEncryptedDirectMessage, PubKey: "owner"},
			{ID: "private-wrap", Kind: 1059, Tags: nostr.Tags{{"p", "owner"}}},
			{ID: "public-note", Kind: nostr.KindTextNote},
		}, 3, nil
	}

	frames, status := executeNIPFE(context.Background(), []byte(`["COUNT","count",{}]`), "")
	if status != fiber.StatusOK {
		t.Fatalf("COUNT status = %d", status)
	}
	if len(frames) != 1 {
		t.Fatalf("COUNT frames = %#v", frames)
	}
	frame, ok := frames[0].([]any)
	if !ok || len(frame) != 3 {
		t.Fatalf("COUNT frame = %#v", frames[0])
	}
	payload, ok := frame[2].(map[string]int64)
	if !ok || payload["count"] != 1 {
		t.Fatalf("COUNT payload = %#v, want only the public event", frame[2])
	}
}

func TestNIPFECountDoesNotDuplicateOverlappingFilters(t *testing.T) {
	previous := findNIPFECountPage
	t.Cleanup(func() { findNIPFECountPage = previous })
	findNIPFECountPage = func(context.Context, string, nostr.Filter, int) ([]*nostr.Event, int, error) {
		return []*nostr.Event{{ID: "same", Kind: nostr.KindTextNote}}, 1, nil
	}

	frames, status := executeNIPFE(context.Background(), []byte(`["COUNT","count",{},{}]`), "")
	if status != fiber.StatusOK {
		t.Fatalf("COUNT status = %d", status)
	}
	frame := frames[0].([]any)
	payload := frame[2].(map[string]int64)
	if payload["count"] != 1 {
		t.Fatalf("COUNT payload = %#v, want one distinct event", payload)
	}
}
