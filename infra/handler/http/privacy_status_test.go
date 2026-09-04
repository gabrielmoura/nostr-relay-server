package http

import (
	stdjson "encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/infra/net/privacy"
	"github.com/gofiber/fiber/v2"
)

func TestPrivacyStatusSerializesEmptyPayloadWhenManagerIsNil(t *testing.T) {
	previous := privacy.GetManager()
	privacy.SetManager(nil)
	t.Cleanup(func() { privacy.SetManager(previous) })

	app := fiber.New()
	app.Get("/privacy/status", PrivacyStatus())

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/privacy/status", nil))
	if err != nil {
		t.Fatalf("request privacy status: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var payload PrivacyStatusResponse
	if err := stdjson.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode privacy status: %v", err)
	}
	if payload.Enabled {
		t.Fatal("enabled = true, want false")
	}
	if payload.Persistence {
		t.Fatal("persistence = true, want false")
	}
	if payload.StateDir != "" {
		t.Fatalf("state_dir = %q, want empty", payload.StateDir)
	}
	if payload.Networks == nil || len(payload.Networks) != 0 {
		t.Fatalf("networks = %#v, want non-nil empty slice", payload.Networks)
	}
}

func TestPrivacyNetworkResponseNormalizesOptionalData(t *testing.T) {
	response := toPrivacyNetworkResponse(privacy.StatusSnapshot{
		ID:       "i2p",
		Mode:     "external",
		Enabled:  true,
		StartErr: "provider unavailable",
	})

	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error != "provider unavailable" {
		t.Fatalf("error = %q, want provider error", response.Error)
	}
	if response.Metrics == nil {
		t.Fatal("metrics = nil, want neutral metrics for a partial provider failure")
	}
	if response.Metrics.TxBytes != 0 || response.Metrics.RxBytes != 0 || response.Metrics.Connections != 0 {
		t.Fatalf("metrics = %#v, want neutral counters for a partial provider failure", response.Metrics)
	}
	if response.Addresses == nil || len(response.Addresses) != 0 {
		t.Fatalf("addresses = %#v, want non-nil empty slice", response.Addresses)
	}
}

func TestPrivacyNetworkResponseIncludesMetricsForStartedProvider(t *testing.T) {
	response := toPrivacyNetworkResponse(privacy.StatusSnapshot{
		ID:          "yggdrasil",
		Mode:        "native",
		Enabled:     true,
		Started:     true,
		Addresses:   []string{"200::1"},
		TxBytes:     12,
		RxBytes:     34,
		Connections: 2,
	})

	if response.Metrics == nil {
		t.Fatal("metrics = nil, want metrics for started provider")
	}
	if response.Metrics.TxBytes != 12 || response.Metrics.RxBytes != 34 || response.Metrics.Connections != 2 {
		t.Fatalf("metrics = %#v, want provider metrics", response.Metrics)
	}
	if response.Metrics.Peers != nil {
		t.Fatalf("metrics.peers = %#v, want nil when provider has no peer count", response.Metrics.Peers)
	}
}
