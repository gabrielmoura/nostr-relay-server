package api

import (
	"context"
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

func TestNIPFECountFiltersPrivateDMs(t *testing.T) {
	previous := findNIPFEEvents
	t.Cleanup(func() { findNIPFEEvents = previous })
	findNIPFEEvents = func(context.Context, string, nostr.Filter) ([]*nostr.Event, error) {
		return []*nostr.Event{
			{ID: "private-four", Kind: nostr.KindEncryptedDirectMessage, PubKey: "owner"},
			{ID: "private-wrap", Kind: 1059, Tags: nostr.Tags{{"p", "owner"}}},
			{ID: "public-note", Kind: nostr.KindTextNote},
		}, nil
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
