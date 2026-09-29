package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gofiber/fiber/v2"
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
