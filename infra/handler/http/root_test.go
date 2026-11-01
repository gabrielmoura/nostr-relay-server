package http

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	assets "github.com/gabrielmoura/nostr-relay-server/internal/embed"
	"github.com/gofiber/fiber/v2"
)

func TestNostrIconServesEmbeddedPNG(t *testing.T) {
	app := fiber.New()
	app.Get("/nostr.png", NostrIcon())

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
	if got := response.Header.Get(fiber.HeaderCacheControl); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q, want public, max-age=31536000, immutable", got)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read icon response: %v", err)
	}
	if !bytes.Equal(body, assets.NostrPNG) {
		t.Fatal("icon response does not match embedded asset")
	}
}

func TestRootUpgradeServesNIP11WithRequiredHeaders(t *testing.T) {
	app := fiber.New()
	app.Get("/", RootUpgrade(&config.Config{
		RelayInformation: config.RelayInformationDocument{SupportedNIPs: []int{1, 11}},
	}, nil))

	request := httptest.NewRequest(fiber.MethodGet, "/", nil)
	request.Header.Set(fiber.HeaderAccept, "application/nostr+json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("NIP-11 request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	if got := response.Header.Get(fiber.HeaderContentType); got != "application/nostr+json" {
		t.Fatalf("Content-Type = %q, want application/nostr+json", got)
	}
	if got := response.Header.Get(fiber.HeaderAccessControlAllowOrigin); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
	if got := response.Header.Get(fiber.HeaderAccessControlAllowHeaders); got == "" {
		t.Fatal("Access-Control-Allow-Headers is missing")
	}
	if got := response.Header.Get(fiber.HeaderAccessControlAllowMethods); got == "" {
		t.Fatal("Access-Control-Allow-Methods is missing")
	}
}

func TestRootUpgradeWithPublicUIServesHTMLWithoutChangingNIP11(t *testing.T) {
	app := fiber.New()
	landing := func(c *fiber.Ctx) error { return c.SendString("<html>public relay</html>") }
	app.Get("/", RootUpgradeWithPublicUI(&config.Config{}, landing))

	htmlRequest := httptest.NewRequest(fiber.MethodGet, "/", nil)
	htmlRequest.Header.Set(fiber.HeaderAccept, fiber.MIMETextHTML)
	htmlResponse, err := app.Test(htmlRequest)
	if err != nil {
		t.Fatalf("HTML request failed: %v", err)
	}
	defer htmlResponse.Body.Close()
	if htmlResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("HTML status = %d, want %d", htmlResponse.StatusCode, fiber.StatusOK)
	}

	nip11Request := httptest.NewRequest(fiber.MethodGet, "/", nil)
	nip11Request.Header.Set(fiber.HeaderAccept, "application/nostr+json")
	nip11Response, err := app.Test(nip11Request)
	if err != nil {
		t.Fatalf("NIP-11 request failed: %v", err)
	}
	defer nip11Response.Body.Close()
	if got := nip11Response.Header.Get(fiber.HeaderContentType); got != "application/nostr+json" {
		t.Fatalf("NIP-11 content type = %q", got)
	}
}

func TestRootUpgradeWithPublicUIDoesNotHandleNIP86(t *testing.T) {
	app := fiber.New()
	app.All("/", RootUpgradeWithPublicUI(&config.Config{NIP86: config.NIP86Config{Enabled: true}}, func(c *fiber.Ctx) error {
		return c.SendString("public")
	}))

	request := httptest.NewRequest(fiber.MethodPost, "/", bytes.NewBufferString(`{"method":"supportedmethods"}`))
	request.Header.Set(fiber.HeaderContentType, "application/nostr+json+rpc")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("public NIP-86 request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusUpgradeRequired {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusUpgradeRequired)
	}
	if got := response.Header.Get(fiber.HeaderContentType); got == "application/nostr+json+rpc" {
		t.Fatal("public root exposed NIP-86 JSON-RPC")
	}
}

func TestRootUpgradeWithPublicUIKeepsWebSocketUpgradePath(t *testing.T) {
	app := fiber.New()
	app.Use("/", RootUpgradeWithPublicUI(&config.Config{}, func(c *fiber.Ctx) error { return c.SendString("public") }))
	app.Get("/", func(c *fiber.Ctx) error {
		if c.Locals("wss") == nil {
			t.Fatal("websocket context was not initialized")
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	request := httptest.NewRequest(fiber.MethodGet, "/", nil)
	request.Header.Set(fiber.HeaderConnection, "Upgrade")
	request.Header.Set(fiber.HeaderUpgrade, "websocket")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("websocket upgrade path failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusNoContent)
	}
}

func TestNIP11WithPrivacy_AdvertisesNIP29(t *testing.T) {
	t.Parallel()

	doc, ok := NIP11WithPrivacy(&config.Config{
		NIP29: config.NIP29Config{Enabled: true},
		RelayInformation: config.RelayInformationDocument{
			SupportedNIPs: []int{1, 11},
		},
	}).(map[string]any)
	if !ok {
		t.Fatal("expected augmented NIP-11 document")
	}
	nip29, ok := doc["nip29"].(map[string]any)
	if !ok {
		t.Fatal("expected nip29 support object")
	}
	if enabled, ok := nip29["subgroups"].(bool); !ok || !enabled {
		t.Fatalf("nip29.subgroups = %v, want true", nip29["subgroups"])
	}

	supported, ok := doc["supported_nips"].([]any)
	if !ok {
		t.Fatalf("supported_nips type = %T", doc["supported_nips"])
	}
	for _, nip := range supported {
		if isNIP29Number(nip) {
			return
		}
	}
	t.Fatalf("supported_nips = %v, want 29", supported)
}

func TestNIP11WithPrivacy_AdvertisesNIP70(t *testing.T) {
	t.Parallel()

	doc, ok := NIP11WithPrivacy(&config.Config{
		NIP70: config.NIP70Config{Enabled: true},
		RelayInformation: config.RelayInformationDocument{
			SupportedNIPs: []int{1, 11},
		},
	}).(map[string]any)
	if !ok {
		t.Fatal("expected augmented NIP-11 document")
	}

	supported, ok := doc["supported_nips"].([]any)
	if !ok {
		t.Fatalf("supported_nips type = %T", doc["supported_nips"])
	}
	for _, nip := range supported {
		if isNIPNumber(nip, 70) {
			return
		}
	}
	t.Fatalf("supported_nips = %v, want 70", supported)
}

func TestNIP11WithPrivacyRetainsEndpointsWhenNIP70AlreadyAdvertised(t *testing.T) {
	doc, ok := NIP11WithPrivacy(&config.Config{
		NIP70:            config.NIP70Config{Enabled: true},
		API:              config.APIConfig{NIPFE: config.NIPFEConfig{Enabled: true}},
		RelayInformation: config.RelayInformationDocument{SupportedNIPs: []int{70}},
	}).(map[string]any)
	if !ok {
		t.Fatal("expected augmented NIP-11 document")
	}
	if doc["nip_fe"] != true {
		t.Fatalf("nip_fe = %v, want true", doc["nip_fe"])
	}
	if _, ok := doc["public_endpoints"]; !ok {
		t.Fatal("public endpoints omitted when NIP-70 was already advertised")
	}
}

func TestNIP11WithPrivacyAdvertisesTrustedNIP42RelayURLs(t *testing.T) {
	doc, ok := NIP11WithPrivacy(&config.Config{
		DMImport: config.DMImportConfig{Enabled: true},
		RelayInformation: config.RelayInformationDocument{
			CanonicalURL:  "wss://relay.example",
			AuthRelayURLs: []string{"ws://localhost:4869"},
		},
	}).(map[string]any)
	if !ok {
		t.Fatal("expected augmented NIP-11 document")
	}
	if doc["auth_relay_url"] != "wss://relay.example" {
		t.Fatalf("auth_relay_url = %#v", doc["auth_relay_url"])
	}
	urls, ok := doc["auth_relay_urls"].([]string)
	if !ok || len(urls) != 2 || urls[1] != "ws://localhost:4869" {
		t.Fatalf("auth_relay_urls = %#v", doc["auth_relay_urls"])
	}
}

func TestNIP11WithPrivacy_AdvertisesNIPFEOnlyWhenEnabled(t *testing.T) {
	doc, ok := NIP11WithPrivacy(&config.Config{API: config.APIConfig{NIPFE: config.NIPFEConfig{Enabled: true}}}).(map[string]any)
	if !ok || doc["nip_fe"] != true {
		t.Fatalf("NIP-FE was not advertised: %#v", doc)
	}

	disabled := NIP11WithPrivacy(&config.Config{})
	if doc, ok := disabled.(map[string]any); ok {
		if _, found := doc["nip_fe"]; found {
			t.Fatalf("disabled NIP-FE was advertised: %#v", doc)
		}
	}
}

func TestNIP11WithPrivacyAdvertisesOnlyEnabledPublicEndpoints(t *testing.T) {
	doc, ok := NIP11WithPrivacy(&config.Config{
		Store: config.StoreConfig{Enabled: true},
		API:   config.APIConfig{Enabled: true, NIPFE: config.NIPFEConfig{Enabled: true}},
	}).(map[string]any)
	if !ok {
		t.Fatal("expected augmented NIP-11 document")
	}
	endpoints, ok := doc["public_endpoints"].([]publicEndpoint)
	if !ok {
		t.Fatalf("public_endpoints = %T", doc["public_endpoints"])
	}
	paths := map[string]bool{}
	for _, endpoint := range endpoints {
		if endpoint.Method == "" || endpoint.Path == "" || endpoint.Description == "" {
			t.Fatalf("invalid public endpoint: %#v", endpoint)
		}
		paths[endpoint.Path] = true
	}
	for _, path := range []string{"/", "/.well-known/nostr.json", "/.well-known/nostr/nip96.json", "/upload", "/api/v1"} {
		if !paths[path] {
			t.Fatalf("missing public endpoint path %q: %#v", path, endpoints)
		}
	}
}

func TestNIP11WithPrivacy_AdvertisesRelaySyncOnlyWhenEnabled(t *testing.T) {
	doc := NIP11WithPrivacy(&config.Config{RelaySync: config.RelaySyncConfig{Enabled: true, HTTPEnabled: true, SnapshotEnabled: true}})
	m, ok := doc.(map[string]any)
	if !ok {
		t.Fatalf("document type = %T", doc)
	}
	if _, ok := m["relay_sync_v2"]; !ok {
		t.Fatal("relay_sync_v2 extension missing")
	}
	if _, ok := m["supported_nip_extensions"]; !ok {
		t.Fatal("supported_nip_extensions missing")
	}
}
