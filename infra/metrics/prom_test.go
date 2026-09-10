package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestNostrRequestProcessingMetricContract(t *testing.T) {
	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(NostrRequestProcessingDuration))

	NostrRequestProcessingDuration.WithLabelValues("REQ").Observe(0.01)
	family := metricFamily(t, registry, "nostr_request_processing_duration_seconds")
	require.Equal(t, client.MetricType_HISTOGRAM, family.GetType())
	require.Len(t, family.Metric, 1)
	require.Equal(t, "type", family.Metric[0].Label[0].GetName())
	require.Equal(t, "REQ", family.Metric[0].Label[0].GetValue())
	require.Contains(t, family.GetHelp(), "subscription registration")
}

func TestExternalRelayMetricsUseControlledLabel(t *testing.T) {
	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(NostrDownloadPageLatencySeconds))

	NostrDownloadPageLatencySeconds.WithLabelValues(ExternalRelayLabel).Observe(200)
	family := metricFamily(t, registry, "nostr_download_page_latency_seconds")
	require.Equal(t, ExternalRelayLabel, family.Metric[0].Label[0].GetValue())
	require.Greater(t, family.Metric[0].Histogram.GetBucket()[len(family.Metric[0].Histogram.GetBucket())-1].GetUpperBound(), 300.0)
}

func TestMetricLabelsRemainBounded(t *testing.T) {
	tests := []struct {
		name      string
		desc      string
		wantLabel string
	}{
		{"request", collectorDescription(NostrRequestCounter), "variableLabels: {type}"},
		{"request duration", collectorDescription(NostrRequestProcessingDuration), "variableLabels: {type}"},
		{"request kind", collectorDescription(NostrKindReqCounter), "variableLabels: {kind}"},
		{"accepted events", collectorDescription(NostrEventsAcceptedTotal), "variableLabels: {}"},
		{"external download", collectorDescription(NostrDownloadFailuresTotal), "variableLabels: {relay}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Contains(t, tt.desc, tt.wantLabel)
			for _, forbidden := range []string{"user", "pubkey", "user_agent", "tag"} {
				require.NotContains(t, tt.desc, forbidden)
			}
		})
	}
}

func TestExternalAndQueueLatencyMetricsHaveDistinctDomains(t *testing.T) {
	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(NostrDownloadPageLatencySeconds))
	require.NoError(t, registry.Register(NostrQueueJobLatencySeconds))
	require.NoError(t, registry.Register(NostrQueueJobDurationSeconds))

	NostrDownloadPageLatencySeconds.WithLabelValues(ExternalRelayLabel).Observe(200)
	NostrQueueJobLatencySeconds.WithLabelValues("jobs", "deliver").Observe(0.1)
	NostrQueueJobDurationSeconds.WithLabelValues("jobs", "deliver").Observe(0.1)

	external := metricFamily(t, registry, "nostr_download_page_latency_seconds")
	queueLatency := metricFamily(t, registry, "nostr_queue_job_latency_seconds")
	queueDuration := metricFamily(t, registry, "nostr_queue_job_duration_seconds")
	require.Contains(t, external.GetHelp(), "End-to-end duration")
	require.Contains(t, external.GetHelp(), "local event processing")
	require.Contains(t, queueLatency.GetHelp(), "Dispatch-to-start")
	require.Contains(t, queueDuration.GetHelp(), "execution duration")
	require.NotEqual(t, strings.Join(labelNames(external), ","), strings.Join(labelNames(queueLatency), ","))
}

func TestIngestionMetricsDistinguishOutcomeAndQueueDepth(t *testing.T) {
	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(NostrRelayIngestionDuration))
	require.NoError(t, registry.Register(NostrRelayIngestionQueueDepth))

	NostrRelayIngestionDuration.WithLabelValues("success").Observe(0.1)
	NostrRelayIngestionDuration.WithLabelValues("error").Observe(0.2)
	NostrRelayIngestionQueueDepth.Set(7)

	duration := metricFamily(t, registry, "nostr_relay_ingestion_duration_seconds")
	require.Len(t, duration.Metric, 2)
	require.Contains(t, collectorDescription(NostrRelayIngestionDuration), "variableLabels: {outcome}")
	require.Greater(t, duration.Metric[0].Histogram.GetBucket()[len(duration.Metric[0].Histogram.GetBucket())-1].GetUpperBound(), 10.0)

	depth := metricFamily(t, registry, "nostr_relay_ingestion_queue_depth")
	require.Equal(t, 7.0, depth.Metric[0].Gauge.GetValue())
}

func labelNames(family *client.MetricFamily) []string {
	labels := make([]string, 0, len(family.Metric[0].Label))
	for _, label := range family.Metric[0].Label {
		labels = append(labels, label.GetName())
	}
	return labels
}

func collectorDescription(collector prometheus.Collector) string {
	descriptions := make(chan *prometheus.Desc, 1)
	collector.Describe(descriptions)
	return (<-descriptions).String()
}

func metricFamily(t *testing.T, registry *prometheus.Registry, name string) *client.MetricFamily {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == name {
			return family
		}
	}
	t.Fatalf("metric family %q not found", name)
	return nil
}
