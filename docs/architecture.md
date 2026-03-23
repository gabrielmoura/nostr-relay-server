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
│  │  server │ cron │ import │ export │ sync │ seed │ conf              │   │
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

## Directory Structure

```
nostr-relay-server/
├── cmd/                    # CLI commands
│   ├── server.go          # Main server command
│   ├── cron.go            # Scheduled tasks
│   ├── import.go          # Import events from JSONL
│   ├── export.go          # Export events to JSONL
│   ├── sync.go            # Negentropy sync
│   ├── seed.go            # Database seeding
│   ├── conf.go            # Configuration management
│   └── root.go            # Root command
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
│   ├── handler/           # HTTP/WS handlers
│   │   ├── event/         # Event handling
│   │   ├── req/           # REQ handling
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

### 3. Policy Chain
- Events pass through multiple policy checks
- Each policy can accept or reject

### 4. DTO Pattern
- `WsServer` - WebSocket connection context
- `Data` - Raw message payload
- Filter objects for queries

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
│  DoEVENT         │
└────────┬─────────┘
         │
     ┌───┴───┐
     │ Policies │
     ├─ ID check
     ├─ Signature check
     ├─ Expiration (NIP-40)
     ├─ POW check
     ├─ Ban check (Redis Cache)
     ├─ Tag size/length
     └─ Base64 media check
     │
     ▼
┌──────────────────┐
│  Ingestion Queue │ ─── Batch processing
└────────┬─────────┘
         │
     ┌───┴───┐
     │ Batch   │
     │ Worker  │
     ├─ Dedupe (Redis)
     ├─ Batch INSERT (COPY)
     └─ Cache event
     │
     ▼
┌──────────────────┐
│  Redis Pub/Sub   │ ─── Broadcast to all instances
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  NotifyListeners │ ─── Match subscriptions (Redis)
└────────┬─────────┘       and send events
         │
         ▼
    Client receives EVENT envelope
```

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

### Batch Insert Optimization
- Worker pool for concurrent batch processing
- PostgreSQL COPY protocol for bulk inserts
- In-flight queue with backpressure
- Deduplication via Redis

### Cache Tuning
- Multi-tier caching: Redis + local memory
- Cache-aside pattern with TTL management
- Cache warming on startup
- Metrics for cache hit/miss rates
