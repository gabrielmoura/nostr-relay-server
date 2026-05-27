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
| **Admin SPA** | React 19 + Vite + TanStack Router + i18next | Internal operations dashboard (English/Portuguese localization) |
| **Embedded Assets** | `embed.FS` | Ships `infra/dash/dist` inside the Go binary |

## Planned NIP-32 Labels Management

The relay already stores `kind:1985` label events in the shared `event` table. The next admin feature extends the internal dashboard and admin API so operators can inspect and create NIP-32 labels without relying on a browser-side Nostr signer.

### Backend shape

- **Transport:** internal admin HTTP only (`/admin/labels`, `/admin/labels/summary`, `/admin/labels` POST)
- **Storage:** existing `event` table; no dedicated `labels` table
- **Read path:** `infra/handler/http/admin.go` + dedicated query methods in `infra/db/admin_query.go`
- **Write path:** admin handler validates command payload, builds a `nostr.Event` (`kind:1985`), signs it with `config.Cfg.RelayInformation.PrivKey`, and persists it through the existing event store
- **Moderation coupling:** optional pubkey ban remains a separate admin action through the existing ban endpoints

### Frontend shape

- **Route:** `/labels` inside `infra/dash`
- **State model:** TanStack Query for list, summary and create mutation
- **Feature goal:** timeline view, by-target view, filters, and label creation dialog
- **NIP coverage:** explicit support for targets `e`, `p`, `a`, `r`, and `t`

## Admin Event Search Hot Path

The internal admin dashboard depends on three read-heavy endpoints that are queried together by the SPA:

- `GET /admin/events/search`
- `GET /admin/events/search/aggregates`
- `GET /admin/events/search/timeline`

### Planned backend shape

- **Transport split:** `infra/handler/http/admin.go` stops being the single admin transport file and is split by concern while preserving the same route surface.
- **Search page path:** paginated search remains backed by the existing `event` table query flow, but gains a dedicated Redis response cache for normalized admin filters plus `limit` and `offset`.
- **Aggregate path:** aggregates stop loading the whole result set into Go memory; counts and trends move to SQL-first queries in `infra/db`.
- **Timeline path:** timeline buckets stop being computed by iterating over all matched events in Go; bucketed counts move to SQL-first queries in `infra/db`.
- **Cron-only warm cache:** when Redis is enabled, the dedicated `cron` process precomputes and stores the default dashboard payloads for the first page, aggregates, and timeline. The HTTP server no longer performs this warmup during `server` startup.
- **Invalidation model:** admin search cache keys follow the existing Redis query version invalidation strategy so writes that already invalidate event query cache also evict warmed admin search payloads.

### DB and cache package refactor

The `infra/db` and `infra/cache` packages now carry both transport-facing query paths and cross-cutting cache helpers. The next refactor keeps the package APIs stable but decomposes oversized files by responsibility:

- `infra/db/admin_query.go` splits into admin event search, profile/admin user lookup, bans, event reports, and reported-event analytics files
- `infra/db/event_query.sql.go` splits into event writes, event reads/query cache, retention/cleanup, and streaming/export files
- `infra/cache/cache.go` splits into base Redis access, query cache versioning, profile/ban/event helpers, counters, and NIP-05 document caching files

This decomposition is structural only: callers should keep importing the same packages and methods while the internals become smaller, easier to test, and easier to review for SQL and Redis correctness.

### Warmed default payloads in cron mode

The `nrserver cron` process warms these default admin dashboard reads when it starts:

- `/admin/events/search?limit=50&offset=0`
- `/admin/events/search/aggregates`
- `/admin/events/search/timeline?bucket=day`
- `/admin/events/search/timeline?bucket=hour`

This warm set is intentionally small and deterministic. Arbitrary filtered queries continue to use read-through Redis caching on demand. The dashboard may still observe a cold first fetch when the cron process has not run yet or when a filter combination has not been warmed before.

## Sync Jobs Operator Drill-down

The queue-backed sync flow exposed in `/panel/sync` must preserve enough backend context for later operator inspection inside the generic jobs modal.

Required backend/frontend contract additions:

- sync payload must preserve the normalized filter exactly as executed, even when the original request omits a filter and the runtime falls back to `[{}]`
- sync result must preserve structured remote rejection details for failed publish attempts (`event_id`, `reason`, and optional raw message)
- `last_error` remains the compact board-level summary; the modal becomes the drill-down surface for richer diagnostics
- no new persistence store is introduced; the existing Redis queue `body` and `result` blobs remain the source of truth

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
- Optional `negentropy_auth` enforcement lives at the websocket boundary and binds a Negentropy session to the authenticated relay pubkey.
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

## Optional NIP-29 Groups

- NIP-29 must be an opt-in module that does not change relay startup or baseline EVENT/REQ behavior when disabled.
- The implementation should preserve the current transport and ingestion flow, adding group-aware checks only when an event/filter targets a group via `h`/`d` tags or NIP-29 kinds.
- EVENT guard scope must stay kind-driven on the hot write path: when `nip29.enabled=true`, group existence and membership checks run only for explicit NIP-29 kinds (`9000`-`9022`, `39000`-`39003`) and must not reject unrelated kinds just because they contain `h`/`d` tags.
- REQ/COUNT guard scope must stay filter-driven on the read path: filters with `#h` keep the NIP-29 permission gate before delivery, while filters without group scope continue through the normal relay query path unchanged.
- The recommended integration points are:
  - startup wiring in `cmd/server.go`
  - event/request validation in `internal/policies`
  - persistence adapters in `infra/db`
  - post-persist side effects in `infra/ingestion`
  - relay-generated metadata emission for `39000`-`39003`
- Group chat content continues to live in the existing `event` table; NIP-29-specific tables store authoritative group state, permissions, invites and moderation support data.
- Redis should be used only on the hot path where PostgreSQL round-trips would be repeated frequently (membership lookup, ban lookup, invite redemption, recent timeline references, and cacheable group metadata).

## Transport Separation

- HTTP handlers own route binding, request decoding, status codes, and HTTP payloads
- admin HTTP endpoints expose operational actions and observability on the internal server
- NIP-86 management uses JSON-RPC over HTTP on the external root `/`, sharing the same URI used for WebSocket upgrade
- the embedded dashboard continues to use the internal `/admin/*` surface; it does not act as a browser-side NIP-86 client
- WebSocket handlers own frame decoding, message dispatch, and Nostr envelopes
- Event and REQ packages act as use-case orchestrators and are reused by WebSocket routing only
- Shared business logic should not live in HTTP or WebSocket transport packages

## Planned NIP-86 Integration

- Keep the existing external Fiber root route and add a pre-upgrade HTTP branch for `Content-Type: application/nostr+json+rpc`.
- Reuse NIP-98 verification patterns already present in Blossom auth, but tighten validation for NIP-86: `kind=27235`, signature, short freshness window, exact `method`, exact absolute URL, and mandatory `payload` SHA-256 match.
- Add a focused NIP-86 service layer behind small repository interfaces instead of placing admin mutation logic directly in HTTP handlers.
- Reuse existing runtime structures for:
  - active websocket tracking in `infra/handler/listener`
  - Redis cache/pubsub for block-list acceleration and disconnect fan-out
  - PostgreSQL via `pgx` for authoritative moderation state
- On `blockip`, persist the block, invalidate the hot cache entry, and actively disconnect matching live websocket sessions.
- Preserve the current internal `/admin/*` token-based panel; NIP-86 is an additional external admin protocol, not a replacement.

## Optional Marmot MIP-00 Module

- Marmot `MIP-00` support should be an opt-in module with a dedicated `marmot.mip00` config block and no behavior changes when disabled.
- The phase 1 scope is relay-side only: recognize, validate, store and query Marmot `kind:30443` KeyPackage events plus `kind:10051` relay-list events.
- No new storage model is required in phase 1; the shared `event` table remains authoritative.
- Existing replaceable/addressable ingestion semantics should be reused instead of introducing Marmot-specific persistence flows.
- Validation should stay Nostr-level in the first increment: required tags, tag values, relay URL shapes and payload-size boundaries. MLS payload parsing is explicitly deferred behind a future strict mode.
- The recommended integration points are:
  - config wiring in `config/*`
  - event validation in `internal/policies`
  - no new query transport or admin surface
- The module must not advertise MIP support as a NIP in `supported_nips`; any operator-visible compatibility claim belongs in project documentation instead of the NIP-11 numeric list.

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

## Planned Redis Queue and Worker Infrastructure

Operational background work currently uses three different execution models:

- in-memory admin download jobs
- fire-and-forget admin sync goroutines
- direct cron execution via `robfig/cron`

This is sufficient for local development, but it does not provide durability, reclaim, retry, dead-letter routing or cross-instance worker coordination.

The planned refactor introduces a shared Redis-backed queue subsystem for operational jobs only.

### Scope

- admin download jobs
- admin negentropy sync jobs
- cron-triggered maintenance/fetch routines

The live relay ingestion pipeline in `infra/ingestion` remains a separate in-memory hot path for now.

### Target runtime model

```text
CLI / Admin HTTP / Cron Scheduler
          │
          ▼
   internal/jobs.Dispatcher
          │
          ▼
 Redis Streams + Lua + ZSET + BITFIELD
          │
   ┌──────┴──────┐
   ▼             ▼
 Worker Loop   Delayed Promoter
   │             │
   └──────┬──────┘
          ▼
   Registered Job Handlers
          │
          ▼
   DB / Relay / Existing Services
```

### Package direction

- `internal/jobs` owns contracts, registry and dispatch-facing orchestration
- `infra/queue/redis` owns Redis Streams, Lua scripts, reclaim, metrics and compact tracking
- `cmd/cron` remains the scheduler entry point, but may dispatch queue jobs instead of executing routines inline when queue mode is enabled
- `cmd/worker` may be added as an optional dedicated process for horizontal worker scale

Current rollout status:

- queue-backed download jobs are implemented behind feature flags
- `cmd/worker` is available for dedicated operational workers
- admin sync dispatches to the queue when queue mode is enabled
- cron scheduler mode dispatches queue jobs when queue mode is enabled; one-shot mode still runs inline for compatibility

Next queue/runtime refinements required by the admin UX:

- sync jobs need a real terminal `canceled` state with explicit operator-driven `resume`
- job history cleanup must become a backend deletion capability, not only a client-side hidden list
- sync queue scheduling must enforce a strict but configurable max concurrent negentropy connection count per remote relay
- sync job payloads/results must preserve normalized filter context for later inspection in the dashboard

See `docs/redis-queue-worker-architecture.md` for the detailed design, incremental rollout plan and compatibility constraints.

## Planned Blossom Admin Workspace

The relay already exposes the public Blossom/NIP-96 surface (`POST|PUT /upload`, `GET|HEAD /blob/:id`, `GET /list/:pubkey`) and stores minimal object metadata in the `objects` table. The next operational step is a first-class internal management workspace for media governance, optimization and storage control.

### Backend shape

- **Internal admin transport:** new `/admin/blossom/*` endpoints only; the browser does not call public Blossom routes directly for management operations
- **Public protocol expansion:** keep `POST|PUT /upload` and `GET|HEAD /blob/:id`; add `PUT /mirror`, `PUT /media`, and `HEAD /media` for BUD-04/BUD-05 flows
- **Blossom auth model:** public mutating Blossom routes (`PUT /upload`, `PUT /mirror`, `PUT /media`, `PUT /report`) require a signed Blossom authorization event `kind:24242` per BUD-11, validated before any expensive work starts
- **Queue model:** all mirror, optimize, thumbnail, blurhash, purge and cleanup work runs through the existing Redis-backed jobs runtime; HTTP handlers only enqueue and inspect state
- **Mirror safety:** `PUT /mirror` never performs the remote download inline; the handler validates the request/auth envelope, enqueues a job, and the worker streams the remote body while computing SHA-256, rejecting any mismatch before persistence
- **Media processing:** `PUT /media` persists the original blob first, returns its canonical descriptor immediately, and enqueues heavy optimization work; background workers use `ffmpeg`/`ffprobe` via `os/exec` for audio/video, `imaging` for image resizing, and Blurhash generation after derivative files are materialized
- **Optimization scope:** the current BUD-05 rollout is configuration-driven. Default posture: metadata extraction enabled, Blurhash enabled, image thumbnails enabled, video thumbnails disabled, HLS/MPEG-DASH disabled
- **Configuration model:** media optimization flags live under `store.media_processing.*` so operators can tune CPU/storage pressure without changing route behavior
- **Metadata enrichment:** the same asynchronous extraction pipeline used by BUD-05 is the only source allowed to populate rich NIP-94 tags for uploads and mirrored files
- **Auditability:** critical administrative mutations are persisted and also emitted as Nostr audit events compatible with `kind:24242`
- **Moderation coupling:** the review queue consumes NIP-56 reports plus AI flags, but final accept/delete actions remain explicit operator decisions
- **Upload policy:** one effective Blossom server policy controls `mandatory_review`, `enabled_users`, or `free` upload mode plus default quota plans
- **Named plan catalog:** the admin backend also manages named Blossom plans so operators can reuse quota presets and switch defaults without editing raw byte values repeatedly
- **Canonical download URL:** blob links prefer `/blob/<sha256>.<ext>` when the extension is known, while keeping plain-hash lookup valid
- **Observability:** Blossom HTTP handlers emit dedicated Prometheus request, latency and error metrics with normalized route labels and categorized auth/policy failures

### Frontend shape

- **Route:** `/blossom` inside `infra/dash`
- **Child route:** `/blossom/plans` for deep plan/quota configuration
- **Child routes:** `/blossom/review`, `/blossom/reports`, `/blossom/audit` for lower-level moderation and audit surfaces
- **Workspace model:** the main route becomes a lighter operational hub for KPIs, file browser, quotas, mirroring, worker monitor and drill-down links
- **Navigation model:** operators keep only the high-frequency sections in the main route; lower-frequency moderation/audit sections move into child routes
- **Primary drill-down:** selecting a file opens a right-side inspection sheet with NIP-94 tags, technical metadata, EXIF/privacy state and quick actions
- **Fast overlays:** a header `Workers` button opens a workers modal, and an `Analytics` action opens charts without forcing a tab change
- **Configuration drill-down:** operators move into the plans screen when they need richer plan modeling, default assignments, quotas by scope and explanatory help content
- **Moderation drill-down:** operators move into dedicated review/reports/audit screens when they need deeper inspection; `review` only appears when manual review is enabled

### Data flow summary

1. uploads resolve the effective server policy, validate upload permission/quota, then persist the source object and initial metadata row
2. optimization/mirroring jobs enrich derivatives, blurhash and technical metadata asynchronously
3. `PUT /media` stores the original object, dispatches one optimization job, and immediately returns a descriptor for the original hash
4. BUD-05 workers compute as much metadata as possible from the original upload, logging recoverable extraction failures and continuing with partial enrichment when needed
5. Depending on configuration, BUD-05 workers generate image thumbnails, video thumbnails, optimized WebP/poster derivatives, Blurhash and optional streaming manifests, then regenerate canonical NIP-94 tags
6. BUD-09 reports persist review-signal rows keyed by blob hash and feed the admin reports tab
7. admin reads aggregate relational metadata through `/admin/blossom/*`
8. destructive actions (`hard-delete`, `purge`) enqueue jobs and append audit entries before physical deletion

### Constraints

- no new browser-side signer is introduced
- no long-running FFmpeg/image work happens inside request handlers
- exact SHA-256 remains the canonical object identity for list/search/mirroring
- EXIF/GPS policy is enforced before an object becomes broadly visible in the admin library
- the optional file extension in blob URLs must never change the canonical hash lookup semantics
