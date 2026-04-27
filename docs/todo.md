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

## Phase 34: Cron Consolidation

- [x] Refactor `cron` command to configuration-driven scheduler
- [x] Add DB optimization routine with enable/schedule controls
- [x] Add automatic NIP-56 reported events fetch with explicit relay list
- [x] Add retention cleanup by `older_than_days` with batched deletion

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
