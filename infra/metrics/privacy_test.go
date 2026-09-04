package metrics

import (
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/net/privacy"
	"github.com/prometheus/client_golang/prometheus"
	clientmodel "github.com/prometheus/client_model/go"
)

type staticPrivacyStatusProvider struct {
	status privacy.ManagerStatus
}

func (p staticPrivacyStatusProvider) Status() privacy.ManagerStatus {
	return p.status
}

func TestPrivacyCollectorCollectsLifecycleMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	provider := staticPrivacyStatusProvider{status: privacy.ManagerStatus{
		Networks: []privacy.StatusSnapshot{{
			ID:            "tor",
			Mode:          "native",
			Started:       true,
			Addresses:     []string{"example.onion"},
			Uptime:        3 * time.Minute,
			StartFailures: 2,
		}},
	}}

	if err := RegisterPrivacyMetrics(registry, provider); err != nil {
		t.Fatalf("RegisterPrivacyMetrics: %v", err)
	}

	families := gatherMetricFamilies(t, registry)
	assertMetricValue(t, families, "nostr_privacy_network_up", 1, map[string]string{"network": "tor", "mode": "native"})
	assertMetricValue(t, families, "nostr_privacy_network_uptime_seconds", 180, map[string]string{"network": "tor", "mode": "native"})
	assertMetricValue(t, families, "nostr_privacy_network_address_configured", 1, map[string]string{"network": "tor", "mode": "native"})
	assertMetricValue(t, families, "nostr_privacy_network_start_failures_total", 2, map[string]string{"network": "tor", "mode": "native"})
}

func TestPrivacyCollectorCollectsYggdrasilMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	provider := staticPrivacyStatusProvider{status: privacy.ManagerStatus{
		Networks: []privacy.StatusSnapshot{{
			ID:      "yggdrasil",
			Mode:    "native",
			Started: true,
			Yggdrasil: &privacy.YggdrasilStatusSnapshot{
				PeersUp:           3,
				PeersDown:         1,
				PeersInbound:      2,
				Sessions:          4,
				SessionRxBytes:    100,
				SessionTxBytes:    200,
				PeerRxBytesPerSec: 30,
				PeerTxBytesPerSec: 40,
				RoutingEntries:    5,
				TreeEntries:       6,
				PathEntries:       7,
				MTUBytes:          65535,
			},
		}},
	}}

	if err := RegisterPrivacyMetrics(registry, provider); err != nil {
		t.Fatalf("RegisterPrivacyMetrics: %v", err)
	}

	families := gatherMetricFamilies(t, registry)
	assertMetricValue(t, families, "nostr_privacy_yggdrasil_peers", 3, map[string]string{"state": "up"})
	assertMetricValue(t, families, "nostr_privacy_yggdrasil_peers", 1, map[string]string{"state": "down"})
	assertMetricValue(t, families, "nostr_privacy_yggdrasil_peers_inbound", 2, nil)
	assertMetricValue(t, families, "nostr_privacy_yggdrasil_sessions", 4, nil)
	assertMetricValue(t, families, "nostr_privacy_yggdrasil_session_rx_bytes", 100, nil)
	assertMetricValue(t, families, "nostr_privacy_yggdrasil_session_tx_bytes", 200, nil)
	assertMetricValue(t, families, "nostr_privacy_yggdrasil_peer_rx_bytes_per_second", 30, nil)
	assertMetricValue(t, families, "nostr_privacy_yggdrasil_peer_tx_bytes_per_second", 40, nil)
	assertMetricValue(t, families, "nostr_privacy_yggdrasil_routing_entries", 5, nil)
	assertMetricValue(t, families, "nostr_privacy_yggdrasil_tree_entries", 6, nil)
	assertMetricValue(t, families, "nostr_privacy_yggdrasil_path_entries", 7, nil)
	assertMetricValue(t, families, "nostr_privacy_yggdrasil_mtu_bytes", 65535, nil)
}

func TestRegisterPrivacyMetricsIgnoresAlreadyRegisteredCollector(t *testing.T) {
	registry := prometheus.NewRegistry()
	provider := staticPrivacyStatusProvider{}

	if err := RegisterPrivacyMetrics(registry, provider); err != nil {
		t.Fatalf("first RegisterPrivacyMetrics: %v", err)
	}
	if err := RegisterPrivacyMetrics(registry, provider); err != nil {
		t.Fatalf("second RegisterPrivacyMetrics: %v", err)
	}
}

func gatherMetricFamilies(t *testing.T, registry *prometheus.Registry) map[string]*clientmodel.MetricFamily {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	result := make(map[string]*clientmodel.MetricFamily, len(families))
	for _, family := range families {
		result[family.GetName()] = family
	}
	return result
}

func assertMetricValue(t *testing.T, families map[string]*clientmodel.MetricFamily, name string, want float64, labels map[string]string) {
	t.Helper()
	family, ok := families[name]
	if !ok {
		t.Fatalf("metric family %q was not collected", name)
	}
	for _, metric := range family.Metric {
		if !metricLabelsMatch(metric, labels) {
			continue
		}
		if got := metricValue(metric); got != want {
			t.Fatalf("metric %q value = %v, want %v", name, got, want)
		}
		return
	}
	t.Fatalf("metric %q with labels %v was not collected", name, labels)
}

func metricLabelsMatch(metric *clientmodel.Metric, want map[string]string) bool {
	if len(metric.Label) != len(want) {
		return false
	}
	for _, label := range metric.Label {
		if want[label.GetName()] != label.GetValue() {
			return false
		}
	}
	return true
}

func metricValue(metric *clientmodel.Metric) float64 {
	if metric.Gauge != nil {
		return metric.Gauge.GetValue()
	}
	return metric.Counter.GetValue()
}
