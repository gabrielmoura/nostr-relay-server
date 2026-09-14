package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestAdminUILanding_ServesEmbeddedDashboardLandingPage(t *testing.T) {
	t.Parallel()

	app := fiber.New()
	app.Get("/", AdminUILanding())

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if !strings.Contains(response.Header.Get(fiber.HeaderContentType), "text/html") {
		t.Fatalf("content type = %q, want text/html", response.Header.Get(fiber.HeaderContentType))
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if strings.Contains(string(body), "cdn.jsdelivr.net") {
		t.Fatal("landing page must not load Tailwind from a CDN")
	}
	if !strings.Contains(string(body), "href=\"/todash.css\"") {
		t.Fatal("landing page must load the embedded stylesheet")
	}
}

func TestAdminUILandingCSS_ServesEmbeddedStylesheet(t *testing.T) {
	t.Parallel()

	app := fiber.New()
	app.Get("/todash.css", AdminUILandingCSS())

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/todash.css", nil))
	if err != nil {
		t.Fatalf("GET /todash.css: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if !strings.Contains(response.Header.Get(fiber.HeaderContentType), "text/css") {
		t.Fatalf("content type = %q, want text/css", response.Header.Get(fiber.HeaderContentType))
	}
}
