package metrics

import (
	"errors"

	"github.com/gabrielmoura/nostr-relay-server/infra/net/privacy"
	"github.com/prometheus/client_golang/prometheus"
)

// PrivacyStatusProvider supplies a read-only privacy status snapshot at scrape
// time. The privacy domain remains independent of Prometheus.
type PrivacyStatusProvider interface {
	Status() privacy.ManagerStatus
}

// privacyCollector translates the privacy domain's scalar snapshots into
// Prometheus metrics. It does not alter service lifecycle or network state.
type privacyCollector struct {
	provider PrivacyStatusProvider

	networkUp             *prometheus.Desc
	networkUptime         *prometheus.Desc
	networkAddressPresent *prometheus.Desc
	networkStartFailures  *prometheus.Desc
	yggPeers              *prometheus.Desc
	yggPeersInbound       *prometheus.Desc
	yggSessions           *prometheus.Desc
	yggSessionRxBytes     *prometheus.Desc
	yggSessionTxBytes     *prometheus.Desc
	yggPeerRxBytesPerSec  *prometheus.Desc
	yggPeerTxBytesPerSec  *prometheus.Desc
	yggRoutingEntries     *prometheus.Desc
	yggTreeEntries        *prometheus.Desc
	yggPathEntries        *prometheus.Desc
	yggMTUBytes           *prometheus.Desc
}

// NewPrivacyCollector creates a scrape-time collector for privacy-network
// lifecycle facts and Yggdrasil's public aggregate diagnostic values.
func NewPrivacyCollector(provider PrivacyStatusProvider) prometheus.Collector {
	return &privacyCollector{
		provider: provider,
		networkUp: prometheus.NewDesc(
			"nostr_privacy_network_up",
			"Whether the privacy network service has started successfully.",
			[]string{"network", "mode"},
			nil,
		),
		networkUptime: prometheus.NewDesc(
			"nostr_privacy_network_uptime_seconds",
			"Seconds since the current successful privacy network service start.",
			[]string{"network", "mode"},
			nil,
		),
		networkAddressPresent: prometheus.NewDesc(
			"nostr_privacy_network_address_configured",
			"Whether the privacy network service has an advertised address.",
			[]string{"network", "mode"},
			nil,
		),
		networkStartFailures: prometheus.NewDesc(
			"nostr_privacy_network_start_failures_total",
			"Total failed privacy network service start attempts.",
			[]string{"network", "mode"},
			nil,
		),
		yggPeers: prometheus.NewDesc(
			"nostr_privacy_yggdrasil_peers",
			"Current Yggdrasil peers by connection state.",
			[]string{"state"},
			nil,
		),
		yggPeersInbound: prometheus.NewDesc(
			"nostr_privacy_yggdrasil_peers_inbound",
			"Current inbound Yggdrasil peers.",
			nil,
			nil,
		),
		yggSessions: prometheus.NewDesc(
			"nostr_privacy_yggdrasil_sessions",
			"Current Yggdrasil sessions.",
			nil,
			nil,
		),
		yggSessionRxBytes: prometheus.NewDesc(
			"nostr_privacy_yggdrasil_session_rx_bytes",
			"Current aggregate Yggdrasil session received bytes.",
			nil,
			nil,
		),
		yggSessionTxBytes: prometheus.NewDesc(
			"nostr_privacy_yggdrasil_session_tx_bytes",
			"Current aggregate Yggdrasil session transmitted bytes.",
			nil,
			nil,
		),
		yggPeerRxBytesPerSec: prometheus.NewDesc(
			"nostr_privacy_yggdrasil_peer_rx_bytes_per_second",
			"Current aggregate Yggdrasil peer receive rate in bytes per second.",
			nil,
			nil,
		),
		yggPeerTxBytesPerSec: prometheus.NewDesc(
			"nostr_privacy_yggdrasil_peer_tx_bytes_per_second",
			"Current aggregate Yggdrasil peer transmit rate in bytes per second.",
			nil,
			nil,
		),
		yggRoutingEntries: prometheus.NewDesc(
			"nostr_privacy_yggdrasil_routing_entries",
			"Current Yggdrasil routing table entries.",
			nil,
			nil,
		),
		yggTreeEntries: prometheus.NewDesc(
			"nostr_privacy_yggdrasil_tree_entries",
			"Current Yggdrasil spanning tree entries.",
			nil,
			nil,
		),
		yggPathEntries: prometheus.NewDesc(
			"nostr_privacy_yggdrasil_path_entries",
			"Current Yggdrasil path entries.",
			nil,
			nil,
		),
		yggMTUBytes: prometheus.NewDesc(
			"nostr_privacy_yggdrasil_mtu_bytes",
			"Configured Yggdrasil MTU in bytes.",
			nil,
			nil,
		),
	}
}

// Describe sends all descriptors, allowing registration-time descriptor
// validation and helping registry collision detection.
func (c *privacyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.networkUp
	ch <- c.networkUptime
	ch <- c.networkAddressPresent
	ch <- c.networkStartFailures
	ch <- c.yggPeers
	ch <- c.yggPeersInbound
	ch <- c.yggSessions
	ch <- c.yggSessionRxBytes
	ch <- c.yggSessionTxBytes
	ch <- c.yggPeerRxBytesPerSec
	ch <- c.yggPeerTxBytesPerSec
	ch <- c.yggRoutingEntries
	ch <- c.yggTreeEntries
	ch <- c.yggPathEntries
	ch <- c.yggMTUBytes
}

// Collect exports a point-in-time snapshot without starting services, dialing
// peers, or querying daemon-specific administration APIs.
func (c *privacyCollector) Collect(ch chan<- prometheus.Metric) {
	if c.provider == nil {
		return
	}

	for _, network := range c.provider.Status().Networks {
		labels := []string{network.ID, network.Mode}
		ch <- prometheus.MustNewConstMetric(c.networkUp, prometheus.GaugeValue, boolFloat(network.Started), labels...)
		ch <- prometheus.MustNewConstMetric(c.networkUptime, prometheus.GaugeValue, network.Uptime.Seconds(), labels...)
		ch <- prometheus.MustNewConstMetric(c.networkAddressPresent, prometheus.GaugeValue, boolFloat(len(network.Addresses) > 0), labels...)
		ch <- prometheus.MustNewConstMetric(c.networkStartFailures, prometheus.CounterValue, float64(network.StartFailures), labels...)

		if network.Yggdrasil != nil {
			c.collectYggdrasil(ch, network.Yggdrasil)
		}
	}
}

func (c *privacyCollector) collectYggdrasil(ch chan<- prometheus.Metric, status *privacy.YggdrasilStatusSnapshot) {
	ch <- prometheus.MustNewConstMetric(c.yggPeers, prometheus.GaugeValue, float64(status.PeersUp), "up")
	ch <- prometheus.MustNewConstMetric(c.yggPeers, prometheus.GaugeValue, float64(status.PeersDown), "down")
	ch <- prometheus.MustNewConstMetric(c.yggPeersInbound, prometheus.GaugeValue, float64(status.PeersInbound))
	ch <- prometheus.MustNewConstMetric(c.yggSessions, prometheus.GaugeValue, float64(status.Sessions))
	ch <- prometheus.MustNewConstMetric(c.yggSessionRxBytes, prometheus.GaugeValue, float64(status.SessionRxBytes))
	ch <- prometheus.MustNewConstMetric(c.yggSessionTxBytes, prometheus.GaugeValue, float64(status.SessionTxBytes))
	ch <- prometheus.MustNewConstMetric(c.yggPeerRxBytesPerSec, prometheus.GaugeValue, float64(status.PeerRxBytesPerSec))
	ch <- prometheus.MustNewConstMetric(c.yggPeerTxBytesPerSec, prometheus.GaugeValue, float64(status.PeerTxBytesPerSec))
	ch <- prometheus.MustNewConstMetric(c.yggRoutingEntries, prometheus.GaugeValue, float64(status.RoutingEntries))
	ch <- prometheus.MustNewConstMetric(c.yggTreeEntries, prometheus.GaugeValue, float64(status.TreeEntries))
	ch <- prometheus.MustNewConstMetric(c.yggPathEntries, prometheus.GaugeValue, float64(status.PathEntries))
	ch <- prometheus.MustNewConstMetric(c.yggMTUBytes, prometheus.GaugeValue, float64(status.MTUBytes))
}

func boolFloat(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

// RegisterPrivacyMetrics registers a privacy collector with the supplied
// registry. Supplying the registerer keeps application bootstrap and tests free
// from global-registry collisions.
func RegisterPrivacyMetrics(registerer prometheus.Registerer, provider PrivacyStatusProvider) error {
	err := registerer.Register(NewPrivacyCollector(provider))
	var alreadyRegistered prometheus.AlreadyRegisteredError
	if errors.As(err, &alreadyRegistered) {
		return nil
	}
	return err
}
