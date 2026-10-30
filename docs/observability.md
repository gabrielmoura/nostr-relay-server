# Observability

The relay uses Prometheus for aggregate operational signals, structured logs for
individual failures, and profiling to prove a CPU, memory, lock, or goroutine
problem. Do not treat development-machine PostgreSQL/Redis latency or an
external relay's duration as intrinsic relay latency.

## Nostr messages and subscriptions

| Metric | Meaning | Labels |
|---|---|---|
| `nostr_request_count` | Decoded, supported client messages handled. | `type` (protocol message type) |
| `nostr_request_processing_duration_seconds` | Synchronous work from dispatch to handler return. For `REQ`, it ends after the initial query/EOSE and subscription registration. | `type` |
| `nostr_events_accepted_total` | EVENT messages accepted into the bounded ingestion queue after synchronous validation. | none |
| `nostr_connection_count` | Open WebSocket connections on this instance. | none |
| `nostr_listeners_active` | Active Nostr subscriptions on this instance. | none |
| `nostr_listener_add_total`, `nostr_listener_remove_total` | Subscription lifecycle transitions. | none |

A long-lived `REQ` subscription is expected. Its lifetime is **not** request
latency and is deliberately not exported as a histogram until there is a
specific operational use for it.

Useful queries:

```promql
sum(rate(nostr_request_count{type="EVENT"}[5m]))
```

```promql
histogram_quantile(0.95, sum by (le) (
  rate(nostr_request_processing_duration_seconds_bucket{type="REQ"}[5m])
))
```

## Ingestion, storage, and cache

- `nostr_relay_batch_processed_total` counts batches that complete without a
  storage error. `nostr_relay_events_inserted_total` counts rows actually
  inserted, excluding ephemeral events and `ON CONFLICT` duplicates.
- `nostr_relay_ingestion_duplicates_total`,
  `nostr_relay_ingestion_rejected_total`,
  `nostr_relay_ingestion_errors_total`, and
  `nostr_relay_ingestion_backpressure_total` distinguish duplicate work,
  policy rejection, storage failure, and a full bounded queue.
  `nostr_relay_ingestion_queue_depth` is the number of events currently in the
  channel; it excludes events that workers have already removed into a batch.
  `nostr_relay_ingestion_duration_seconds{outcome="success|error"}` measures
  the complete local batch pipeline, including validation, persistence, and
  post-persistence work.
- `nostr_db_pool_acquired`, `nostr_db_pool_idle`, and `nostr_db_pool_total`
  are pgx pool snapshots collected at scrape time. Use
  `nostr_db_pool_empty_acquires_total` and
  `nostr_db_pool_empty_acquire_wait_seconds_total` to investigate sustained
  pool contention before changing pool size.
- `nostr_redis_query_cache_total{result="hit|miss"}` supports a hit ratio:

```promql
sum(rate(nostr_redis_query_cache_total{result="hit"}[5m]))
/
sum(rate(nostr_redis_query_cache_total[5m]))
```

A low hit ratio is an investigation prompt, not automatically a reason to
redesign the cache.

`nostr_redis_pool_total_connections`, `nostr_redis_pool_idle_connections`,
`nostr_redis_pool_pending_requests`, `nostr_redis_pool_timeouts_total`, and
`nostr_redis_pool_wait_seconds_total` describe client-pool contention. They
are separate from cache effectiveness and contain no key or command labels.

Useful ingestion queries:

```promql
histogram_quantile(0.95, sum by (le) (
  rate(nostr_relay_ingestion_duration_seconds_bucket{outcome="success"}[5m])
))
```

```promql
rate(nostr_relay_events_inserted_total[5m])
```

## Queues and external relays

`nostr_queue_job_latency_seconds` is dispatch-to-start wait time;
`nostr_queue_job_duration_seconds` is execution time. `nostr_queue_depth`,
`nostr_queue_dead_jobs`, `nostr_queue_jobs_failed_total`, and
`nostr_queue_reclaims_total` indicate accumulation and recovery behaviour.

Relay downloads use `nostr_download_*`; their `relay="external"` label is
intentionally bounded. `nostr_download_page_latency_seconds` measures remote
relay/network duration with buckets appropriate for long operations. It must
not be compared to internal request processing latency. Inspect the structured
logs for the actual relay URL and error.

## Profiling and alerts

The pprof server is compiled only with `-tags=diagnostics` and additionally
requires `app_env: development`. In that explicit developer build it starts on
`http://127.0.0.1:6060/debug/pprof/`; standard builds and every non-development
environment, including production, leave it disabled. It binds only to loopback
and stops with the relay. Capture a profile locally with, for example,
`go tool pprof http://127.0.0.1:6060/debug/pprof/heap`.

Use protected, on-demand Go pprof (`goroutine`, `heap`, `allocs`, `mutex`, and
`block`) only after a sustained symptom: growing goroutine count, memory
growth, lock contention, or latency under representative load. Do not expose
profiling endpoints on a public relay.

Race detection is a development/CI build mode, not a runtime setting: execute
`make dev-race` to produce `bin/nrserver-dev-race`, which also carries the
diagnostics build tag. A running relay cannot enable `-race` dynamically; never
use this artifact as a production release.

Prefer conservative alerts for persistent ingestion errors, growing queue/dead
jobs, unavailable PostgreSQL/Redis, or sustained growth in goroutines/memory.
Avoid alerts based only on an external relay taking a long time.
