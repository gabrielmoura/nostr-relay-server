package http

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestAdminIngestionMetricsFromCollectors(t *testing.T) {
	newCounter := func(value float64) prometheus.Counter {
		counter := prometheus.NewCounter(prometheus.CounterOpts{Name: "test_admin_overview_counter"})
		counter.Add(value)
		return counter
	}
	queueDepth := prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_admin_overview_queue_depth"})
	queueDepth.Set(9)

	got := adminIngestionMetricsFromCollectors(
		newCounter(101),
		newCounter(80),
		newCounter(12),
		newCounter(3),
		newCounter(2),
		newCounter(1),
		newCounter(14),
		newCounter(4),
		queueDepth,
	)

	require.Equal(t, adminIngestionMetricsResponse{
		AcceptedEvents: 101,
		InsertedEvents: 80,
		Duplicates:     12,
		Rejected:       3,
		Uncertain:      2,
		BatchErrors:    1,
		Batches:        14,
		Backpressure:   4,
		QueueDepth:     9,
	}, got)
}
