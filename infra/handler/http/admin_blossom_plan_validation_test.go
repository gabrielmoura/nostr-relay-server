package http

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestBlossomPlanRejectsNegativeQuotas(t *testing.T) {
	app := fiber.New()
	app.Post("/plans", BlossomPlanUpsert())
	for _, body := range []string{
		`{"id":"plan","name":"Plan","scope":"free","storage_quota_bytes":-1}`,
		`{"id":"plan","name":"Plan","scope":"free","egress_quota_bytes":-1}`,
	} {
		req := httptest.NewRequest("POST", "/plans", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("body %s: got status %d, want 400", body, resp.StatusCode)
		}
	}
}

func TestBlossomMirrorRejectsInvalidInputBeforeDispatch(t *testing.T) {
	app := fiber.New()
	app.Post("/mirror", BlossomMirror())
	for _, body := range []string{
		`{"source_url":"file:///etc/passwd","expected_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
		`{"source_url":"https://example.com/file","expected_sha256":"invalid"}`,
		`{"source_url":"http://127.0.0.1/file","expected_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
	} {
		req := httptest.NewRequest("POST", "/mirror", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("body %s: got status %d, want 400", body, resp.StatusCode)
		}
	}
}
