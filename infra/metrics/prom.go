package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	UploadCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nostr_uploads",
		Help: "The total number of uploads",
	})
	DownloadCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nostr_downloads",
		Help: "The total number of files fetched",
	})

	HttpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_response_duration_seconds",
		Help: "Latency of requests in second.",
	}, []string{"path"})

	NostrRequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_request_count",
			Help: "No of request handled by Nostr handler",
		},
		[]string{"method"},
	)

	NostrRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nostr_request_duration",
			Help:    "Duration of request handled by Nostr handler",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	NostrConnectionCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nostr_connection_count",
			Help: "No of connection handled by Nostr handler",
		},
	)

	// NostrKindReqCounter - Qual o consumo de dados por kind?
	NostrKindReqCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_kind_request_count",
			Help: "No of request handled by Nostr handler",
		},
		[]string{"kind"},
	)

	// NostrKindEventCounter - Qual é o kind mais popular?
	NostrKindEventCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_kind_event_count",
			Help: "No of request handled by Nostr handler",
		},
		[]string{"kind"},
	)

	// NostrUserReqCounter - Qual o consumo de dados por usuário?
	NostrUserReqCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_user_request_count",
			Help: "No of Request per User",
		},
		[]string{"user"},
	)

	// NostrUserEventCounter - Qual o usuário mais ativo?
	NostrUserEventCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_user_event_count",
			Help: "No of Event per User",
		},
		[]string{"user"},
	)

	// NostrTagReqCounter - Qual o consumo de dados por tag?
	NostrTagReqCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_tag_request_count",
			Help: "No of Tags per REQUEST",
		},
		[]string{"tag"},
	)

	// NostrTagEventCounter - Qual a tag mais popular?
	NostrTagEventCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_tag_event_count",
			Help: "No of Tags per Event",
		},
		[]string{"tag"},
	)
	// NostrRelayAuthFailuresTotal - Total de falhas de autenticação ou assinatura inválida.
	NostrRelayAuthFailuresTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_auth_failures_total",
			Help: "total authentication failures or invalid signature.",
		})

	// NostrRelayWsMessagesReceived - Total de mensagens WebSocket recebidas pelo relay
	NostrRelayWsMessagesReceived = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_ws_messages_received",
			Help: "Total WebSocket messages received by the relay",
		})
	// NostrRelayWsMessagesSend - Total de mensagens WebSocket enviadas pelo relay.
	NostrRelayWsMessagesSend = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_ws_messages_sent",
			Help: "Total WebSocket messages sent by the relay.",
		})

	// NostrRelayEventSignatureFailures - Total de eventos com assinatura inválida.
	NostrRelayEventSignatureFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_event_signature_failures",
			Help: "Total number of events with invalid signature.",
		})
	// NostrRelayEventDuplicateRejections - Número de eventos rejeitados por serem duplicados.
	NostrRelayEventDuplicateRejections = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_event_duplicate_rejections",
			Help: "number of events rejected because they are duplicates.",
		})
	// NostrRelayEventForwardedTotal - Número de eventos encaminhados para outros relays
	NostrRelayEventForwardedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_event_forwarded_total",
			Help: "Number of events forwarded to other relays",
		})
	// NostrRelayEventForwardedFailuresTotal - Número de falhas ao tentar encaminhar eventos para outros relays.
	NostrRelayEventForwardedFailuresTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_event_forward_failures_total",
			Help: "Number of failures when trying to forward events to other relays.",
		})
	// NostrRelayEventDeletionSuccessful - Total de eventos deletados com sucesso.
	NostrRelayEventDeletionSuccessful = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_event_deletions_successful",
			Help: "Total events deleted successfully.",
		})
	// NostrListenerGauge - Número de listeners ativos no servidor Nostr.
	NostrListenerGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nostr_listeners_active",
			Help: "Number of active listeners on the Nostr server.",
		},
	)
	// NostrListenerAddCounter - Contador de listeners adicionados.
	NostrListenerAddCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_listener_add_total",
			Help: "Total listeners added.",
		},
	)
	// NostrListenerRemoveCounter - Contador de listeners removidos.
	NostrListenerRemoveCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_listener_remove_total",
			Help: "Total listeners removed.",
		},
	)
	// NostrEventsNotifiedCounter - Contador de eventos notificados aos listeners.
	NostrEventsNotifiedCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_events_notified_total",
			Help: "Total number of events notified to listeners.",
		},
	)
)

func RegisterMetrics() {
	prometheus.MustRegister(NostrRequestCounter,
		NostrRequestDuration,
		NostrConnectionCounter,
		NostrKindReqCounter,
		NostrKindEventCounter,
		NostrUserReqCounter,
		NostrTagReqCounter,
		NostrTagEventCounter,
		NostrUserEventCounter,
		UploadCounter,
		DownloadCounter,
		HttpDuration,
		NostrRelayAuthFailuresTotal,
		NostrRelayEventSignatureFailures,
		NostrRelayEventDuplicateRejections,
		NostrRelayEventForwardedTotal,
		NostrRelayEventForwardedFailuresTotal,
		NostrRelayEventDeletionSuccessful,
		// Listener metrics
		NostrListenerGauge,
		NostrListenerAddCounter,
		NostrListenerRemoveCounter,
		NostrEventsNotifiedCounter,
	)

}
