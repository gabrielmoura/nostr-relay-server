package net

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	assets "github.com/gabrielmoura/nostr-relay-server/internal/embed"
	"github.com/gofiber/fiber/v2"
)

func TestSetupExternalRoutesServesEmbeddedNostrIcon(t *testing.T) {
	app := fiber.New()
	factory := &RouterFactory{Config: &config.Config{}}
	factory.setupExternalRoutes(app)

	response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/nostr.png", nil))
	if err != nil {
		t.Fatalf("icon request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	if got := response.Header.Get(fiber.HeaderContentType); got != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", got)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read icon response: %v", err)
	}
	if !bytes.Equal(body, assets.NostrPNG) {
		t.Fatal("icon response does not match embedded asset")
	}
}
