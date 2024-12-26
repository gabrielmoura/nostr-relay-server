package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var NostrRequestCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "nostr_request_count",
		Help: "No of request handled by Nostr handler",
	},
	[]string{"method"},
)

var NostrRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "nostr_request_duration",
		Help:    "Duration of request handled by Nostr handler",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method"},
)

var NostrConnectionCounter = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "nostr_connection_count",
		Help: "No of connection handled by Nostr handler",
	},
)

// NostrKindReqCounter - Qual o consumo de dados por kind?
var NostrKindReqCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "nostr_kind_request_count",
		Help: "No of request handled by Nostr handler",
	},
	[]string{"kind"},
)

// NostrKindEventCounter - Qual é o kind mais popular?
var NostrKindEventCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "nostr_kind_event_count",
		Help: "No of request handled by Nostr handler",
	},
	[]string{"kind"},
)

// NostrUserReqCounter - Qual o consumo de dados por usuário?
var NostrUserReqCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "nostr_user_request_count",
		Help: "No of Request per User",
	},
	[]string{"user"},
)

// NostrUserEventCounter - Qual o usuário mais ativo?
var NostrUserEventCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "nostr_user_event_count",
		Help: "No of Event per User",
	},
	[]string{"user"},
)

// NostrTagReqCounter - Qual o consumo de dados por tag?
var NostrTagReqCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "nostr_tag_request_count",
		Help: "No of Tags per REQUEST",
	},
	[]string{"tag"},
)

// NostrTagEventCounter - Qual a tag mais popular?
var NostrTagEventCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "nostr_tag_event_count",
		Help: "No of Tags per Event",
	},
	[]string{"tag"},
)

var (
	UploadCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "uploads",
		Help: "The total number of uploads",
	})
	DownloadCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "downloads",
		Help: "The total number of files fetched",
	})

	HttpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_response_duration_seconds",
		Help: "Latency of requests in second.",
	}, []string{"path"})
)

func RegisterMetrics() {
	prometheus.MustRegister(NostrRequestCounter)
	prometheus.MustRegister(NostrRequestDuration)
	prometheus.MustRegister(NostrConnectionCounter)
	prometheus.MustRegister(NostrKindReqCounter)
	prometheus.MustRegister(NostrKindEventCounter)
	prometheus.MustRegister(NostrUserReqCounter)
	prometheus.MustRegister(NostrTagReqCounter)
	prometheus.MustRegister(NostrTagEventCounter)
	prometheus.MustRegister(NostrUserEventCounter)
	prometheus.MustRegister(UploadCounter)
	prometheus.MustRegister(DownloadCounter)
	prometheus.MustRegister(HttpDuration)

}
