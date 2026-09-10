package ws

import (
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/prometheus/client_golang/prometheus"
	client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestHandleMessageMeasuresSupportedSynchronousProcessing(t *testing.T) {
	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(metrics.NostrRequestCounter))
	require.NoError(t, registry.Register(metrics.NostrRequestProcessingDuration))

	messageTypes := []string{dto.TypeREQ, dto.TypeEVENT, dto.TypeCLOSE}
	originalHandlers := make(map[string]messageHandler, len(messageTypes))
	for _, messageType := range messageTypes {
		originalHandlers[messageType] = wsMessageHandlers[messageType]
		wsMessageHandlers[messageType] = func(*dto.WsServer, dto.Data) string {
			time.Sleep(5 * time.Millisecond)
			return ""
		}
	}
	t.Cleanup(func() {
		for messageType, handler := range originalHandlers {
			wsMessageHandlers[messageType] = handler
		}
	})

	ws := &dto.WsServer{ChanSender: make(chan any, len(messageTypes))}
	for _, messageType := range messageTypes {
		handleMessage(ws, []byte(`["`+messageType+`","subscription"]`))
	}

	families, err := registry.Gather()
	require.NoError(t, err)
	processing := metricFamilyByName(t, families, "nostr_request_processing_duration_seconds")
	require.Len(t, processing.Metric, len(messageTypes))
	for _, metric := range processing.Metric {
		require.Equal(t, uint64(1), metric.Histogram.GetSampleCount())
		require.GreaterOrEqual(t, metric.Histogram.GetSampleSum(), 0.005)
	}

	requests := metricFamilyByName(t, families, "nostr_request_count")
	require.Len(t, requests.Metric, len(messageTypes))
}

func metricFamilyByName(t *testing.T, families []*client.MetricFamily, name string) *client.MetricFamily {
	t.Helper()
	for _, family := range families {
		if family.GetName() == name {
			return family
		}
	}
	t.Fatalf("metric family %q not found", name)
	return nil
}
