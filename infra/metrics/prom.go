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
	NostrDownloadEventsReceivedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_download_events_received_total",
			Help: "Total number of events received by the download command per relay.",
		},
		[]string{"relay"},
	)
	NostrDownloadEventsPersistedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_download_events_persisted_total",
			Help: "Total number of events persisted by the download command per relay.",
		},
		[]string{"relay"},
	)
	NostrDownloadDuplicatesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_download_duplicates_total",
			Help: "Total number of duplicate events detected by the download command per relay.",
		},
		[]string{"relay"},
	)
	NostrDownloadFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_download_failures_total",
			Help: "Total number of download command failures per relay.",
		},
		[]string{"relay"},
	)
	NostrDownloadPageLatencySeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nostr_download_page_latency_seconds",
			Help:    "Latency per paginated download request by relay.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"relay"},
	)

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
	// NostrNegentropyCounter - Contador de mensagens de Negentropia
	NostrNegentropyCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_negentropy_count",
			Help: "No of Negentropy messages handled by Nostr handler",
		},
		[]string{"type"},
	)
	NostrNegentropyV2RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_negentropy_v2_requests_total",
			Help: "Total Negentropy V2 requests handled by operation and result.",
		},
		[]string{"operation", "result"},
	)
	NostrNegentropyV2CacheTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_negentropy_v2_cache_total",
			Help: "Negentropy V2 cache operations by backend and result.",
		},
		[]string{"backend", "result"},
	)
	NostrNegentropyV2SessionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nostr_negentropy_v2_sessions_active",
			Help: "Current number of active Negentropy V2 sessions.",
		},
	)
	NostrNegentropyV2ProtocolErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_negentropy_v2_protocol_errors_total",
			Help: "Total Negentropy V2 protocol errors returned to clients.",
		},
	)
	NostrNegentropyV2EventsImportedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_negentropy_v2_events_imported_total",
			Help: "Total events imported through NEG-HAVE during Negentropy synchronization.",
		},
	)
	// NostrUserAgentCounter - Contador de User-Agent
	NostrUserAgentCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_user_agent_count",
			Help: "No of User-Agent handled by Nostr handler",
		},
		[]string{"user_agent"},
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
	// NostrRelayRequestForwardedTotal - Número de requisições encaminhadas para outros relays.
	NostrRelayRequestForwardedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_request_forwarded_total",
			Help: "Number of requests forwarded to other relays",
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
	// NostrRelayEventDeletionFailures - Total de falhas ao tentar deletar eventos.
	NostrRelayEventDeletionFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_event_deletion_failures",
			Help: "Total number of failures when trying to delete events.",
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

	// Batch ingestion metrics
	NostrRelayBatchProcessed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_batch_processed_total",
			Help: "Total number of batches processed by ingestion workers.",
		},
	)
	NostrRelayEventsInserted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_events_inserted_total",
			Help: "Total number of events inserted via batch processing.",
		},
	)
	NostrRelayIngestionDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nostr_relay_ingestion_duration_seconds",
			Help:    "Duration of batch insertion in seconds.",
			Buckets: prometheus.DefBuckets,
		},
	)
	NostrRelayIngestionBackpressure = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_ingestion_backpressure_total",
			Help: "Total number of times the ingestion queue was full (backpressure).",
		},
	)
	NostrRelayIngestionDuplicates = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_ingestion_duplicates_total",
			Help: "Total number of duplicate events rejected by deduplication.",
		},
	)
	NostrRelayIngestionErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_relay_ingestion_errors_total",
			Help: "Total number of errors during batch insertion.",
		},
	)
	NostrRedisQueryCacheResult = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_redis_query_cache_total",
			Help: "Redis query cache results by outcome.",
		},
		[]string{"result"},
	)
	NostrDBPoolAcquired = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nostr_db_pool_acquired",
			Help: "Number of currently acquired database connections.",
		},
	)
	NostrDBPoolIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nostr_db_pool_idle",
			Help: "Number of currently idle database connections.",
		},
	)
	NostrDBPoolTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nostr_db_pool_total",
			Help: "Total number of database connections in the pool.",
		},
	)
	NostrDBPoolAcquireCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_db_pool_acquire_total",
			Help: "Total successful database pool acquires.",
		},
	)
	NostrListenerOrphanCleanup = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_listener_orphan_cleanup_total",
			Help: "Total orphan listener subscriptions removed from Redis.",
		},
	)
	NostrCronNIP40RunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_cron_nip40_runs_total",
			Help: "Total NIP-40 cron runs by result.",
		},
		[]string{"result"},
	)
	NostrCronNIP40DeletedEventsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nostr_cron_nip40_deleted_events_total",
			Help: "Total events deleted by NIP-40 expiration cron.",
		},
	)
	NostrCronNIP40DurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nostr_cron_nip40_duration_seconds",
			Help:    "Duration of NIP-40 expiration cron runs in seconds.",
			Buckets: prometheus.DefBuckets,
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
		NostrDownloadEventsReceivedTotal,
		NostrDownloadEventsPersistedTotal,
		NostrDownloadDuplicatesTotal,
		NostrDownloadFailuresTotal,
		NostrDownloadPageLatencySeconds,
		HttpDuration,
		NostrRelayAuthFailuresTotal,
		NostrRelayEventSignatureFailures,
		NostrRelayEventDuplicateRejections,
		NostrRelayEventForwardedTotal,
		NostrRelayEventForwardedFailuresTotal,
		NostrRelayRequestForwardedTotal,
		NostrRelayEventDeletionSuccessful,
		NostrRelayEventDeletionFailures,
		// Listener metrics
		NostrListenerGauge,
		NostrListenerAddCounter,
		NostrListenerRemoveCounter,
		NostrEventsNotifiedCounter,
		// Negentropy metrics
		NostrNegentropyCounter,
		NostrNegentropyV2RequestsTotal,
		NostrNegentropyV2CacheTotal,
		NostrNegentropyV2SessionsActive,
		NostrNegentropyV2ProtocolErrorsTotal,
		NostrNegentropyV2EventsImportedTotal,
		NostrUserAgentCounter,
		// Ingestion metrics
		NostrRelayBatchProcessed,
		NostrRelayEventsInserted,
		NostrRelayIngestionDuration,
		NostrRelayIngestionBackpressure,
		NostrRelayIngestionDuplicates,
		NostrRelayIngestionErrors,
		NostrRedisQueryCacheResult,
		NostrDBPoolAcquired,
		NostrDBPoolIdle,
		NostrDBPoolTotal,
		NostrDBPoolAcquireCount,
		NostrListenerOrphanCleanup,
		NostrCronNIP40RunsTotal,
		NostrCronNIP40DeletedEventsTotal,
		NostrCronNIP40DurationSeconds,
	)

}
