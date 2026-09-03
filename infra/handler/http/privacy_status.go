package http

import (
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/infra/net/privacy"
	"github.com/gofiber/fiber/v2"
)

// PrivacyStatusResponse mirrors the GraphQL PrivacyStatus contract so the
// resolver can decode it 1:1 via decodeRESTModel. Field JSON keys match the
// gqlgen model struct tags.
type PrivacyStatusResponse struct {
	Enabled     bool                     `json:"enabled"`
	Persistence bool                     `json:"persistence"`
	StateDir    string                   `json:"state_dir,omitempty"`
	Networks    []PrivacyNetworkResponse `json:"networks"`
}

type PrivacyNetworkResponse struct {
	ID        string                  `json:"id"`
	Name      string                  `json:"name"`
	Mode      string                  `json:"mode"`
	Enabled   bool                    `json:"enabled"`
	Started   bool                    `json:"started"`
	Status    string                  `json:"status"`
	Addresses []string                `json:"addresses"`
	Metrics   *PrivacyMetricsResponse `json:"metrics,omitempty"`
	Error     string                  `json:"error,omitempty"`
	UptimeMs  int64                   `json:"uptime_ms"`
}

type PrivacyMetricsResponse struct {
	TxBytes     int64 `json:"tx_bytes"`
	RxBytes     int64 `json:"rx_bytes"`
	Peers       *int  `json:"peers,omitempty"`
	Connections int   `json:"connections"`
}

// PrivacyStatus is the admin HTTP handler for /privacy/status. It returns the
// aggregated privacy-network observability snapshot. If privacy is disabled or
// the manager is nil it returns a safe empty payload (never an error), so the
// dashboard renders a "disabled" state instead of breaking.
func PrivacyStatus() fiber.Handler {
	return func(c *fiber.Ctx) error {
		st := privacy.GetManager().Status() // nil-safe; returns zero flags
		resp := PrivacyStatusResponse{
			Enabled:     st.Enabled,
			Persistence: st.Persistence,
			StateDir:    st.StateDir,
			Networks:    make([]PrivacyNetworkResponse, 0, len(st.Networks)),
		}
		for _, n := range st.Networks {
			resp.Networks = append(resp.Networks, toPrivacyNetworkResponse(n))
		}
		return c.JSON(resp)
	}
}

func toPrivacyNetworkResponse(n privacy.StatusSnapshot) PrivacyNetworkResponse {
	status := "operational"
	switch {
	case !n.Enabled:
		status = "disabled"
	case !n.Started && n.StartErr != "":
		status = "error"
	case !n.Started:
		status = "disabled"
	case n.StartErr != "":
		status = "error"
	}

	var metrics *PrivacyMetricsResponse
	if n.Started {
		metrics = &PrivacyMetricsResponse{
			TxBytes:     n.TxBytes,
			RxBytes:     n.RxBytes,
			Peers:       n.Peers,
			Connections: n.Connections,
		}
	}

	return PrivacyNetworkResponse{
		ID:        n.ID,
		Name:      networkName(n.ID),
		Mode:      n.Mode,
		Enabled:   n.Enabled,
		Started:   n.Started,
		Status:    status,
		Addresses: n.Addresses,
		Metrics:   metrics,
		Error:     n.StartErr,
		UptimeMs:  n.Uptime.Milliseconds(),
	}
}

// networkName maps a privacy network id to a human-readable label.
func networkName(id string) string {
	switch strings.ToLower(id) {
	case "tor":
		return "Tor"
	case "i2p":
		return "I2P"
	case "yggdrasil":
		return "Yggdrasil"
	default:
		return id
	}
}
