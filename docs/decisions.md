# Architecture Decision Records (ADRs)

## ADR-001: Choose Go as Implementation Language

**Status:** Accepted  
**Date:** 2024-01-01

### Context

We needed a language for building a high-performance Nostr relay server.

### Decision

Choose **Go 1.24+** as the implementation language.

### Reasons

1. **Performance**: Go has excellent concurrency support with goroutines
2. **Ecosystem**: Strong support for WebSocket, PostgreSQL, and networking
3. **Deployment**: Single binary compilation simplifies deployment
4. **Nostr Libraries**: `go-nostr` provides comprehensive protocol support
5. **Team Familiarity**: Development team has Go expertise

### Consequences

- ✅ High-performance concurrent connections
- ✅ Simple deployment model
- ✅ Strong type safety
- ⚠️ No built-in generics before Go 1.18 (now resolved)

---

## ADR-002: Use Fiber as HTTP Framework

**Status:** Accepted  
**Date:** 2024-01-01

### Context

Needed an HTTP framework with WebSocket support for the relay server.

### Decision

Use **Fiber v2** for HTTP routing and **gorilla/websocket** for WebSocket handling.

### Reasons

1. **Performance**: Fiber is one of the fastest Go HTTP frameworks
2. **WebSocket**: Built-in WebSocket support via `gofiber/contrib/websocket`
3. **Middleware**: Easy to add CORS, compression, logging
4. **Compatibility**: Works well with Prometheus metrics

### Consequences

- ✅ High throughput for HTTP endpoints
- ✅ Clean middleware chain
- ⚠️ Less flexibility than raw `net/http`

---

## ADR-003: PostgreSQL for Event Storage

**Status:** Accepted  
**Date:** 2024-01-01

### Context

Needed a reliable database for storing Nostr events at scale.

### Decision

Use **PostgreSQL** with **pgx/v5** driver.

### Reasons

1. **JSONB Support**: Native JSONB for tags and flexible schemas
2. **GIN Indexes**: Efficient array and JSONB queries
3. **Full-Text Search**: Built-in `tsvector` for content search
4. **Maturity**: Battle-tested with excellent tooling
5. **pgx**: High-performance driver with connection pooling

### Reasons Against Alternatives

- **SQLite**: Limited concurrency, no network access
- **MySQL**: Weaker JSON support
- **MongoDB**: Overkill for structured data
- **Badger**: Limited query flexibility (used for local cache only)

### Consequences

- ✅ Flexible queries on event data
- ✅ Full-text search capability
- ✅ Connection pooling built-in
- ⚠️ Requires PostgreSQL server setup

---

## ADR-004: Viper for Configuration

**Status:** Accepted  
**Date:** 2024-01-01

### Context

Needed a configuration management system supporting YAML files.

### Decision

Use **Viper** for all configuration management.

### Reasons

1. **YAML Support**: Native YAML parsing
2. **Environment Variables**: Easy override support
3. **Defaults**: Built-in default value support
4. **Type Safety**: Unmarshals to typed Go structs

### Consequences

- ✅ Flexible configuration
- ✅ No hardcoded values
- ⚠️ Runtime config changes require restart

---

## ADR-005: Blossom for File Storage

**Status:** Accepted  
**Date:** 2024-01-01

### Context

Clients needed a way to upload and share media files.

### Decision

Implement **NIP-96 Blossom** file storage server.

### Reasons

1. **Standard**: NIP-96 is the Nostr standard for file storage
2. **Simple**: Hash-based addressing, no complex structure
3. **Decentralized**: Can federate with other Blossom servers
4. **MIME Validation**: Built-in content type checking

### Consequences

- ✅ Standard-compliant file storage
- ✅ Media caching for relay
- ⚠️ Requires disk space management

---

## ADR-006: Negentropy for Relay Sync

**Status:** Accepted  
**Date:** 2024-01-01

### Context

Needed efficient synchronization between relays.

### Decision

Implement **Negentropy** protocol for relay synchronization.

### Reasons

1. **Efficiency**: O(n) instead of O(n²) for full sync
2. **Privacy**: Doesn't reveal exact events to peer
3. **Standard**: Part of Nostr ecosystem
4. **Implementation**: `go-negentropy` library available

### Consequences

- ✅ Fast relay sync
- ✅ Bandwidth efficient
- ⚠️ Requires both relays to support Negentropy

---

## ADR-007: Prometheus for Metrics

**Status:** Accepted  
**Date:** 2024-01-01

### Context

Needed observability for the relay server.

### Decision

Use **Prometheus** for metrics collection with Grafana dashboards.

### Metrics Collected

- Request counts by type (EVENT, REQ, CLOSE, AUTH)
- Event counts by kind
- WebSocket connection counts
- Listener counts
- Upload/download counts
- Event forwarding counts
- Error counts (signature failures, duplicates)

### Consequences

- ✅ Standard monitoring
- ✅ Grafana integration
- ✅ Alert capabilities

---

## ADR-008: Relay Pool for Event Forwarding

**Status:** Accepted  
**Date:** 2024-01-01

### Context

Needed ability to forward events to other relays.

### Decision

Implement a **relay pool** singleton for publishing events.

### Design

```
┌─────────────────────────────────────────┐
│           Nostr Relay Server            │
│                                         │
│   Event ──► Relay Pool ──► Relay 1     │
│                     ├──► Relay 2        │
│                     └──► Relay 3        │
└─────────────────────────────────────────┘
```

### Reasons

1. **Fan-out**: One event to multiple relays
2. **Resilience**: Retry logic for failed publishes
3. **Performance**: Concurrent publishing

### Consequences

- ✅ Automatic federation
- ✅ Reduced client work
- ⚠️ Additional network traffic

---

## ADR-009: Policy-Based Event Validation

**Status:** Accepted  
**Date:** 2024-01-01

### Context

Needed a flexible system for event validation rules.

### Decision

Implement **policy chain** pattern for event validation.

### Policy Chain

1. **ID Check**: Verify event ID matches SHA256
2. **Signature Check**: Verify cryptographic signature
3. **Expiration Check**: NIP-40 expiration timestamps
4. **POW Check**: Proof of work difficulty
5. **Ban Check**: User ban status
6. **Tag Validation**: Tag size/length limits
7. **Content Policy**: Block base64 media

### Consequences

- ✅ Extensible validation
- ✅ Clear separation of concerns
- ✅ Easy to add new rules

---

## ADR-010: Cobra for CLI

**Status:** Accepted  
**Date:** 2024-01-01

### Context

Needed a CLI tool for server management.

### Decision

Use **Cobra** for CLI command structure.

### Commands

- `server` - Start the relay server
- `cron` - Run scheduled tasks
- `import` - Import events from JSONL
- `export` - Export events to JSONL
- `sync` - Sync with remote relay (Negentropy)
- `seed` - Initialize database
- `conf` - Manage configuration

### Consequences

- ✅ Standard CLI patterns
- ✅ Easy subcommands
- ✅ Help generation

---

## ADR-011: Zap for Logging

**Status:** Accepted  
**Date:** 2024-01-01

### Context

Needed structured logging for production debugging.

### Decision

Use **Uber's Zap** for structured logging.

### Reasons

1. **Performance**: Zero allocation logging
2. **Structured**: JSON output for log aggregation
3. **Context**: Easy field addition
4. **Levels**: Debug, Info, Warn, Error

### Consequences

- ✅ Fast logging
- ✅ Easy to parse
- ✅ Grafana Loki compatible

---

## ADR-012: Redis for Cache and Pub/Sub

**Status:** Accepted  
**Date:** 2025-01-01

### Context

Needed a distributed cache and pub/sub system for horizontal scaling and improved performance.

### Decision

Use **Redis** for:
1. **Cache Layer**: Cache banned users, profiles, and query results
2. **Pub/Sub**: Real-time event distribution across instances
3. **Subscriptions**: Replace in-memory listener storage with Redis

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Nostr Relay Cluster                           │
│                                                                      │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐           │
│  │  Instance 1 │    │  Instance 2 │    │  Instance N │           │
│  │   (Go)      │    │   (Go)      │    │   (Go)      │           │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘           │
│         │                    │                    │                  │
│         └────────────────────┼────────────────────┘                  │
│                              │                                       │
│                    ┌─────────▼─────────┐                            │
│                    │      Redis        │                            │
│                    ├───────────────────┤                            │
│                    │  • Cache (STRING) │                           │
│                    │  • Pub/Sub        │                            │
│                    │  • Streams        │                            │
│                    └───────────────────┘                            │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Redis Data Structures

| Key Pattern | Type | TTL | Purpose |
|-------------|------|-----|---------|
| `ban:{pubkey}` | STRING | 1h | Banned user cache |
| `profile:{pubkey}` | HASH | 5m | Profile cache |
| `query:{hash}` | STRING | 30s | Query result cache |
| `event:{id}` | STRING | 10m | Event cache |
| `channel:events` | CHANNEL | - | Event pub/sub |
| `channel:sub:{id}` | CHANNEL | - | Subscription notifications |
| `subs:{ws_id}` | HASH | - | Active subscriptions per WS |

### Reasons

1. **Horizontal Scaling**: Multiple instances share state via Redis
2. **Low Latency**: Sub-millisecond cache access
3. **Pub/Sub**: Real-time event distribution
4. **Persistence**: Optional persistence for durability
5. **Clustering**: Redis Cluster for high availability

### Consequences

- ✅ Horizontal scaling support
- ✅ Reduced database load
- ✅ Real-time event distribution
- ✅ Consistent state across instances
- ⚠️ Redis becomes a single point of failure (requires Sentinel/Cluster)

---

## ADR-013: Optimized Batch Inserts

**Status:** Accepted  
**Date:** 2025-01-01

### Context

High-volume event ingestion required optimization of database write operations.

### Decision

Implement **batch insert optimization** with:
1. **COPY Protocol**: Use PostgreSQL COPY for bulk inserts
2. **Worker Pool**: Concurrent batch processing
3. **Backpressure**: Flow control to prevent memory exhaustion
4. **Deduplication**: Skip already processed events

### Design

```
┌─────────────────────────────────────────────────────────────────┐
│                      Event Ingestion Pipeline                    │
│                                                                   │
│  Client ──► WebSocket ──► Validation ──► Queue ──► Batch       │
│                                                    │             │
│                                              ┌────┴────┐        │
│                                              │ Workers │        │
│                                              │ (COPY)  │        │
│                                              └────┬────┘        │
│                                                   │             │
│                                                   ▼             │
│                                              PostgreSQL          │
└─────────────────────────────────────────────────────────────────┘
```

### Configuration

```yaml
ingestion:
  batch_size: 1000           # Events per batch
  batch_timeout: 100ms       # Max wait time
  workers: 4                  # Concurrent workers
  queue_size: 10000          # In-memory queue
  dedup_cache_size: 50000    # Recent event IDs
```

### Consequences

- ✅ 10x faster bulk imports
- ✅ Reduced database connections
- ✅ Memory-efficient processing
- ⚠️ Added complexity in error handling

---

## ADR-014: Query Optimization

**Status:** Accepted  
**Date:** 2025-01-01

### Context

Query performance degraded with large event tables.

### Decision

Implement **query optimization** strategies:

1. **Covering Indexes**: Include frequently accessed columns
2. **Partial Indexes**: Index filtered subsets
3. **Prepared Statements**: Reuse query plans
4. **Connection Pooling**: Optimal pool sizing
5. **Query Result Caching**: Cache frequent queries

### Index Strategy

| Query Type | Index | Type |
|------------|-------|------|
| By ID | `(id)` | B-tree |
| By Author | `(pubkey, created_at DESC)` | B-tree |
| By Kind | `(kind, created_at DESC)` | B-tree |
| By Tag | `tagvalues` | GIN |
| Full-text | `content_search` | GIN |
| Recent | `WHERE created_at > now()-interval '1 day'` | Partial |

### Prepared Statements

```sql
PREPARE event_by_id AS 
  SELECT * FROM event WHERE id = $1;

PREPARE events_by_author_kind AS 
  SELECT * FROM event 
  WHERE pubkey = $1 AND kind = $2 
  ORDER BY created_at DESC LIMIT $3;
```

### Consequences

- ✅ 5x faster query response
- ✅ Reduced index size
- ✅ Better cache hit rates
- ⚠️ Index maintenance overhead
