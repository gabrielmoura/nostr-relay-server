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

---

## ADR-009: Incremental Security Hardening Layer

**Status:** Accepted  
**Date:** 2026-04-22

### Context

The relay already had validation and moderation hooks, but lacked a cohesive hardening layer for configurable whitelist rules, request shaping, resilient abuse controls, and protocol-level integrity metrics.

### Decision

Add a focused `internal/security` package and integrate it incrementally into existing websocket, EVENT, REQ, and policy hub flows instead of introducing a parallel architecture.

### Reasons

1. **Low-risk evolution**: preserves the current runtime flow and global initialization pattern.
2. **Operational clarity**: keeps security configuration centralized in `config.Security`.
3. **Protocol consistency**: centralizes standardized Nostr rejection prefixes.
4. **Observability readiness**: exposes security counters in Prometheus-compatible form.
5. **Redis compatibility**: allows optional progressive defense using TTL-based counters without making Redis mandatory.

### Consequences

- ✅ Security changes remain localized and incremental.
- ✅ Whitelist and defense behavior can be enabled gradually.
- ✅ Future Prometheus dashboards can build on stable counters.
- ⚠️ Global state remains the integration style for now.
- ⚠️ Redis-backed defense requires threshold tuning before production enablement.

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
| `nip05:doc` | STRING | 24h | NIP-05 document cache |
| `channel:events` | CHANNEL | - | Event pub/sub |
| `channel:sub:{id}` | CHANNEL | - | Subscription notifications |
| `subs:{ws_id}` | HASH | - | Active subscriptions per WS |

### Reasons

1. **Horizontal Scaling**: Multiple instances share state via Redis
2. **Low Latency**: Sub-millisecond cache access
3. **Pub/Sub**: Real-time event distribution
4. **Persistence**: Optional persistence for durability

---

## ADR-013: Cron Data Consolidation Routines

**Status:** Accepted  
**Date:** 2026-03-25

### Context

Operational data quality required scheduled backend routines for:

1. database optimization,
2. automatic ingestion of NIP-56 reported events from external relays,
3. retention cleanup of old events.

The previous cron implementation was static and not configurable enough for production tuning.

### Decision

Refactor `cron` command into a configuration-driven scheduler with independent jobs:

- `cron.db_optimization`: analyze/vacuum/index maintenance,
- `cron.reported_events_fetch`: fetch kind `1984` events from explicit relay list,
- `cron.delete_old_events`: remove events older than configured day threshold.

Each job is enabled/disabled independently and has its own cron expression.

### Consequences

- ✅ safer operations through explicit feature flags
- ✅ better observability with per-job logging
- ✅ predictable retention policy management
- ⚠️ more configuration surface to maintain
5. **Clustering**: Redis Cluster for high availability

### NIP-05 cache TTL decision

The NIP-05 document cache (`nip05:doc`) uses a default TTL of **24h**.

Rationale:
- long enough to reduce repeated assembly and NIP-65 hint parsing cost,
- short enough to keep eventual consistency bounded,
- every manual mutation (`create/update/delete`) invalidates the cache immediately.

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

---

## ADR-015: Consolidate Policies Into a Single Hub

**Status:** Proposed  
**Date:** 2026-03-23

### Context

Policy logic is currently split across WebSocket handlers and multiple files in `internal/policies`, with some event checks still embedded directly in handlers and batch ingestion not reusing the same validation flow.

### Decision

Refactor policy handling into a single policy hub that exposes clear entrypoints for:

1. Incoming EVENT validation
2. Incoming REQ validation
3. Storage-safe batch ingestion validation
4. Stream forwarding decisions

### Reasons

1. **Consistency**: The same rules should apply to live ingest and batch ingest
2. **Maintainability**: Handlers should not own business rules
3. **Performance**: Centralized normalization avoids repeated work
4. **Safety**: Batch ingestion must not bypass runtime rules

### Consequences

- ✅ One place to understand relay policy behavior
- ✅ Thinner handlers
- ✅ Easier testing of event and request rules
- ⚠️ Requires careful migration to avoid protocol regressions

---

## ADR-016: Async Stream Dispatcher

**Status:** Proposed  
**Date:** 2026-03-23

### Context

`infra/stream` currently forwards requests and events directly from handler paths, which adds avoidable latency and mixes relay federation with local response handling.

### Decision

Refactor stream forwarding into a bounded asynchronous dispatcher with dedicated workers for:

1. Event upstream forwarding
2. REQ downstream backfill

### Reasons

1. **Hot-path protection**: WebSocket handlers should stay focused on local protocol work
2. **Backpressure**: Bounded queues protect relay stability
3. **Observability**: Worker metrics clarify forwarding health
4. **Isolation**: Federation failures should not slow down local clients

### Consequences

- ✅ Lower tail latency on EVENT / REQ handlers
- ✅ Better control over retries and queue pressure
- ✅ Simpler performance tuning
- ⚠️ Requires explicit queue/drop policy

---

## ADR-017: Tune Query, Pool and Cache Hot Paths

**Status:** Proposed  
**Date:** 2026-03-24

### Context

The relay now has batch ingestion, Redis-backed subscriptions and prepared statements, but query construction, database pool lifecycle, cache metadata and subscription cleanup still need production-oriented tuning.

### Decision

Implement focused performance tuning in four areas:

1. **Query optimization**: canonical query normalization, prepared-plan reuse and targeted cache invalidation
2. **Connection pool tuning**: lifetime, idle timeout, health checks and pool observability
3. **Redis cache tuning**: metadata-aware query cache, bounded TTLs and explicit invalidation paths
4. **Subscription hygiene**: heartbeat-based cleanup of orphaned Redis subscription keys

### Reasons

1. **Latency**: hot REQ paths should avoid repeated avoidable work
2. **Stability**: long-lived stale DB or Redis resources hurt tail latency
3. **Observability**: tuning without metrics is guesswork
4. **Distributed correctness**: stale subscriptions create noisy fan-out and memory leaks

### Consequences

- ✅ Better query/cache hit rates
- ✅ More predictable pgx pool behavior under load
- ✅ Lower Redis key churn and stale subscription buildup
- ✅ Safer horizontal scaling behavior
- ⚠️ Adds a few operational knobs that must be documented clearly

---

## ADR-018: Separate HTTP and WebSocket Transports

**Status:** Proposed  
**Date:** 2026-03-24

### Context

`infra/handler` currently mixes transport concerns with business orchestration. HTTP route concerns and WebSocket message concerns should not share the same structural entrypoints.

### Decision

Split handler responsibilities into transport-specific packages:

1. `infra/handler/http` for HTTP-only flows
2. `infra/handler/ws` for WebSocket-only flows
3. keep event/req/count/auth packages as use-case orchestration layers

### Reasons

1. **Clarity**: transport concerns become easier to trace
2. **Testing**: HTTP and WebSocket paths can be exercised independently
3. **Maintainability**: less incidental coupling between envelope handling and HTTP responses

### Consequences

- ✅ clearer package boundaries
- ✅ better readability for future features
- ⚠️ requires moving some files and imports

---

## ADR-019: Refactor Query Helper Package for Deterministic SQL Generation

**Status:** Proposed  
**Date:** 2026-03-24

### Context

`infra/db/helper` became more important after query normalization, prepared-plan routing, and Redis query cache versioning. It now needs clearer internal boundaries and stronger tests.

### Decision

Refactor the helper package into explicit responsibilities:

1. normalization
2. validation
3. SQL rendering
4. stable filter hashing

### Reasons

1. **Readability**: each concern stays small and obvious
2. **Determinism**: query ordering and hashing must stay stable
3. **Testing**: smaller helpers are easier to verify precisely

### Consequences

- ✅ easier maintenance of query behavior
- ✅ better tests with less incidental setup
- ⚠️ small increase in file count inside the helper package

---

## ADR-020: Internal Admin API With Optional Token Enforcement

**Status:** Proposed  
**Date:** 2026-03-24

### Context

The relay needs operational endpoints for ban management, connection inspection, and event search. These endpoints live on the internal server, but some deployments still require explicit application-level protection.

### Decision

Expose admin endpoints on the internal server and support `X-Admin-Token` as an optional configuration. When `admin_token` is non-empty, the header becomes mandatory for all `/admin/*` routes.

### Consequences

- ✅ keeps admin surface on the internal server
- ✅ allows secure deployments without forcing local-only setups
- ⚠️ token rotation remains an operational concern

---

## ADR-021: Refactor `download` Command and Add JSON `--filter`

**Status:** Accepted  
**Date:** 2026-04-16

### Context

The `download` command had parsing, relay orchestration and persistence concerns concentrated in one file, with limited validation and no flexible JSON filter input.

### Decision

Refactor `download` into cohesive layers and add optional JSON filter input via `--filter` or `--filter-file`.

The command now separates:

1. CLI argument parsing (`cmd/down.go`)
2. option validation + filter merge (`cmd/internal/down/options.go`)
3. runtime setup + concurrent relay execution (`cmd/internal/down/download.go`)
4. paginated fetch and persistence (`cmd/internal/down/fetch.go`)

### Precedence Rule

To preserve existing behavior, specific flags override overlapping fields from `--filter`:

- `--kinds` overrides `kinds`
- `--tags` overrides `#t`
- `--public-key` sets `authors=[pk]`
- `--mentioned --public-key` sets `#p=[pk]` and clears `authors`

The merge behavior is configurable:

- `override` (default): explicit flags overwrite overlapping JSON fields
- `strict-conflict`: command fails when overlapping values conflict

### Consequences

- ✅ clearer separation of responsibilities
- ✅ explicit and testable filter parsing/validation
- ✅ improved CLI error messages for invalid input
- ✅ adds per-relay download observability metrics (received/persisted/duplicates/failures/page latency)
- ✅ better maintainability for future pagination and batching changes
- ⚠️ stricter validation (`--timeout > 0`, `--mentioned` requires `--public-key`) may reject previously ambiguous invocations

---

## ADR-022: CLI Operational Refactor for `seed`, `cron` and `conf`

**Status:** Accepted  
**Date:** 2026-04-16

### Context

Operational commands had limited terminal UX, sparse help text, low discoverability and mixed responsibilities between Cobra wiring and runtime logic.

### Decision

Refactor command surfaces and split runtime logic into command-focused internal packages:

- `cmd/internal/seed`
- `cmd/internal/cron`
- `cmd/internal/conf`

Also improve root command UX by removing placeholder root flags and standardizing error behavior.

### Key Command Decisions

1. **`seed`**
   - Add explicit operational controls: `--bootstrap`, `--skip-migrate`, `--dry-run`, `--timeout`.
   - Add optional idempotent bootstrap mode (`--bootstrap-idempotent`) using marker tags to avoid duplicated bootstrap insertion.
   - Keep default behavior backward-compatible: migration still runs by default.

2. **`cron`**
   - Add operational modes: `--list`, `--run-once`, `--job`, `--timeout`.
   - Keep default mode as long-running scheduler.

3. **`conf`**
   - Expand command set: `print/show`, `effective`, `validate`, `write`.
   - Add output controls (`--format`) and file target controls (`--file`, `--force`).

### Consequences

- ✅ clearer operator workflows and better discoverability
- ✅ more predictable command execution and error handling
- ✅ lower coupling between Cobra adapters and execution logic
- ✅ improved maintainability and extension paths for future commands
- ⚠️ slightly larger internal command package surface

---

## ADR-023: Refactor `import`/`export` Commands and Add TSV Export

**Status:** Accepted  
**Date:** 2026-04-16

### Context

`import` and `export` command surfaces had limited operational controls, sparse validation and weak format extensibility.

### Decision

Refactor both commands with structured option parsing and clearer runtime flow:

- `import`: explicit CLI options, validation, configurable stats interval, and optional fail-on-error behavior
- `export`: structured options with format abstraction, optional filter source (`--filter` or `--filter-file`), export `--limit`, segmented files (`--segment-size`), and safer write controls (`--overwrite`, `--no-header` for TSV)

### TSV Decision

TSV export format uses stable columns:

`id`, `pubkey`, `created_at`, `kind`, `tags`, `content`, `sig`

The `tags` field is serialized as JSON string inside TSV to preserve nested tag structure.

### Consequences

- ✅ better operational UX and safer validations
- ✅ extensible export format layer for future formats
- ✅ improved compatibility for spreadsheet/ETL workflows with TSV
- ✅ safer automation behavior through explicit overwrite policy
- ⚠️ segmented writes introduce additional file open/close overhead for very small segment sizes

---

## ADR-F001: React 19 + TanStack Router for Admin SPA

**Status:** Accepted  
**Date:** 2026-04-17

### Context

The admin dashboard needed a modern frontend stack that provides good developer experience, type safety, and performance.

### Decision

Use **React 19** with **TanStack Router** and **i18next** for the admin SPA.

### Technology Choices

| Component | Technology | Rationale |
|-----------|------------|-----------|
| UI Framework | React 19 | New hooks (useActionState, useOptimistic), compiler benefits |
| Routing | TanStack Router | File-based routing, type-safe navigation, lightweight |
| i18n | i18next | Mature, React bindings, good tooling |
| Build | Vite | Fast HMR, optimized production builds |
| UI Primitives | Radix UI | Accessible, headless, composable |
| Styling | Tailwind CSS | Utility-first, consistent design system |

### Consequences

- ✅ Type-safe routing with TanStack Router
- ✅ React 19 hooks for modern form handling
- ✅ i18n support (English/Portuguese)
- ✅ Fast development with Vite
- ⚠️ Need to stay current with React 19 evolution

---

## ADR-F002: Component Separation (Smart vs. Dumb)

**Status:** Accepted  
**Date:** 2026-04-17

### Context

Route components grew too large (944 lines for event-detail-page) with mixed UI logic and business logic.

### Decision

Enforce strict separation:

1. **Dumb Components** (`components/ui/`, `components/shared/`):
   - Pure presentational
   - Receive data via props
   - Emit events via callbacks
   - No API calls

2. **Smart Components** (`routes/`, `components/features/`):
   - Fetch data
   - Manage state
   - Orchestrate dumb components

### Implementation

- Extract parsing logic to `lib/*.ts`
- Create feature-specific components in `components/features/[feature]/`
- Route components orchestrate, don't render

### Consequences

- ✅ More testable components
- ✅ Better reusability
- ✅ Clearer code ownership
- ⚠️ More files to manage

---

## ADR-F003: Parser Extraction Pattern

**Status:** Accepted  
**Date:** 2026-04-17

### Context

Event parsing logic (imeta tags, media extraction, reference tracking) was duplicated across route components.

### Decision

Extract all parsing logic to dedicated files in `lib/`:

- `lib/event-parser.ts` - Event tag parsing for detail page
- `lib/event-search.ts` - Search result transformation
- `lib/router.ts` - Navigation utilities

### Pattern

```typescript
// lib/event-parser.ts
export interface ParsedEvent {
  images: MediaItem[];
  videos: MediaItem[];
  references: NostrReference[];
}

export function parseEventTags(event: RawEvent): ParsedEvent {
  // parsing logic
}
```

### Consequences

- ✅ No duplicated parsing logic
- ✅ Single source of truth for transformations
- ✅ Easier to test parsing in isolation
- ⚠️ Need to keep parsers in sync with backend changes

---

## ADR-024: Optional NIP-29 Groups Module

**Status:** Proposed  
**Date:** 2026-04-22

### Context

The relay already stores generic Nostr events and exposes a shared policy/ingestion pipeline. The goal is to add NIP-29 group support without rewriting the relay core, while keeping startup, existing event flow and deployment topology stable when the feature is disabled.

The live PostgreSQL database already contains draft `nip29_*` tables, but the repository schema and runtime code do not currently wire them into production behavior.

### Decision

Implement NIP-29 as an **optional module** with explicit configuration and incremental integration points:

1. **Configuration-first enablement**: no group logic runs unless `nip29.enabled=true`
2. **Reuse current hot paths**: keep `event` as the source of stored group content; add NIP-29 state tables for membership, roles, bans, invites and policy overrides
3. **Minimal startup wiring**: initialize the module in `cmd/server.go` after DB/Redis setup and before handlers start serving traffic
4. **Policy integration**: extend `internal/policies` with group-aware EVENT/REQ checks instead of branching logic in handlers
5. **Ingestion side effects**: update group state and emit relay-generated metadata/admin/member/role events after accepted writes
6. **Targeted Redis use**: cache only repeated hot lookups (membership, bans, invite codes, recent timeline references, metadata)

### Reasons

1. **Compatibility**: disabled mode must behave like the current relay
2. **Safety**: incremental integration keeps rollback surface small
3. **Performance**: fast membership/ban checks are needed on write and read hot paths
4. **Observability**: group metrics should integrate into the existing Prometheus surface
5. **Pragmatism**: the codebase already uses package-level initialization and shared singletons, so the groups module should align with that pattern instead of forcing a large DI migration now

### Consequences

- ✅ NIP-29 can be deployed per environment without forking the relay
- ✅ Existing event storage/query model remains intact
- ✅ Redis stays optional and value-oriented
- ⚠️ Group state now exists in both event history and relational tables and must stay synchronized carefully
- ⚠️ Relay-generated metadata events require a configured relay private key when the module is enabled
