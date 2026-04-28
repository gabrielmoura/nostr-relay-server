package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	NostrQueueJobsEnqueuedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_queue_jobs_enqueued_total",
			Help: "Total queued jobs by queue, job type and priority.",
		},
		[]string{"queue", "job_type", "priority"},
	)
	NostrQueueJobsStartedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_queue_jobs_started_total",
			Help: "Total started jobs by queue, job type and worker.",
		},
		[]string{"queue", "job_type", "worker"},
	)
	NostrQueueJobsSucceededTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_queue_jobs_succeeded_total",
			Help: "Total succeeded jobs by queue and job type.",
		},
		[]string{"queue", "job_type"},
	)
	NostrQueueJobsFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_queue_jobs_failed_total",
			Help: "Total failed jobs by queue, job type and terminal status.",
		},
		[]string{"queue", "job_type", "status"},
	)
	NostrQueueJobsRetriedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_queue_jobs_retried_total",
			Help: "Total retried jobs by queue and job type.",
		},
		[]string{"queue", "job_type"},
	)
	NostrQueueJobsDeadTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_queue_jobs_dead_total",
			Help: "Total dead-lettered jobs by queue and job type.",
		},
		[]string{"queue", "job_type"},
	)
	NostrQueueJobDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nostr_queue_job_duration_seconds",
			Help:    "Job execution duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"queue", "job_type"},
	)
	NostrQueueJobLatencySeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nostr_queue_job_latency_seconds",
			Help:    "Dispatch-to-start latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"queue", "job_type"},
	)
	NostrQueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nostr_queue_depth",
			Help: "Approximate queue depth per queue and priority stream.",
		},
		[]string{"queue", "priority"},
	)
	NostrQueueDelayedJobs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nostr_queue_delayed_jobs",
			Help: "Current delayed jobs per queue.",
		},
		[]string{"queue"},
	)
	NostrQueueDeadJobs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nostr_queue_dead_jobs",
			Help: "Current dead jobs per queue.",
		},
		[]string{"queue"},
	)
	NostrQueueActiveWorkers = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nostr_queue_active_workers",
			Help: "Approximate active workers per queue.",
		},
		[]string{"queue"},
	)
	NostrQueueRedisErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_queue_redis_errors_total",
			Help: "Redis errors in queue operations by queue and operation.",
		},
		[]string{"queue", "operation"},
	)
	NostrQueueLuaErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_queue_lua_errors_total",
			Help: "Lua script execution errors by queue and script.",
		},
		[]string{"queue", "script"},
	)
	NostrQueueReclaimsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_queue_reclaims_total",
			Help: "Total reclaimed pending jobs by queue.",
		},
		[]string{"queue"},
	)
)
