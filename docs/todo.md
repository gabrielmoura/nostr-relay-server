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
