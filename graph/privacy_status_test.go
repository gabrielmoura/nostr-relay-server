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
	Started   bool            `json:"started"`
	Status    string          `json:"status"`
	Addresses []string        `json:"addresses"`
	Metrics   *privacyMetrics `json:"metrics"`
	Error     *string         `json:"error"`
}

type privacyMetrics struct {
	TxBytes     int  `json:"txBytes"`
	RxBytes     int  `json:"rxBytes"`
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
