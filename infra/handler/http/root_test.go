package http

import (
	"net/http/httptest"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gofiber/fiber/v2"
)

func TestRootUpgradeServesNIP11WithRequiredHeaders(t *testing.T) {
	app := fiber.New()
	app.Get("/", RootUpgrade(&config.Config{
		RelayInformation: config.RelayInformationDocument{SupportedNIPs: []int{1, 11}},
	}))

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
