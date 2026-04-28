# Redis Queue and Worker Architecture

## Goal

Refactor operational background work to use one Redis-backed queue subsystem that is durable, observable, low-overhead and compatible with the current Cobra + Redis + Prometheus + package-level bootstrap style already used by the project.

The first migration targets are:

1. admin download jobs
2. admin negentropy sync jobs
3. cron-triggered maintenance and fetch routines

Current implementation status:

- queue-backed download jobs are implemented behind `jobs.enabled` + `redis.queue.enabled`
- the repository already contains a dedicated `worker` Cobra command for queue-only execution
- admin sync is queue-backed when queue mode is enabled
- cron scheduler mode dispatches queue jobs when queue mode is enabled; `--run-once` remains synchronous for compatibility

The live relay ingestion queue in `infra/ingestion` remains a separate in-memory hot-path pipeline for now. It is latency-sensitive and should not be rewritten together with operational jobs.

## Current Architecture Summary

### What exists today

- `internal/down/jobs.go` stores admin download jobs in memory and starts them with `go runJob(...)`
- `internal/sync/sync.go` runs sync inline for CLI and as fire-and-forget goroutines for admin HTTP
- `cmd/internal/cron/run.go` runs cron jobs directly in process through `robfig/cron`
- Redis is already used for cache, pub/sub and listener coordination, but not for durable job execution

### Main gaps

- no durable job persistence
- no ACK/reclaim/retry/dead-letter semantics
- no cross-instance worker coordination
- no compact job status tracking
- no unified metrics or operational visibility
- no graceful way to pause, requeue or inspect failed work

## Design Principles

- preserve current command and admin route behavior whenever possible
- keep handlers and cron adapters thin; job logic stays outside Redis code
- use Redis Streams for delivery, Sorted Sets for delayed/retry/dead scheduling, Strings for payloads, compact Hashes only for small metadata, and BITFIELD for dense status/attempt state
- introduce interfaces only at the consumption boundaries
- keep package-level wiring compatible with current startup style, but isolate new queue code behind constructors so later DI cleanup stays possible

## Proposed Package Structure

```text
cmd/
  worker.go                      # optional dedicated worker process
  internal/worker/               # Cobra adapter/runtime wiring only

internal/jobs/
  contracts.go                   # Job, Dispatcher, Registry, Monitor contracts
  envelope.go                    # compact serialized envelope
  options.go                     # dispatch/worker/retry options
  registry.go                    # handler registration
  scheduler.go                   # backoff + jitter policies
  service.go                     # application-facing orchestration
  types.go                       # JobID, Queue, Status

infra/queue/
  redis/
    client.go                    # queue-specific Redis wrapper over infra/redis
    dispatcher.go                # Dispatch/DispatchIn/DispatchAt
    worker.go                    # consumer-group worker loop
    promoter.go                  # delayed -> stream promotion
    reclaimer.go                 # pending reclaim and retry/dead routing
    tracker.go                   # compact status/metadata reads
    keys.go                      # Redis key builder with hash tags
    scripts.go                   # SHA loading and execution
    metrics.go                   # queue-specific Prometheus metrics helpers
    lua/
      enqueue.lua
      start.lua
      ack_success.lua
      retry.lua
      move_dead.lua
      promote_delayed.lua
      record_cancel.lua
      metrics_rollup.lua
```

## Main Interfaces

```go
type Job interface {
	Name() string
}

type Handler[T Job] interface {
	Handle(ctx context.Context, job T) error
}

type Dispatcher interface {
	Dispatch(ctx context.Context, job Job, opts ...DispatchOption) (JobID, error)
	DispatchIn(ctx context.Context, job Job, delay time.Duration, opts ...DispatchOption) (JobID, error)
	DispatchAt(ctx context.Context, job Job, runAt time.Time, opts ...DispatchOption) (JobID, error)
}

type Monitor interface {
	Get(ctx context.Context, queue string, id JobID) (Snapshot, error)
	List(ctx context.Context, queue string, filter ListFilter) ([]Snapshot, error)
	Retry(ctx context.Context, queue string, id JobID) error
	Cancel(ctx context.Context, queue string, id JobID) error
}

type Registry interface {
	Register(name string, decoder Decoder, handler HandlerFunc)
	Lookup(name string) (RegisteredHandler, bool)
}
```

Notes:

- constructors return concrete structs; callers depend on small interfaces
- generics are only used on typed handler registration, not on the Redis execution core
- handlers do not know about Redis, Streams, or Lua

## Queue Model

### Redis structures

| Need | Structure | Notes |
|---|---|---|
| worker distribution | Stream + Consumer Group | one stream per queue/priority |
| delayed jobs | ZSET | score = run timestamp in ms |
| retry schedule | ZSET | same structure as delayed |
| dead-letter storage | ZSET + compact meta | ordered by dead timestamp |
| payload | STRING | serialized envelope/body |
| compact state | BITFIELD | 3 bits per job id |
| attempts | BITFIELD | 8 bits per job id |
| started timestamp | HASH or STRING only when needed | avoid per-job large structures |
| per-job tiny metadata | HASH | only short fields (`j`, `q`, `p`, `a`, `ma`, `ca`, `la`, `sa`, `fa`, `ra`, `e`) |
| active workers approximation | HyperLogLog | windowed unique workers |
| per-minute metrics rollup | HASH | TTL-based minute buckets |
| recent job ordering | ZSET | ordered by creation timestamp |
| terminal result snapshot | STRING | optional structured result for detail endpoints |

### Key scheme

All queue keys for the same logical queue share the same hash tag to keep Lua multi-key operations cluster-safe.

```text
rq:{queue}:seq
rq:{queue}:stream:{priority}
rq:{queue}:delayed
rq:{queue}:dead
rq:{queue}:jobs
rq:{queue}:body:{job_id}
rq:{queue}:result:{job_id}
rq:{queue}:state
rq:{queue}:attempts
rq:{queue}:meta:{job_id}
rq:{queue}:metrics:{yyyyMMddHHmm}
rq:{queue}:workers:{yyyyMMddHHmm}
rq:{queue}:unique:{fingerprint}
```

### Compact state codes

| Code | Status |
|---:|---|
| 0 | unknown |
| 1 | queued |
| 2 | running |
| 3 | succeeded |
| 4 | failed |
| 5 | delayed |
| 6 | dead |
| 7 | canceled |

## Execution Model

### Dispatch

1. caller builds typed job payload
2. dispatcher serializes compact envelope
3. `enqueue.lua` allocates sequential job id
4. payload is stored in `rq:{queue}:body:{job_id}`
5. state and attempts bitfields are initialized atomically
6. job goes either to stream immediately or to delayed ZSET
7. per-minute metrics hash is incremented in the same script

### Worker loop

1. `XREADGROUP` blocks on one or more priority streams
2. worker loads body string and compact metadata
3. `start.lua` marks running and increments attempts atomically
4. registry resolves handler by job name
5. handler executes under `context.WithTimeout`
6. on success, `ack_success.lua` performs `XACK`, `XDEL`, state update and metric aggregation
7. on failure, worker computes retry policy and calls `retry.lua` or `move_dead.lua`

### Delayed promotion

- one lightweight promoter loop runs per queue group
- `promote_delayed.lua` moves due ids from delayed ZSET into the main stream in bounded batches
- promotion must be idempotent and safe under multiple worker instances

### Reclaim

- workers periodically inspect the Pending Entries List with idle thresholds
- stale messages are claimed with `XAUTOCLAIM`
- reclaim count is exported as a metric and dead-lettered when attempts exceed limits

## Retry and Dead Letter Rules

- default retry policy: bounded exponential backoff with jitter
- job-level `max_attempts`, `timeout`, `queue`, `priority`, `unique_for` and `trace` options stay in dispatch metadata
- terminal failures write compact error summary to `meta.e`
- dead jobs go to `rq:{queue}:dead` with score = dead timestamp in ms
- reprocessing dead jobs reuses the same payload but creates a fresh stream entry

## Metrics and Observability

Required Prometheus metrics:

- `nostr_queue_jobs_enqueued_total{queue,job_type,priority}`
- `nostr_queue_jobs_started_total{queue,job_type,worker}`
- `nostr_queue_jobs_succeeded_total{queue,job_type}`
- `nostr_queue_jobs_failed_total{queue,job_type,status}`
- `nostr_queue_jobs_retried_total{queue,job_type}`
- `nostr_queue_jobs_dead_total{queue,job_type}`
- `nostr_queue_job_duration_seconds{queue,job_type}`
- `nostr_queue_job_latency_seconds{queue,job_type}`
- `nostr_queue_depth{queue,priority}`
- `nostr_queue_delayed_jobs{queue}`
- `nostr_queue_dead_jobs{queue}`
- `nostr_queue_active_workers{queue}`
- `nostr_queue_redis_errors_total{queue,operation}`
- `nostr_queue_lua_errors_total{queue,script}`
- `nostr_queue_reclaims_total{queue}`

Rules:

- never label with `job_id`
- `worker` label should use stable worker names, not random per-message values
- duration/latency are histograms, not summaries

## CLI and Runtime Integration

### Server mode

- `server` may optionally start embedded workers/promoters for queues needed by the admin HTTP surface
- worker count stays config-driven and defaults to zero when queue mode is disabled

### Cron mode

- `cron` remains the scheduling entry point
- instead of executing business logic inline, each cron tick dispatches a queue job when queue mode is enabled
- `--run-once` may keep a synchronous compatibility path during migration, then optionally dispatch-and-wait later

### Dedicated worker mode

- add an optional `worker` Cobra command for queue-only deployments
- this supports horizontal scale without running full HTTP/WebSocket server instances

Current status:

- implemented as `nrserver worker`
- currently registers the download queue handler and starts the same embedded runtime used by `server`

## Planned Job Types

First-class job names:

- `download.events`
- `sync.negentropy`
- `cron.db_optimization`
- `cron.reported_events_fetch`
- `cron.delete_old_events`
- `cron.nip40`

Future candidates:

- `wot.recompute`
- NIP-29 maintenance jobs
- batch operational reindex/rebuild jobs

## Incremental Refactor Plan

### Stage 1

- add queue contracts, Redis key builder, Lua loader, metrics and tracker
- no public behavior change yet
- keep download/sync/cron implementations untouched

### Stage 2

- migrate admin download jobs to the queue
- preserve existing admin response shape and list/detail endpoints
- keep CLI `download` synchronous for now by calling the existing runtime directly

Status:

- implemented

### Stage 3

- migrate admin sync to a tracked queue job
- keep current `status: started` response and add `job_id`
- add status endpoint before changing frontend behavior

Status:

- implemented for dispatch path
- admin sync now returns `job_id` when queue mode is enabled

### Stage 4

- wrap current cron routines as job handlers
- keep `nrserver cron` as the scheduling entry point
- when enabled, cron dispatches queue jobs instead of executing inline

Status:

- implemented for scheduler mode
- `--run-once` intentionally keeps direct execution for compatibility

### Stage 5

- add dedicated `worker` command
- optionally move selected server-managed background work to queue execution where that improves durability

## Compatibility Rules

- existing admin routes remain valid
- existing command names remain valid
- existing download result semantics remain valid: only all-relay failure marks the job as failed
- Redis-disabled environments must fail clearly for queue-backed operations instead of silently dropping work
- queue internals stay hidden behind Go interfaces; handlers and HTTP/CLI code must not manipulate Streams directly

## Out of Scope for the First Iteration

- replacing `infra/ingestion` hot-path event buffering with Redis Streams
- a Horizon-like UI beyond the minimal APIs needed for existing admin screens
- distributed locks beyond what Consumer Groups, idempotency keys and Lua atomicity already provide
- storing large job payload JSON in Hashes
