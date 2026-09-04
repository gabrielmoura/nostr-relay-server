package graph

import (
	"bytes"
	stdjson "encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/net/privacy"
)

func TestPrivacyStatusReturnsDisabledDataWhenManagerIsNil(t *testing.T) {
	previous := privacy.GetManager()
	privacy.SetManager(nil)
	t.Cleanup(func() { privacy.SetManager(previous) })

	response := executePrivacyStatusQuery(t)
	assertPrivacyStatusResponse(t, response, false)
}

func TestPrivacyStatusReturnsEmptyNetworksForDisabledManager(t *testing.T) {
	previous := privacy.GetManager()
	privacy.SetManager(privacy.NewManager(config.PrivacyConfig{Enabled: false}, nil))
	t.Cleanup(func() { privacy.SetManager(previous) })

	response := executePrivacyStatusQuery(t)
	assertPrivacyStatusResponse(t, response, false)
}

func TestPrivacyStatusReturnsProviderFailureWithoutGraphQLError(t *testing.T) {
	previous := privacy.GetManager()
	manager := privacy.NewManager(config.PrivacyConfig{
		Enabled:     true,
		Persistence: true,
		I2P:         config.I2PConfig{Mode: "unsupported"},
	}, nil)
	if err := manager.Start(t.Context(), 0); err != nil {
		t.Fatalf("start privacy manager: %v", err)
	}
	privacy.SetManager(manager)
	t.Cleanup(func() {
		manager.Close()
		privacy.SetManager(previous)
	})

	response := executePrivacyStatusQuery(t)
	if len(response.Errors) != 0 {
		t.Fatalf("GraphQL errors = %s, want none", response.Errors)
	}
	if response.Data.PrivacyStatus.StateDir == nil || *response.Data.PrivacyStatus.StateDir != "" {
		t.Fatalf("privacyStatus.stateDir = %#v, want empty string for missing state dir", response.Data.PrivacyStatus.StateDir)
	}
	if len(response.Data.PrivacyStatus.Networks) != 1 {
		t.Fatalf("privacyStatus.networks = %#v, want one failed provider", response.Data.PrivacyStatus.Networks)
	}

	network := response.Data.PrivacyStatus.Networks[0]
	if network.Status != "error" || network.Started {
		t.Fatalf("provider status = %q, started = %t; want error and false", network.Status, network.Started)
	}
	if len(network.Addresses) != 0 {
		t.Fatalf("provider addresses = %#v, want []", network.Addresses)
	}
	if network.Metrics == nil || network.Metrics.TxBytes != 0 || network.Metrics.RxBytes != 0 || network.Metrics.Connections == nil || *network.Metrics.Connections != 0 {
		t.Fatalf("provider metrics = %#v, want neutral metrics", network.Metrics)
	}
	if network.Error == nil || *network.Error == "" {
		t.Fatal("provider error = nil or empty, want the provider-local start failure")
	}
}

func TestPrivacyStatusReturnsPopulatedAndDegradedNetworksWithoutGraphQLError(t *testing.T) {
	previous := privacy.GetManager()
	stateDir := t.TempDir()
	manager := privacy.NewManager(config.PrivacyConfig{
		Enabled:     true,
		Persistence: true,
		StateDir:    stateDir,
		I2P:         config.I2PConfig{Mode: "unsupported"},
		Ygg:         config.YggConfig{Mode: "native"},
	}, nil)
	if err := manager.Start(t.Context(), 0); err != nil {
		t.Fatalf("start privacy manager: %v", err)
	}
	privacy.SetManager(manager)
	t.Cleanup(func() {
		manager.Close()
		privacy.SetManager(previous)
	})

	response := executePrivacyStatusQuery(t)
	if len(response.Errors) != 0 {
		t.Fatalf("GraphQL errors = %s, want none", response.Errors)
	}
	if !response.Data.PrivacyStatus.Enabled || !response.Data.PrivacyStatus.Persistence {
		t.Fatalf("privacyStatus = %#v, want enabled persistent status", response.Data.PrivacyStatus)
	}
	if response.Data.PrivacyStatus.StateDir == nil || *response.Data.PrivacyStatus.StateDir != stateDir {
		t.Fatalf("privacyStatus.stateDir = %#v, want %q", response.Data.PrivacyStatus.StateDir, stateDir)
	}
	if len(response.Data.PrivacyStatus.Networks) != 2 {
		t.Fatalf("privacyStatus.networks = %#v, want populated and degraded networks", response.Data.PrivacyStatus.Networks)
	}

	degraded := response.Data.PrivacyStatus.Networks[0]
	if degraded.ID != "i2p" || degraded.Name != "I2P" || degraded.Mode != "unsupported" || !degraded.Enabled || degraded.Started || degraded.Status != "error" {
		t.Fatalf("degraded network = %#v, want enabled i2p error snapshot", degraded)
	}
	if len(degraded.Addresses) != 0 {
		t.Fatalf("degraded addresses = %#v, want []", degraded.Addresses)
	}
	if degraded.Metrics == nil || degraded.Metrics.TxBytes != 0 || degraded.Metrics.RxBytes != 0 || degraded.Metrics.Peers != nil || degraded.Metrics.Connections == nil || *degraded.Metrics.Connections != 0 {
		t.Fatalf("degraded metrics = %#v, want neutral metrics with nullable peers", degraded.Metrics)
	}
	if degraded.Error == nil || *degraded.Error == "" || degraded.UptimeMs != 0 {
		t.Fatalf("degraded error = %#v, uptimeMs = %d; want provider error and zero uptime", degraded.Error, degraded.UptimeMs)
	}

	populated := response.Data.PrivacyStatus.Networks[1]
	if populated.ID != "yggdrasil" || populated.Name != "Yggdrasil" || populated.Mode != "native" || !populated.Enabled || !populated.Started || populated.Status != "operational" {
		t.Fatalf("populated network = %#v, want enabled operational yggdrasil snapshot", populated)
	}
	if len(populated.Addresses) != 1 || populated.Addresses[0] == "" {
		t.Fatalf("populated addresses = %#v, want one yggdrasil address", populated.Addresses)
	}
	if populated.Metrics == nil || populated.Metrics.TxBytes != 0 || populated.Metrics.RxBytes != 0 || populated.Metrics.Peers == nil || *populated.Metrics.Peers != 0 || populated.Metrics.Connections == nil || *populated.Metrics.Connections != 0 {
		t.Fatalf("populated metrics = %#v, want neutral metrics with a reported peer count", populated.Metrics)
	}
	if populated.Error != nil || populated.UptimeMs < 0 {
		t.Fatalf("populated error = %#v, uptimeMs = %d; want nil error and non-negative uptime", populated.Error, populated.UptimeMs)
	}
}

func executePrivacyStatusQuery(t *testing.T) graphResponse {
	t.Helper()

	body, err := stdjson.Marshal(map[string]string{
		"query": `{ privacyStatus { enabled persistence stateDir networks { id name mode enabled started status addresses metrics { txBytes rxBytes peers connections } error uptimeMs } } }`,
	})
	if err != nil {
		t.Fatalf("marshal GraphQL request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	HTTPHandler().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GraphQL response status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response graphResponse
	if err := stdjson.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode GraphQL response: %v", err)
	}
	return response
}

type graphResponse struct {
	Data struct {
		PrivacyStatus struct {
			Enabled     bool             `json:"enabled"`
			Persistence bool             `json:"persistence"`
			StateDir    *string          `json:"stateDir"`
			Networks    []privacyNetwork `json:"networks"`
		} `json:"privacyStatus"`
	} `json:"data"`
	Errors []stdjson.RawMessage `json:"errors"`
}

type privacyNetwork struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Mode      string          `json:"mode"`
	Enabled   bool            `json:"enabled"`
	Started   bool            `json:"started"`
	Status    string          `json:"status"`
	Addresses []string        `json:"addresses"`
	Metrics   *privacyMetrics `json:"metrics"`
	Error     *string         `json:"error"`
	UptimeMs  int             `json:"uptimeMs"`
}

type privacyMetrics struct {
	TxBytes     int  `json:"txBytes"`
	RxBytes     int  `json:"rxBytes"`
	Peers       *int `json:"peers"`
	Connections *int `json:"connections"`
}

func assertPrivacyStatusResponse(t *testing.T, response graphResponse, wantEnabled bool) {
	t.Helper()
	if len(response.Errors) != 0 {
		t.Fatalf("GraphQL errors = %s, want none", response.Errors)
	}
	if response.Data.PrivacyStatus.Enabled != wantEnabled {
		t.Fatalf("privacyStatus.enabled = %t, want %t", response.Data.PrivacyStatus.Enabled, wantEnabled)
	}
	if response.Data.PrivacyStatus.Persistence {
		t.Fatal("privacyStatus.persistence = true, want false")
	}
	if len(response.Data.PrivacyStatus.Networks) != 0 {
		t.Fatalf("privacyStatus.networks = %#v, want []", response.Data.PrivacyStatus.Networks)
	}
}
