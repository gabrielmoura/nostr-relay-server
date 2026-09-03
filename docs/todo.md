# Implementation TODO

## Phase 1: Project Setup ✅

- [x] Initialize Go module
- [x] Set up directory structure
- [x] Configure Cobra CLI
- [x] Set up Viper configuration

## Phase 2: Database Layer ✅

- [x] Create PostgreSQL schema (event, profiles, banned_users, objects)
- [x] Implement pgx connection pool
- [x] Create Queries wrapper
- [x] Implement event CRUD operations
- [x] Implement profile CRUD operations
- [x] Implement object metadata storage

## Phase 3: Configuration ✅

- [x] Define Config struct
- [x] Load configuration from YAML
- [x] Set default values
- [x] Configuration validation

## Phase 4: HTTP Server ✅

- [x] Set up Fiber app
- [x] Configure CORS middleware
- [x] Configure compression middleware
- [x] Set up internal server (metrics)
- [x] Set up external server (relay)

## Phase 5: WebSocket Handler ✅

- [x] WebSocket upgrade handling
- [x] NIP-11 (Accept header for info)
- [x] Ping/Pong keep-alive
- [x] Message parsing
- [x] Type dispatch (EVENT, REQ, CLOSE, AUTH, COUNT)

## Phase 6: Event Handling ✅

- [x] Event ID validation
- [x] Event signature verification
- [x] Event storage (insert/replace)
- [x] Replaceable event handling (NIP-16)
- [x] Parameterized replaceable (NIP-33)
- [x] Ephemeral event handling (NIP-16)

## Phase 7: Request Handling ✅

- [x] Filter parsing
- [x] Dynamic SQL query building
- [x] Subscription management
- [x] Event notification to subscribers
- [x] EOSE (End of Stored Events)
- [x] COUNT support (NIP-45)

## Phase 8: Policies ✅

- [x] Ban user check
- [x] Event expiration (NIP-40)
- [x] POW check (NIP-13)
- [x] Tag size/length validation
- [x] Base64 media prevention
- [x] Authentication policy (NIP-42)
- [x] Empty filter policy

## Phase 9: NIP Implementations ✅

- [x] NIP-01: Basic protocol
- [x] NIP-09: Event deletion
- [x] NIP-11: Relay info document
- [x] NIP-42: Authentication
- [x] NIP-45: Event counts
- [x] NIP-50: Search capability
- [x] NIP-62: Request to vanish
- [x] NIP-96: Blossom storage
- [x] NIP-98: HTTP auth
- [x] NIP-77: Kind 30078 (bookmarks)

## Phase 10: Blossom Storage ✅

- [x] Upload endpoint (POST /upload)
- [x] Download endpoint (GET /blob/:id)
- [x] Head check (HEAD /blob/:id)
- [x] List endpoint (GET /list/:id)
- [x] MIME type validation
- [x] SHA256 hash verification
- [x] File metadata storage
- [x] NIP-96 server config

## Phase 11: Metrics ✅

- [x] Request counters by type
- [x] Event counters by kind
- [x] User activity counters
- [x] Connection gauges
- [x] Listener gauges
- [x] Upload/download counters
- [x] Forwarding counters
- [x] Prometheus endpoint

## Phase 12: Relay Pool ✅

- [x] Singleton pool management
- [x] Concurrent publish to relays
- [x] Connection retry logic
- [x] Event forwarding configuration

## Phase 13: Negentropy ✅

- [x] Session manager
- [x] NEG-OPEN handling
- [x] NEG-MSG handling
- [x] NEG-HAVE handling
- [x] NEG-NEED handling
- [x] NEG-CLOSE handling
- [x] Vector reconciliation

## Phase 14: CLI Commands ✅

- [x] `server` - Main server
- [x] `cron` - Scheduled tasks
- [x] `import` - JSONL import
- [x] `export` - JSONL export
- [x] `sync` - Negentropy sync
- [x] `seed` - Database seeding
- [x] `conf` - Config management

## Phase 15: Bootstrap ✅

- [x] Generate relay bot keypair
- [x] Create Kind 0 (profile metadata)
- [x] Create Kind 411 (relay info)
- [x] Create Kind 10002 (relay list)
- [x] Create Kind 10063 (NIP-98 auth)

## Phase 16: Cron Jobs ✅

- [x] Delete old events
- [x] Configurable retention
- [x] Pool statistics update

## Phase 17: Documentation

- [x] README with features
- [x] Architecture diagram
- [x] API specification
- [x] Configuration reference
- [ ] Deployment guide
- [ ] Troubleshooting guide

## Phase 40: Admin GraphQL Migration

### 39.1: Docs and schema
- [x] Inventory current `/admin/*` REST routes and payloads
- [x] Validate relevant Nostr protocol semantics for admin operations (NIP-19, NIP-32, NIP-56, NIP-86)
- [x] Document GraphQL target contract in `docs/graphql-admin-schema.graphqls`
- [x] Document migration rules and REST-to-GraphQL mapping in `docs/graphql-admin.md`
- [x] Add ADR for internal admin GraphQL adoption
- [ ] Get explicit documentation approval before touching `graph/` or router wiring

### 39.2: Planned implementation after approval
- [ ] Replace sample `graph/schema.graphqls` todo schema with approved admin SDL
- [ ] Add gqlgen scalar mappings for `Time`, `Int64`, `JSON`, and `Upload`
- [ ] Mount `/admin/graphql` on the internal Fiber app behind `AdminTokenMiddleware`
- [ ] Implement resolver composition over existing admin services and query packages
- [ ] Add resolver tests for normalization, auth, pagination, async job payloads, and Blossom/NIP-86 operations
- [ ] Define REST deprecation path for the admin SPA migration

## Phase 38: Marmot MIP-00 Relay Support

### 38.1: Docs and design
- [x] Document Marmot `MIP-00` scope and relay responsibilities
- [x] Add ADR for optional `marmot.mip00` module
- [x] Define config surface and validation modes
- [x] Define phase 1 compatibility boundary: relay-aware, not MLS-complete

### 38.2: Planned implementation
- [x] Add `marmot` and `mip00` config structs and defaults
- [x] Validate contradictory or unsupported `marmot.mip00` settings during config load
- [x] Add policy validator for `kind:30443`
- [x] Add policy validator for `kind:10051`
- [x] Gate optional legacy `kind:443` support behind config
- [x] Add deterministic rejection reasons for Marmot validation failures
- [x] Add explicit `marmot_mip00` Prometheus counters for accepted and rejected relevant events

### 38.3: Planned verification
- [x] Add tests for disabled mode preserving generic relay behavior
- [x] Add tests for valid `kind:30443` and `kind:10051` events in `basic` mode
- [x] Add tests for required-tag rejection paths
- [x] Add tests for invalid relay URL rejection
- [x] Add tests proving existing addressable replacement semantics cover `kind:30443`

### 38.4: Deferred kinds outside current phase
- [ ] Decide whether `444` and `445` will stay generic relay kinds or gain explicit Marmot module treatment
- [ ] Decide whether `447`, `448`, `449`, and `10050` should become part of a future MIP-05 module instead of `mip00`

## Phase 34: Cron Consolidation

- [x] Refactor `cron` command to configuration-driven scheduler
- [x] Add DB optimization routine with enable/schedule controls
- [x] Add automatic NIP-56 reported events fetch with explicit relay list
- [x] Add retention cleanup by `older_than_days` with batched deletion

## Phase 35: Redis Queue and Worker Refactor

### 35.1: Docs and design
- [x] Document current background-work architecture and gaps
- [x] Write Redis queue/worker design doc
- [x] Add ADR for Redis Streams operational queue
- [x] Define planned Redis key schema and config surface

### 35.2: Queue core
- [x] Add `internal/jobs` contracts and registry
- [x] Add `infra/queue/redis` package with key builder and script loader
- [x] Embed and preload Lua scripts via SHA
- [x] Add queue-specific Prometheus metrics

### 35.3: Worker runtime
- [x] Implement Redis Streams dispatcher
- [x] Implement concurrent worker loop with graceful shutdown
- [x] Implement delayed promoter and pending reclaim flow
- [x] Implement retry/backoff/jitter and dead-letter routing

### 35.4: Incremental migrations
- [x] Migrate admin download jobs from in-memory store to queue-backed execution
- [x] Migrate admin sync from fire-and-forget goroutine to tracked queue job
- [x] Wrap cron routines as queue handlers while preserving `nrserver cron`
- [x] Add optional dedicated `worker` Cobra command

### 35.5: Observability and validation
- [ ] Add integration tests for dispatch, retry, delayed and dead-letter flows
- [ ] Add compatibility tests for existing admin endpoints
- [ ] Add metrics assertions for queue depth, retries and worker activity
- [ ] Document rollout and fallback strategy

### 35.6: Sync queue hardening and operator controls
- [x] Add `negentropy_auth` config and bind Negentropy auth to `relay_information.pub_key`
- [x] Gate websocket `NEG-*` messages with NIP-42 when `negentropy_auth=true`
- [x] Teach sync CLI to answer remote `AUTH` challenges with `relay_information.priv_key`
- [ ] Add terminal cancel semantics for `sync.negentropy`
- [ ] Add explicit resume action for canceled sync jobs
- [ ] Add backend history cleanup endpoint for dashboard job boards
- [ ] Add configurable per-remote negentropy concurrency cap
- [ ] Persist normalized filter context in sync job payload/result for admin inspection
- [ ] Persist structured relay rejection details in sync job result for admin inspection

## Phase 18: Testing 🔄 IN PROGRESS

- [ ] Unit tests for policies
- [ ] Unit tests for handlers
- [ ] Integration tests
- [ ] Benchmark tests

## Phase 19: Production Readiness 🔄 IN PROGRESS

- [ ] Docker image optimization
- [ ] Health check endpoint
- [ ] Graceful shutdown
- [ ] Backup strategy
- [ ] Monitoring alerts

## Phase 20: Performance Optimization 🔄 IN PROGRESS

- [x] Remove Ristretto cache (migrating to Redis)
- [x] Add go-redis dependency to go.mod
- [ ] Batch inserts optimization
- [ ] Query optimization
- [ ] Connection pool tuning
- [ ] Cache tuning (Redis)
- [ ] Load testing

## Phase 22: Admin API and Dashboard

- [x] Expand `/admin` API with overview, users, bans and disconnect actions
- [x] Add cursorless incremental windows (`limit` + `offset`) for admin endpoints consumed by virtual scrolling
- [x] Connect `infra/dash` to backend-served admin endpoints
- [x] Replace manual pagination in the SPA with virtualized incremental lists

## Phase 37: NIP-32 Labels Management (NEW)

### 37.1: Documentation and contract
- [ ] Replace legacy labels docs tied to the reference frontend with relay-native requirements
- [ ] Document internal admin endpoints for labels list, summary and creation
- [ ] Record NIP-32 target support (`e`, `p`, `a`, `r`, `t`) and relay-signing behavior

### 37.2: Backend read path
- [ ] Add `infra/db/admin_query.go` queries for `kind:1985` list filters
- [ ] Add `infra/db/admin_query.go` aggregations for labels summary
- [ ] Add `GET /admin/labels`
- [ ] Add `GET /admin/labels/summary`
- [ ] Support repeated `label=` query params for multi-label filtering

### 37.3: Backend write path
- [ ] Add request validation for label creation payloads
- [ ] Build and sign `kind:1985` events with `relay_information.priv_key`
- [ ] Persist signed labels through the existing event storage flow
- [ ] Add focused tests for target mapping, namespace mapping and validation errors

### 37.4: Frontend integration
- [ ] Add `/labels` route to `infra/dash`
- [ ] Add labels service methods, types and TanStack Query hooks
- [ ] Add timeline and by-target management views
- [ ] Add label creation dialog with optional pubkey ban chaining

### 37.5: Admin search/detail enrichment

## Phase 38: Admin Event Search Refactor and Cache Warmup

### 38.1: Docs and contract
- [ ] Document admin event-search hot path refactor in architecture and API docs
- [ ] Record ADR for admin handler split plus warmed Redis cache strategy

### 38.2: Transport split
- [ ] Split `infra/handler/http/admin.go` into focused admin transport files with shared helpers
- [ ] Keep each resulting admin HTTP file under 300 lines

### 38.3: Query optimization
- [ ] Add SQL-first aggregate query methods for `/admin/events/search/aggregates`
- [ ] Add SQL-first timeline query methods for `/admin/events/search/timeline`
- [ ] Keep existing `/admin/events/search` contract while adding normalized response cache keys for paginated pages

### 38.4: Redis cache and warmup
- [ ] Add Redis-backed response cache helpers for admin search, aggregates, and timeline payloads
- [ ] Warm the default admin dashboard event-search payloads only in cron mode when Redis is enabled
- [ ] Reuse event-query invalidation semantics so event writes evict warmed admin search payloads

### 38.5: Validation
- [ ] Verify the three admin search endpoints keep the same response contracts
- [ ] Run focused tests for cache hit/miss behavior and startup warmup flow
- [ ] Extend event search UX to highlight kind `34550` metadata (`d`, `description`, `image`)
- [ ] Include semantic `description` tags in full-text event search
- [ ] Enrich event detail with moderators, richer replies and community metadata

## Phase 39: DB and Redis Cache File Decomposition

### 39.1: Docs and boundary definition
- [ ] Document the `infra/db` and `infra/cache` structural split plan
- [ ] Record ADR for oversized DB and Redis cache file decomposition

### 39.2: Admin DB queries
- [ ] Split `infra/db/admin_query.go` into focused files with max 300 lines each
- [ ] Preserve existing exported query methods and response structs

### 39.3: Event DB queries
- [ ] Split `infra/db/event_query.sql.go` into focused files with max 300 lines each
- [ ] Keep query-cache behavior, prepared statement routing, and event read/write contracts unchanged

### 39.4: Redis cache helpers
- [ ] Split `infra/cache/cache.go` into focused files with max 300 lines each
- [ ] Centralize shared Redis timeout and TTL helpers to reduce duplication

### 39.5: Validation
- [ ] Run focused backend tests for `infra/db` and `infra/cache`
- [ ] Verify cache invalidation and query cache behavior remain compatible

---

## Phase 21: Redis Integration (NEW)

### 21.1: Redis Client Setup
- [ ] Add go-redis/v9 dependency
- [ ] Create `infra/redis/client.go`
- [ ] Implement connection pooling
- [ ] Add configuration for Redis
- [ ] Health check for Redis

### 21.2: Cache Layer
- [ ] Implement `infra/cache/redis.go`
- [ ] Ban user caching (ban:{pubkey})
- [ ] Profile caching (profile:{pubkey})
- [ ] Query result caching (query:{hash})
- [ ] Event caching (event:{id})
- [ ] TTL management

### 21.3: Pub/Sub Infrastructure
- [ ] Create `infra/pubsub/redis.go`
- [ ] Event broadcast channel
- [ ] Subscription channels
- [ ] WebSocket connect/disconnect channels
- [ ] Graceful reconnection

---

## Phase 22: Redis Pub/Sub Listener (NEW)

### 22.1: Subscription Migration
- [ ] Migrate from in-memory to Redis subscriptions
- [ ] Implement `infra/handler/listener/redis.go`
- [ ] Store subscriptions in Redis HASH (subs:{ws_id})
- [ ] Handle subscription create/close via Pub/Sub
- [ ] Maintain backward compatibility

### 22.2: Event Distribution
- [ ] Publish events to Redis on insert
- [ ] Subscribe to event channel
- [ ] Match events against local subscriptions
- [ ] Send to WebSocket clients
- [ ] Handle cross-instance distribution

### 22.3: Connection Management
- [ ] Track WebSocket connections in Redis
- [ ] Publish connect/disconnect events
- [ ] Handle instance shutdown gracefully
- [ ] Clean up orphaned subscriptions

---

## Phase 23: Batch Insert Optimization (NEW)

### 23.1: Ingestion Pipeline
- [ ] Create `infra/ingestion/queue.go`
- [ ] Implement buffered channel queue
- [ ] Add backpressure mechanism
- [ ] Worker pool implementation

### 23.2: Batch Processing
- [ ] Create `infra/ingestion/batch.go`
- [ ] Implement batch accumulation
- [ ] Add timeout-based flush
- [ ] PostgreSQL COPY protocol implementation

### 23.3: Deduplication
- [ ] Implement Redis-based dedup (dedup:{id})
- [ ] Local in-memory dedup cache
- [ ] Graceful handling of duplicates

### 23.4: Error Handling
- [ ] Partial batch success handling
- [ ] Retry logic for failed batches
- [ ] Metrics for batch processing

---

## Phase 24: Query Optimization (NEW)

### 24.1: Prepared Statements
- [ ] Create prepared statements for common queries
- [ ] Implement statement reuse
- [ ] Add query plan caching

### 24.2: Index Improvements
- [ ] Add partial indexes for recent events
- [ ] Add covering indexes
- [ ] Add GIN indexes for JSONB
- [ ] Index maintenance

### 24.3: Query Result Caching
- [ ] Cache frequent query results
- [ ] Invalidate on new events
- [ ] Cache key generation (hash of filter)

### 24.4: Connection Pool Tuning
- [ ] Optimize pool size calculation
- [ ] Add pool metrics
- [ ] Connection health checks
- [ ] Add connection lifetime and idle timeout settings
- [ ] Prepare statements during connection setup

---

## Phase 25: Cache Tuning (NEW)

### 25.1: Multi-tier Cache
- [ ] Implement L1 (local) + L2 (Redis) cache
- [ ] Cache-aside pattern
- [ ] Write-through for critical data

### 25.2: Cache Warming
- [ ] Load hot data on startup
- [ ] Background refresh
- [ ] Cache eviction policies

### 25.3: Metrics
- [ ] Cache hit/miss rates
- [ ] Latency histograms
- [ ] Memory usage tracking
- [ ] Query cache metadata counters

---

## Phase 26: Handler Refactor (NEW)

### 26.1: WebSocket Routing
- [ ] Extract message decode/dispatch from transport loop
- [ ] Keep handler functions transport-thin
- [ ] Centralize envelope/notice mapping

### 26.2: Event / REQ Use Cases
- [ ] Move business flow out of `infra/handler/event`
- [ ] Move business flow out of `infra/handler/req`
- [ ] Reuse single validation entrypoints

---

## Phase 27: Policy Consolidation (NEW)

### 27.1: Single Policy Hub
- [ ] Consolidate event and request policies under one package entrypoint
- [ ] Define structured policy decisions/results
- [ ] Remove duplicated ban and validation checks from handlers

### 27.2: Policy Coverage
- [ ] Event id validation
- [ ] Signature validation
- [ ] Event size validation
- [ ] Ban checks
- [ ] Expiration validation
- [ ] POW validation
- [ ] Tag size/count validation
- [ ] Base64 content rejection
- [ ] Empty filter rejection
- [ ] Protected kind access validation
- [ ] Anti-sync-bot validation

---

## Phase 28: Ingestion Policy Enforcement (NEW)

### 28.1: Batch Validation
- [ ] Reuse consolidated policies inside batch ingestion
- [ ] Validate deduplicated events before persistence
- [ ] Partition ephemeral / replaceable / addressable / regular events

### 28.2: Persistence Semantics
- [ ] Resolve replaceable conflicts in batch flow
- [ ] Resolve addressable conflicts in batch flow
- [ ] Skip ephemeral events without persistence
- [ ] Publish notifications only for accepted events

---

## Phase 29: Stream Performance Refactor (NEW)

### 29.1: Async Dispatcher
- [ ] Introduce bounded async forwarding queue for upstream events
- [ ] Introduce bounded async backfill queue for downstream REQ
- [ ] Add worker-based forwarding execution

### 29.2: Performance Controls
- [ ] Add kind allowlist prefilter before enqueue
- [ ] Avoid synchronous relay-pool calls in handler hot path
- [ ] Add stream queue pressure metrics
- [ ] Add drop/fallback policy for saturated queues

---

## Phase 30: Query / Pool / Cache Tuning (NEW)

### 30.1: Query Optimization
- [ ] Normalize query filters before SQL generation
- [ ] Reuse prepared plans for hot query shapes
- [ ] Add targeted query-cache invalidation on accepted writes

### 30.2: Connection Pool Tuning
- [ ] Add configurable connection lifetime
- [ ] Add configurable idle timeout
- [ ] Add configurable health check period
- [ ] Export pgx pool stats to metrics

### 30.3: Redis Cache Tuning
- [ ] Add query metadata cache keys
- [ ] Track query cache hits/misses
- [ ] Avoid full scan invalidation when targeted invalidation is possible

### 30.4: Subscription Cleanup
- [ ] Add websocket heartbeat keys in Redis
- [ ] Add periodic orphan subscription cleanup loop
- [ ] Broadcast `sub:cleanup` to all instances

---

## Phase 31: Transport Separation (NEW)

### 31.1: HTTP vs WebSocket
- [ ] Move HTTP-only handlers into `infra/handler/http`
- [ ] Move WebSocket-only routing into `infra/handler/ws`
- [ ] Keep transport-agnostic orchestration outside transport packages
- [ ] Move Blossom handlers under HTTP transport package

### 31.2: Event Handler Cleanup
- [ ] Reorganize `infra/handler/event` files by concern
- [ ] Clarify success/error flow for EVENT handling
- [ ] Centralize enqueue/result mapping

---

## Phase 33: Admin API (NEW)

### 33.1: Security
- [ ] Add optional `admin_token` configuration
- [ ] Require `X-Admin-Token` when configured

### 33.2: Endpoints
- [ ] Add ban status endpoint
- [ ] Add ban/unban endpoints
- [ ] Add active connections endpoint
- [ ] Add authenticated connections endpoint
- [ ] Add admin event search endpoint

---

## Phase 35: Optional NIP-29 Groups (NEW)

### 35.1: Discovery and Documentation
- [ ] Document current architecture impact and integration points
- [ ] Record schema deltas between repository and live database
- [ ] Maintain `docs/nip29-coordination.md` during implementation

### 35.2: Configuration and Startup
- [ ] Add `nip29` configuration block and feature toggles
- [ ] Require relay signing key only when `nip29.enabled=true`
- [ ] Initialize optional groups module in server bootstrap
- [x] Normalize `relay_information` keys (`npub`/`nsec`/hex) during config load and derive missing public key from private key

### 35.3: Persistence
- [ ] Add repository-backed queries for group metadata, roles, members and bans
- [ ] Add schema support for invite codes and per-group policy overrides
- [ ] Keep group chat content in the existing `event` table

### 35.4: Policy and Query Flow
- [ ] Validate NIP-29 moderation/join/leave/group content events
- [ ] Enforce membership/read policies on `REQ` and `COUNT`
- [ ] Generate relay-owned `39000`-`39003` state events
- [ ] Narrow NIP-29 EVENT validation to explicit NIP-29 kinds only
- [ ] Narrow NIP-29 REQ/COUNT pre-validation to explicit group-scoped filters (`#h` / NIP-29 state kinds)
- [ ] Add regression coverage for non-NIP-29 events carrying `h`/`d` tags

### 35.5: Optional Protections
- [ ] Invite code support (`kind:9009`)
- [ ] Group/global PoW enforcement
- [ ] Timeline reference enforcement (`previous` tag)

### 35.6: Observability
- [ ] Add Prometheus metrics for group lifecycle, rejections, cache hits/misses and processing latency
- [ ] Add admin/operational visibility for groups where practical

---

## Phase 36: NIP-86 Relay Management API (NEW)

### 36.1: Documentation and Design
- [ ] Document NIP-86 protocol branch on external `/`
- [ ] Document NIP-98 requirements for `kind:27235` with mandatory `payload`
- [ ] Record schema additions for allowlist, blocked IPs, banned events, and relay metadata overrides

### 36.2: Configuration
- [ ] Add `admin_pubkey` configuration with hex/`npub` normalization
- [ ] Keep `admin_token` behavior unchanged for internal `/admin/*`

### 36.3: Transport and Auth
- [ ] Extend root HTTP handler to detect `application/nostr+json+rpc`
- [ ] Implement NIP-98 middleware for exact URL, method, signature, freshness, and payload hash validation
- [ ] Reject non-admin pubkeys with `401`

---

## Phase 39: PostgreSQL Ingestion Hardening (NEW)

### 39.1: Immediate Safety
- [x] Identify the wide `event` indexes causing PostgreSQL `SQLSTATE 54000`
- [x] Document the root cause and the safe replacement index strategy
- [x] Drop `idx_event_covering_author` and `idx_event_covering` concurrently
- [x] Create `idx_event_pubkey_created_at` concurrently

### 39.2: Query and Auth Correctness
- [x] Emit native PostgreSQL placeholders from the event query builder
- [x] Improve batch insert failure logs with event metadata and SQLSTATE
- [x] Improve NIP-42 failure logs with explicit rejection reasons
- [x] Align default `relay_information.canonical_url` with the websocket route `/`

### 39.3: Follow-up Improvements
- [x] Migrate hot tag filters from `tagvalues` overlap to `tags @>` + GIN `jsonb_path_ops`
- [ ] Evaluate normalized `event_tag` helper table only for analytics-heavy workloads
- [ ] Add PostgreSQL integration coverage for large `content` and large `tags`

### 36.4: Service and Repository
- [ ] Add NIP-86 dispatcher and method handlers
- [ ] Add repository support for `banpubkey`, `unbanpubkey`, `allowpubkey`, `unallowpubkey`
- [ ] Add repository support for `allowevent`, `banevent`
- [ ] Add repository support for `changerelayname`, `changerelaydescription`
- [ ] Add repository support for `blockip`, `unblockip`

### 36.5: Runtime Side Effects and Observability
- [ ] Drop live websocket connections after `blockip`
- [ ] Add structured JSON logs for each admin mutation
- [ ] Add focused tests for auth, dispatch, repository mutations, and IP disconnect flow

---

## Phase 32: DB Helper Refactor (NEW)

### 32.1: Documentation First
- [x] Create `infra/db/helper/README.md`
- [ ] Document normalization, validation, SQL rendering, and hashing responsibilities

### 32.2: Package Refactor
- [ ] Split helper package into smaller focused files
- [ ] Improve naming and control flow
- [ ] Preserve deterministic SQL and filter hashing behavior

### 32.3: Test Refactor
- [ ] Remove debug-style assertions and prints
- [ ] Add deterministic normalization tests
- [ ] Add hash stability tests
- [ ] Improve SQL generation assertions

---

## Phase 39: Blossom Admin Workspace and BUD Compliance

### 39.1: Documentation and contracts
- [x] Document `/admin/blossom/*` endpoint contracts and BUD-04/BUD-05/BUD-08 public routes
- [x] Record the queue-backed Blossom processing architecture and EXIF/privacy policy
- [x] Finalize data model for enriched object metadata, quotas, review queue and audit log
- [x] Document Blossom upload policy modes, default plans, BUD-09 report ingestion and BUD-10 IDs
- [ ] Document named plan management and Blossom Prometheus metrics

### 39.2: Backend read path
- [ ] Add paginated `GET /admin/blossom/objects` with exact SHA-256 search and MIME/extension filters
- [ ] Add `GET /admin/blossom/objects/:hash` detail contract
- [ ] Add `GET /admin/blossom/overview` KPI and alert summary
- [ ] Add `GET /admin/blossom/policy` and `GET /admin/blossom/analytics`
- [ ] Add `GET /admin/blossom/plans`
- [ ] Add `GET /admin/blossom/users` and `GET /admin/blossom/users/:pubkey`
- [ ] Add `GET /admin/blossom/reports`
- [ ] Add `GET /admin/blossom/workers` and `GET /admin/blossom/audit`

### 39.3: Backend mutation and processing
- [ ] Add `POST /admin/blossom/objects/bulk-review` for approve, hard-delete and requeue actions
- [ ] Add approval unblock flow for `mandatory_review` uploads
- [ ] Add whitelist/quota mutation endpoints for uploader pubkeys
- [ ] Add `PUT /admin/blossom/policy` for effective mode/default plan management
- [ ] Add `PUT /admin/blossom/plans` and `DELETE /admin/blossom/plans/:id`
- [ ] Add `POST /admin/blossom/users/:pubkey/purge`
- [ ] Add `PUT /mirror` with strict SHA-256 verification and background download execution
- [ ] Validate Blossom auth `kind:24242` for `PUT /mirror` using BUD-11 `upload` semantics (`t`, `expiration`, optional `server`, required `x`)
- [ ] Add `PUT /report` for BUD-09 blob reports
- [x] Add `PUT /media` and `HEAD /media` with derivative status inspection
- [x] Integrate FFmpeg, image processing and blurhash generation in workers
- [x] Persist original `PUT /media` payloads first, then enqueue optimization jobs keyed by canonical hash
- [x] Expose `HEAD /media` readiness headers from persisted derivative state
- [x] Add runtime flags for optional BUD-05 steps (metadata extraction, blurhash, image thumbnails, video thumbnails, HLS, DASH)
- [x] Enable real HLS/DASH generation only when explicitly configured
- [ ] Emit relational and Nostr `kind:24242` audit records for critical mutations
- [ ] Add Prometheus metrics for Blossom requests, latency and categorized errors

### 39.4: Enforcement and cleanup
- [ ] Enforce EXIF/GPS stripping or rejection before object publication
- [ ] Track per-pubkey storage and monthly egress usage
- [ ] Enforce mode-aware quota validation for `free` and `enabled_users`
- [ ] Prefer extension-bearing `/blob/<sha256>.<ext>` URLs when extension is known
- [ ] Add automatic cleanup policies for idle or orphaned objects
- [ ] Persist BUD-08 NIP-94 metadata enrichment after mirror/media processing
- [ ] Regenerate `nip94_tags` as ordered NIP-94 tag tuples from extracted media facts instead of storing partial JSON maps

### 39.5: Validation
- [ ] Add handler tests for admin Blossom routes and policy failures
- [ ] Add queue/worker integration coverage for mirror and optimize jobs
- [ ] Add migration tests for enriched Blossom metadata tables
