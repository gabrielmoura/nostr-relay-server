# Architecture

## Overview

Nostr Relay Server is a high-performance Nostr relay implementation in Go, supporting multiple NIPs, Blossom file storage (NIP-96), Negentropy synchronization, and Prometheus metrics.

## C4 Model

### Level 1: System Context

```
┌─────────────────────────────────────────────────────────────────┐
│                      Nostr Relay Server                          │
│                                                                   │
│  ┌─────────────┐    WebSocket    ┌─────────────┐                │
│  │   Nostr     │◄───────────────│   Nostr     │                │
│  │   Clients   │                 │   Relay     │                │
│  └─────────────┘                 │   Server   │                │
│                                   │            │                │
│  ┌─────────────┐    HTTP/REST    │            │    ┌────────┐ │
│  │   Nostr     │◄───────────────│            │───►│  DB    │ │
│  │   Clients   │                 │            │    │(Postgres)│ │
│  └─────────────┘                 │            │    └────────┘ │
│                                   │            │                │
│  ┌─────────────┐                 │            │    ┌────────┐ │
│  │ Prometheus  │◄───────────────│            │───►│ Blossom │ │
│  │ / Grafana   │                 │            │    │  Store │ │
│  └─────────────┘                 └────────────┘    └────────┘ │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Level 2: Container View

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Nostr Relay Server                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         CLI (Cobra)                                  │   │
│  │  server │ cron │ import │ export │ sync │ download │ seed │ conf   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                     │                                       │
│  ┌─────────────────────────────────┼───────────────────────────────────┐ │
│  │                         Config Layer                                 │ │
│  │                    (Viper + YAML)                                    │ │
│  └─────────────────────────────────┼───────────────────────────────────┘ │
│                                     │                                       │
│  ┌─────────────────────────────────┼───────────────────────────────────┐ │
│  │                    Application Core                                 │ │
│  │                                                                      │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │ │
│  │  │   Policies   │  │    DTOs      │  │   Bootstrap  │            │ │
│  │  │   (Rules)    │  │  (Messages)  │  │   (Seed)     │            │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘            │ │
│  └─────────────────────────────────┼───────────────────────────────────┘ │
│                                     │                                       │
│  ┌─────────────────────────────────┼───────────────────────────────────┐   │
│  │                     Infrastructure                                │   │
│  │                                                                      │ │
│  │  ┌──────────────────────────────────────────────────────────────┐  │ │
│  │  │                    HTTP Servers (Fiber)                       │  │ │
│  │  │                                                                  │  │ │
│  │  │  Internal (Port+1)          External (Port)                   │  │ │
│  │  │  ├─ /metrics               ├─ /.well-known/...                │  │ │
│  │  │  ├─ /admin                 ├─ /upload (Blossom)               │  │ │
│  │  │  ├─ /panel (Admin SPA)     │                                  │  │ │
│  │  │  │  ├─ overview/users      │                                  │  │ │
│  │  │  │  ├─ connections/events  │                                  │  │ │
│  │  │  │  └─ moderation actions  │                                  │  │ │
│  │  │  └─                         ├─ /blob/:id (Blossom)           │  │ │
│  │  │                             ├─ / (WS - NIP-11)                │  │ │
│  │  │                             └─ /nostr.png                     │  │ │
│  │  └──────────────────────────────────────────────────────────────┘  │ │
│  │                                                                      │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │ │
│  │  │  Event Handler  │  │   REQ Handler   │  │  Auth Handler   │   │ │
│  │  │   (NIP-01)      │  │    (NIP-01)     │  │   (NIP-42)     │   │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘   │ │
│  │                                                                      │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │ │
│  │  │  Listener Mgmt   │  │  Stream/Fwd    │  │   Blossom       │   │ │
│  │  │                  │  │                 │  │   (NIP-96)      │   │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘   │ │
│  │                                                                      │ │
│  └─────────────────────────────────┬───────────────────────────────────┘ │
│                                     │                                       │
│  ┌─────────────────────────────────┼───────────────────────────────────┐ │
│  │                         Data Layer                                 │ │
│  │                                                                      │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │ │
│  │  │  PostgreSQL      │  │   Negentropy    │  │    Relay Pool  │   │ │
│  │  │  (pgx/v5)       │  │   Session Mgmt  │  │   (Forward)     │   │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘   │ │
│  │                                                                      │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| **CLI** | Cobra | Command-line interface |
| **HTTP** | Fiber v2 | Web framework (HTTP + WebSocket) |
| **Database** | PostgreSQL + pgx/v5 | Event storage |
| **Cache/PubSub** | Redis (go-redis/v9) | Distributed cache and pub/sub |
| **Config** | Viper | Configuration management |
| **Logging** | Zap | Structured logging |
| **Metrics** | Prometheus | Observability |
| **WebSocket** | gorilla/websocket | WebSocket connections |
| **Nostr SDK** | go-nostr | Nostr protocol implementation |
| **Admin SPA** | React 19 + Vite + TanStack Router | Internal operations dashboard |
| **Embedded Assets** | `embed.FS` | Ships `infra/dash/dist` inside the Go binary |

## Directory Structure

```
nostr-relay-server/
├── cmd/                    # CLI commands
│   ├── server.go          # Main server command
│   ├── cron.go            # Scheduled tasks
│   ├── import.go          # Import events from JSONL
│   ├── export.go          # Export events to JSONL
│   ├── sync.go            # Negentropy sync
│   ├── down.go            # Relay event download command
│   ├── seed.go            # Database seeding
│   ├── conf.go            # Configuration management
│   ├── root.go            # Root command
│   └── internal/          # Command-specific runtime logic
│       ├── down/          # Download parsing + execution
│       ├── import/        # Import parsing + execution
│       ├── export/        # Export parsing + execution
│       ├── seed/          # Seed parsing + execution
│       ├── cron/          # Cron options, job map and runner
│       ├── conf/          # Config print/write/validate flows
│       └── sync/          # Sync parsing + execution
├── config/                # Configuration
│   ├── config.go         # Config loading (Viper)
│   └── conf-data.go      # Config structures
├── internal/              # Private application code
│   ├── bootstrap/         # Initial data seeding
│   ├── db/                # Database interfaces
│   ├── dto/               # Data transfer objects
│   ├── errors/            # Custom errors
│   └── policies/          # Business rules
├── infra/                  # Infrastructure
│   ├── cache/             # Redis caching layer
│   ├── cron/              # Scheduled tasks
│   ├── db/                # Database implementation
│   ├── handler/           # Transport handlers only
│   │   ├── http/          # HTTP-only handlers and endpoints
│   │   ├── ws/            # WebSocket-only message routing
│   │   ├── event/         # EVENT use-case orchestration
│   │   ├── req/           # REQ use-case orchestration
│   │   ├── auth/          # NIP-42 auth
│   │   ├── listener/      # Subscription management (Redis Pub/Sub)
│   │   ├── store/blossom/ # Blossom file storage
│   │   └── count/         # COUNT handling
│   ├── ingestion/          # Batch event ingestion pipeline
│   ├── log/               # Logging (Zap)
│   ├── metrics/          # Prometheus metrics
│   ├── net/               # HTTP router
│   ├── nostr-custom/      # Custom Nostr kinds
│   ├── pubsub/            # Redis Pub/Sub wrapper
│   └── stream/            # Event streaming
├── pkg/                    # Public packages
│   ├── negentropy/         # Negentropy sync
│   ├── negentropyV2/       # Negentropy V2 engine/cache/service
│   ├── nostrpool/         # Relay pool
│   ├── magic/             # File type detection
│   └── webc/              # Web content fetcher
└── docs/                  # Documentation
```

## Key Design Patterns

### 1. Listener Pattern (Pub/Sub)
- Clients subscribe to events via `REQ` messages
- Events are matched against active subscriptions
- Notification is sent via WebSocket channels

### 2. Singleton Pool
- `nostrpool.RelayPool` - singleton for forwarding events
- `SessionManager` - singleton for Negentropy sessions

### 3. Policy Hub
- A single policy hub validates EVENT and REQ flows
- The same rules are reused by live handlers and batch ingestion
- Handlers translate policy decisions into protocol envelopes

### 4. DTO Pattern
- `WsServer` - WebSocket connection context
- `Data` - Raw message payload
- Filter objects for queries

### 5. Negentropy Layering
- `pkg/negentropy` provides relay-facing message handlers and protocol wiring.
- `pkg/negentropyV2` provides reconciliation engine, cache abstraction, and session management.
- Handler layer delegates `NEG-OPEN` / `NEG-MSG` / `NEG-CLOSE` orchestration to V2 services.
- Cache backend is selected by runtime capabilities (Redis when enabled, memory fallback otherwise).

## Flow: Negentropy Synchronization

```text
Remote Relay / Sync Client
       │
       ▼
┌──────────────────┐
│ WS Message Router │ ─── NEG-OPEN / NEG-MSG / NEG-CLOSE
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ pkg/negentropy    │ ─── protocol validation + envelope mapping
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ negentropyV2      │ ─── session manager + reconcile service
├──────────────────┤
│ Cache             │ ─── memory or Redis TTL cache
│ EventStore        │ ─── query bridge to db.DbQueries
└────────┬─────────┘
         │
         ├─► NEG-MSG response frames
         ├─► optional EVENT/REQ transfer path (sync client compatibility)
         └─► metrics emission (`nostr_negentropy_v2_*`)
```

## Flow: Event Processing

```
Client WebSocket
       │
       ▼
┌──────────────────┐
│  handleMessage   │ ─── Parse JSON
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Type Dispatch    │ ─── EVENT/REQ/CLOSE/AUTH/COUNT
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Handler Router  │
└────────┬─────────┘
         │
      ┌───┴───┐
     │ Policy Hub │
     ├─ Event identity
     ├─ Signature and ban checks
     ├─ Expiration and POW
     ├─ Tag/content constraints
     └─ Event-kind semantics
     │
     ▼
┌──────────────────┐
│  Ingestion Queue │ ─── Validated event handoff
└────────┬─────────┘
         │
      ┌───┴───┐
     │ Batch Worker │
     ├─ Dedupe
     ├─ Storage-safe policies
     ├─ Replaceable resolution
     ├─ Batch INSERT
     └─ Cache + notify
     │
     ▼
┌──────────────────┐
│ Stream Dispatcher│ ─── Async upstream / downstream forwarding
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Redis Pub/Sub   │ ─── Broadcast to all instances
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  NotifyListeners │ ─── Match subscriptions and send envelopes
└────────┬─────────┘
         │
         ▼
    Client receives EVENT envelope
```

## Flow: Request Processing

```text
Client WebSocket
       │
       ▼
┌──────────────────┐
│  Handler Router  │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Policy Hub      │
├─ subscription id │
├─ filter decode   │
├─ auth rules      │
├─ empty filter    │
└─ protected kinds │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Query Executor  │
├─ local DB        │
└─ optional stream │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Listener Registry│
└──────────────────┘
```

## Policy Consolidation

- `infra/handler` should only decode, route, and translate protocol responses
- `internal/policies` becomes the single source of truth for EVENT and REQ validation
- `infra/ingestion` reuses the same policy package before persistence
- `infra/stream` uses dedicated forwarding rules instead of handler-driven checks

## Transport Separation

- HTTP handlers own route binding, request decoding, status codes, and HTTP payloads
- admin HTTP endpoints expose operational actions and observability on the internal server
- WebSocket handlers own frame decoding, message dispatch, and Nostr envelopes
- Event and REQ packages act as use-case orchestrators and are reused by WebSocket routing only
- Shared business logic should not live in HTTP or WebSocket transport packages

## Helper Package Design

- `infra/db/helper` is a query-construction package, not a persistence layer
- it owns filter normalization, SQL rendering, and cache-key hashing
- it does not know about Redis invalidation, pgx execution, or transport concerns

## Redis Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Redis Data Structures                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                        CACHE LAYER                               │ │
│  │                                                                  │ │
│  │  ban:{pubkey}     → "reason"          TTL: 1h                  │ │
│  │  profile:{pubkey} → Hash{name,about}  TTL: 5m                  │ │
│  │  query:{hash}     → JSON results      TTL: 30s                  │ │
│  │  event:{id}       → JSON event        TTL: 10m                  │ │
│  │  dedup:{id}       → "1"               TTL: 1h                   │ │
│  │                                                                  │ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                       PUB/SUB LAYER                              │ │
│  │                                                                  │ │
│  │  Channel: events                                                 │ │
│  │  Channel: sub:{subscription_id}                                 │ │
│  │  Channel: ws:{ws_id}                                           │ │
│  │                                                                  │ │
│  │  ┌─────────┐    ┌─────────┐    ┌─────────┐                     │ │
│  │  │ Instance│    │ Instance│    │ Instance│                     │ │
│  │  │    1   │    │    2   │    │    N   │                     │ │
│  │  └────┬────┘    └────┬────┘    └────┬────┘                     │ │
│  │       │              │              │                            │ │
│  │       └──────────────┼──────────────┘                            │ │
│  │                      │                                           │ │
│  │                      ▼                                           │ │
│  │               ┌─────────────┐                                    │ │
│  │               │    Redis    │                                    │ │
│  │               └─────────────┘                                    │ │
│  │                                                                  │ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                     SUBSCRIPTION STORAGE                         │ │
│  │                                                                  │ │
│  │  subs:{ws_id} → Hash{                                          │ │
│  │                   sub_1: JSON{filter, created_at},             │ │
│  │                   sub_2: JSON{filter, created_at}               │ │
│  │                 }                                               │ │
│  │                                                                  │ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```
Client WebSocket
       │
       ▼
┌──────────────────┐
│  handleMessage   │ ─── Parse JSON
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Type Dispatch    │ ─── EVENT/REQ/CLOSE/AUTH/COUNT
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  DoEVENT         │
└────────┬─────────┘
         │
    ┌────┴────┐
    │ Policies │
    ├─ ID check
    ├─ Signature check
    ├─ Expiration (NIP-40)
    ├─ POW check
    ├─ Ban check
    ├─ Tag size/length
    └─ Base64 media check
    │
    ▼
┌──────────────────┐
│  publish()       │
└────────┬─────────┘
         │
    ┌────┴────┐
    │ Event    │
    │ Type     │
    ├─ Replaceable → Delete old
    ├─ Ephemeral  → Don't store
    ├─ Parameter  → Delete old
    └─ Regular     → Insert
    │
    ▼
┌──────────────────┐
│  NotifyListeners │ ─── Match subscriptions
└────────┬─────────┘       and send events
         │
         ▼
    Client receives EVENT envelope
```

## Security Considerations

1. **Event Validation**: ID, signature, size checks
2. **Rate Limiting**: Per-connection limits via Fiber middleware
3. **Authentication**: NIP-42 challenge-response
4. **Content Policy**: Blossom content filtering
5. **Ban System**: User/PubKey blocking

## Scalability Notes

- Connection pooling for PostgreSQL
- **Redis-based subscriptions**: Shared state across instances for horizontal scaling
- **Redis Pub/Sub**: Real-time event distribution to all instances
- **Redis Cache**: Sub-millisecond access for banned users, profiles, queries
- Optional event forwarding to relay pool
- Batch inserts for imports using PostgreSQL COPY protocol
- Prepared statements for query optimization
- Partial indexes for common query patterns

## Performance Optimizations

### Query Optimization
- Covering indexes for frequent queries
- Partial indexes for recent events
- Prepared statement reuse
- Query result caching

### Connection Pool Tuning
- Dynamic pgx pool sizing based on configured limits
- Connection lifetime and idle eviction
- Health checks and pool metrics
- Prepared statements on acquired connections

### Batch Insert Optimization
- Worker pool for concurrent batch processing
- In-flight queue with backpressure
- Deduplication via Redis
- Policy validation before persistence
- Replaceable/addressable partitioning before insert

### Cache Tuning
- Multi-tier caching: Redis + local memory
- Cache-aside pattern with TTL management
- Cache warming on startup
- Metrics for cache hit/miss rates

### Subscription Hygiene
- Redis-backed websocket registration heartbeat
- Periodic orphan subscription cleanup
- Local and distributed subscription eviction

### Handler / Stream Refactor
- Thin WebSocket handlers with explicit message router
- Single policy hub shared across transport and ingestion
- Async stream dispatcher with bounded workers
- Reduced synchronous work on hot WebSocket paths

## Cron Consolidation Pipeline

The `cron` command is responsible for backend data consolidation and maintenance routines.

### Scheduled Jobs

1. **Database Optimization**
   - Optional `ANALYZE` and `VACUUM (ANALYZE)` routines.
   - Optional index maintenance (`REINDEX TABLE CONCURRENTLY event`).
   - Controlled by configuration flags and explicit cron schedule.

2. **Reported Events Auto-Fetch (NIP-56)**
   - Optional background fetch for kind `1984` report events from explicit relay list.
   - Requires `enabled=true` and at least one configured relay URL.
   - Uses lookback window and per-relay result logging.
   - Persists events into local DB through existing deduplicated insert path.

3. **Old Event Retention Cleanup**
   - Optional deletion of events older than configured period in days.
   - Uses batched deletion loops to avoid long-running table locks.
   - Example policy: delete events older than 365 days.

### Operational Notes

- All cron jobs are opt-in by configuration.
- Each routine has independent schedule and enable flags.
- Jobs are executed with bounded context timeouts and structured logs.
