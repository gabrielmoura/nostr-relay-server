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

---

## ADR-025: Normalize Relay Keys at Config Boundary

**Status:** Accepted  
**Date:** 2026-04-23

### Context

The relay uses the configured relay keypair in more than one runtime path: NIP-11 metadata exposure, bootstrap flows and NIP-29 relay-signed moderation/state events. Operators often provide keys as NIP-19 values (`npub`, `nsec`) while lower-level runtime code expects raw 32-byte hex.

The previous behavior mixed formats across the codebase and allowed a startup path where NIP-29 attempted to derive a pubkey directly from an `nsec` string, causing a fatal error during server boot. The config validation command also traversed URL checks unsafely and could panic instead of returning a validation error.

### Decision

Normalize relay keys immediately after config load:

1. Accept `relay_information.pub_key` as hex or `npub`
2. Accept `relay_information.priv_key` as hex or `nsec`
3. Store both keys internally as lowercase hex
4. Derive `relay_information.pub_key` from `relay_information.priv_key` when the public key is omitted
5. Reject mismatched public/private keypairs during config load instead of failing later during server initialization

### Consequences

- ✅ Runtime callers receive one canonical key format
- ✅ NIP-29 startup no longer depends on callers remembering the input format details
- ✅ Config validation fails deterministically instead of panicking on malformed relay metadata
- ⚠️ Effective config output now shows normalized hex values instead of the original NIP-19 input strings

---

## ADR-026: Add NIP-86 as an External JSON-RPC Admin Surface

**Status:** Proposed  
**Date:** 2026-04-27

### Context

The relay already exposes an internal admin API protected by `X-Admin-Token`, plus existing runtime structures for bans, connection tracking, Redis cache/pubsub, and NIP-98 verification patterns in the Blossom flow. We need Nostr-native relay management without replacing the current admin panel or rewriting the transport stack.

### Decision

Implement NIP-86 on the same external root route `/` used for WebSocket upgrades, using a strict HTTP branch activated by `Content-Type: application/nostr+json+rpc`.

The design is:

1. `RootUpgrade` becomes a small protocol switch: NIP-11, NIP-86 JSON-RPC, or WebSocket upgrade.
2. A dedicated NIP-98 middleware validates `kind:27235`, signature, freshness, `method`, exact URL, and mandatory `payload` hash.
3. Authorization is limited to one configured relay administrator pubkey (`admin_pubkey`).
4. A focused NIP-86 service layer owns method dispatch and delegates persistence to repository interfaces backed by `pgx`.
5. PostgreSQL remains authoritative for allowlists, blocked IPs, banned events, and relay metadata overrides; Redis is only a hot-path accelerator and disconnect fan-out tool.

### Reasons

1. **Compatibility**: preserves the current root URI and WebSocket flow.
2. **Low-risk evolution**: adds one HTTP branch instead of introducing a second public admin endpoint.
3. **Reuse**: builds on existing listener, cache, pubsub, and moderation infrastructure.
4. **Auditability**: NIP-86 mutations map cleanly to structured JSON logs and relational audit columns.
5. **Operational safety**: `blockip` can immediately drop active sessions while persisting the durable block.

### Consequences

- ✅ Keeps WebSocket and admin management on the standards-compliant root URI.
- ✅ Avoids replacing the existing internal admin panel.
- ✅ Makes admin mutations observable and durable.
- ⚠️ Requires careful route ordering so JSON-RPC does not interfere with WebSocket upgrades.
- ⚠️ Introduces a new config secret/identity boundary (`admin_pubkey`) that must be documented clearly.

---

## ADR-027: Redis Streams Queue for Operational Background Jobs

**Status:** Proposed  
**Date:** 2026-04-28

### Context

Operational jobs are currently split across multiple execution styles:

1. download jobs are stored in memory and launched with unmanaged goroutines,
2. admin sync launches fire-and-forget goroutines with no durable tracking,
3. cron executes work directly in-process without a shared delivery substrate.

This makes retry, dead-letter routing, cross-instance scaling, job visibility and graceful operational recovery harder than necessary.

### Decision

Introduce a Redis-backed operational queue subsystem with these rules:

1. **Redis Streams** become the primary delivery mechanism.
2. **Consumer Groups** provide concurrent worker distribution and ACK semantics.
3. **Lua scripts** are used for atomic enqueue, start, success ACK, retry, dead-letter move, delayed promotion and compact metric/state updates.
4. **Sorted Sets** store delayed and retry-ready jobs.
5. **BITFIELD** stores compact lifecycle state and attempts by sequential job id.
6. **Compact Hashes** store only small metadata fields; job payload stays in `STRING`.
7. **HyperLogLog** tracks approximate active worker uniqueness per time window.

### Reasons

1. **Durability**: job state survives process restarts.
2. **Scalability**: workers can scale horizontally without distributed locks.
3. **Operational clarity**: retries, dead jobs and delayed backlog become observable.
4. **Performance**: Redis-native structures and Lua reduce round-trips and per-job overhead.
5. **Incremental migration**: download, sync and cron can adopt the same substrate one by one.

### Consequences

- ✅ one operational queue model for admin and cron work
- ✅ reliable ACK/retry/dead-letter behavior
- ✅ queue-specific metrics fit the existing Prometheus surface
- ✅ compatible with Redis Cluster key-slot rules via `{queue}` hash tags
- ⚠️ Redis becomes required for queue-backed job execution
- ⚠️ the first rollout must preserve current HTTP/CLI contracts while old and new execution paths coexist briefly

---

## ADR-028: Narrow NIP-29 Validation to Explicit Group Scope

**Status:** Proposed  
**Date:** 2026-04-29

### Context

The first NIP-29 integration reused generic `h`/`d` tag discovery to decide whether an incoming EVENT or REQ belonged to the groups module. That made the write path overly broad: unrelated events carrying those tags were forced through group lookup and rejected with `invalid: group does not exist`.

This behavior breaks the documented requirement that NIP-29 remains optional and must not alter baseline relay behavior for unrelated traffic.

### Decision

Split NIP-29 scope detection by protocol path:

1. **Write path (`EVENT`)**: apply NIP-29 validation only to explicit NIP-29 kinds (`9000`-`9022`, `39000`-`39003`)
2. **Read path (`REQ` / `COUNT`)**: apply pre-query permission validation only when the filter explicitly targets a group through `#h` or asks directly for NIP-29 state kinds
3. **Delivery path**: keep per-event filtering for private and hidden group events as the final safety net, even when the original filter was not group-scoped

### Consequences

- ✅ Unrelated events are no longer rejected because they happen to contain `h`/`d` tags
- ✅ Group read permissions remain enforced for explicit `#h` filters
- ✅ Private and hidden group events stay protected on mixed queries through post-query filtering
- ⚠️ NIP-29 write validation becomes intentionally narrower than raw tag detection, so future group-related kinds must be added explicitly to the helper set

---

## ADR-029: Manage NIP-32 Labels Through the Internal Admin API

**Status:** Proposed  
**Date:** 2026-05-06

### Context

The relay already stores `kind:1985` events, and the product requirement is to add an operator-facing labels management screen inspired by `ref/divine-relay-manager` without reintroducing its browser/worker publishing model.

We need a native relay implementation that:

1. lists stored labels,
2. aggregates them by namespace and target,
3. creates new signed NIP-32 events from the internal admin dashboard.

### Decision

Implement NIP-32 labels management through the internal `/admin` API and the embedded dashboard.

The design is:

1. add `/admin/labels` and `/admin/labels/summary` for server-side reads,
2. add `POST /admin/labels` for signed label creation,
3. keep PostgreSQL `event` as the only source of label truth,
4. sign new label events with `relay_information.priv_key`,
5. keep banning as a separate moderation action using the existing ban endpoints.

### Reasons

1. **Reuse:** label events already fit the existing event storage model.
2. **Low-risk rollout:** no migration is required for first delivery.
3. **Operational consistency:** the dashboard already trusts the internal admin API and token model.
4. **Protocol fidelity:** the stored object remains a real Nostr `kind:1985` event.
5. **Feature coverage:** this route lets us support `e`, `p`, `a`, `r`, and `t` targets cleanly.

### Consequences

- ✅ No duplicate labels table.
- ✅ The admin dashboard stays service-first and browser-safe.
- ✅ Existing stored `kind:1985` data becomes immediately visible in the UI.
- ⚠️ Labels created by the dashboard are authored by the relay admin identity, not by each moderator browser key.
- ⚠️ JSONB tag extraction queries need dedicated tests to avoid regressions in filtering and aggregation.

---

## ADR-030: Treat Sync Cancellation as Terminal and Add Explicit Resume + Backend History Cleanup

**Status:** Proposed  
**Date:** 2026-05-06

### Context

The queue-backed admin jobs model now powers `/download` and `/sync`, but two operational gaps remain:

1. canceled sync jobs may resume implicitly later,
2. history clearing is only a frontend hide operation.

Additionally, negentropy sync work can create excessive remote pressure if multiple jobs target the same relay concurrently.

### Decision

1. sync cancellation becomes a terminal backend state,
2. resuming a canceled sync job requires an explicit new admin action,
3. backend job history deletion is exposed for bounded dashboard cleanup,
4. a strict but configurable per-remote concurrency ceiling is enforced for negentropy jobs,
5. queued sync jobs preserve their normalized filter payload for inspection and reenqueue flows.

### Reasons

1. **Operational trust:** cancel must mean stop.
2. **Remote safety:** per-relay concurrency needs a hard cap.
3. **Auditability:** resume and cleanup become explicit actions.
4. **Usability:** operators need to inspect filters used by old jobs and reenqueue terminal work safely.

### Consequences

- ✅ `/sync` becomes behaviorally consistent with operator intent
- ✅ backend and frontend state stop diverging on canceled jobs
- ✅ remote relays are protected from uncontrolled negentropy fan-out
- ⚠️ queue monitor/worker semantics need implementation work beyond the dashboard layer

---

## ADR-031: Surface Sync Filter and Structured Relay Rejections in Admin Job Details

---

## ADR-032: Split Admin HTTP Handlers and Warm Redis Cache for Admin Search

**Status:** Proposed  
**Date:** 2026-05-12

### Context

`infra/handler/http/admin.go` grew into a monolithic transport file with nearly two thousand lines. At the same time, the admin SPA issues three coupled event-search reads on page load, and the aggregate/timeline endpoints currently rebuild their payloads by loading whole event result sets into Go memory. Under real data volume this pushes latency above two seconds and can trigger dashboard timeouts.

### Decision

Refactor the admin HTTP surface in two coordinated steps:

1. split `infra/handler/http/admin.go` into smaller files by concern while keeping the same package and route contracts
2. move admin event search hot paths to Redis-backed response caching plus SQL-first aggregate and timeline queries

### Detailed shape

1. **Transport decomposition**: keep shared admin helpers in small common files and move users, NIP-05, event search, reports, import/fetch, and shared mappers into focused files so no admin transport file exceeds 300 lines
2. **SQL-first aggregates**: replace Go-side full-scan aggregate and timeline computation with dedicated grouped queries in `infra/db`
3. **Read-through Redis cache**: cache normalized admin search page, aggregates, and timeline responses with versioned invalidation compatible with existing event query cache invalidation
4. **Cron-mode warming**: precompute and store the default dashboard payloads only from the dedicated `cron` process when Redis is enabled

### Reasons

1. **Latency**: the dashboard needs hot responses for its default load path
2. **Memory efficiency**: aggregate and timeline handlers should not materialize all matching events in Go
3. **Maintainability**: the admin HTTP transport must stop being a single oversized file
4. **Low-risk evolution**: route contracts stay stable while internals become cheaper and easier to test

### Consequences

- ✅ faster default admin dashboard load after cron warmup runs
- ✅ lower DB and application CPU cost for aggregates and timeline
- ✅ smaller admin HTTP files with clearer responsibilities
- ✅ cache invalidation stays aligned with the existing Redis query version model
- ⚠️ cold dashboard reads can still happen before the cron process warms the cache
- ⚠️ arbitrary uncached filters still pay a first-hit database cost before entering the cache

---

## ADR-033: Decompose Oversized DB and Redis Cache Files

**Status:** Proposed  
**Date:** 2026-05-12

### Context

After the admin HTTP split, the main remaining oversized backend files are concentrated in `infra/db/admin_query.go`, `infra/db/event_query.sql.go`, and `infra/cache/cache.go`. They mix unrelated concerns such as event writes, streaming, admin analytics, profile lookup, query cache versioning, counters, and document caching. This hurts reviewability and makes it harder to audit SQL and Redis behavior for correctness and performance.

### Decision

Refactor these files by responsibility while keeping the existing package-level API stable:

1. split `infra/db/admin_query.go` into focused admin query files
2. split `infra/db/event_query.sql.go` into focused event read/write/query-cache files
3. split `infra/cache/cache.go` into focused Redis helper files with shared timeout and TTL helpers

### Reasons

1. **Maintainability:** files over 300 lines are hiding multiple concerns
2. **Database clarity:** SQL-heavy methods become easier to review in smaller groups
3. **Redis correctness:** TTL, key naming, and query-version invalidation logic become easier to audit
4. **Low-risk change:** public methods and callers remain unchanged

### Consequences

- ✅ smaller files with clearer concern boundaries
- ✅ easier auditing of parameterized SQL and row handling
- ✅ easier auditing of Redis TTL and key-version behavior
- ⚠️ more files in `infra/db` and `infra/cache`

**Status:** Proposed  
**Date:** 2026-05-08

### Context

Operators using `/panel/sync` can already open the generic job modal, but today the modal only shows raw payload/result blobs. In practice, sync failures are often caused by relay-specific `OK ... false ...` responses, such as moderated-community restrictions, and the current UI does not surface the executed filter clearly enough.

### Decision

1. keep the generic job endpoints unchanged at the route level (`GET /admin/jobs`, `GET /admin/jobs/:jobId`)
2. enrich `sync.negentropy` result payloads with a normalized `filter` field and bounded `rejections[]`
3. keep `last_error` as a compact summary for list cards, while the modal renders structured diagnostics from `result`
4. preserve the original serialized `filter_json` in the queued payload for retry/resume compatibility

### Reasons

1. **Operator clarity:** the modal should answer "what filter did we run?" without forcing raw JSON inspection.
2. **Faster debugging:** structured rejection items reveal which event ids failed and why.
3. **Low-risk rollout:** reuse the existing Redis queue result blob instead of adding new tables or endpoints.

### Consequences

- ✅ `/panel/sync` gains meaningful drill-down diagnostics
- ✅ relay-specific rejections become auditable after the job finishes
- ✅ retry/resume flows keep using the same persisted payload contract
- ⚠️ sync runtime must cap stored rejection details to avoid unbounded result growth

## ADR-037: Negentropy Authentication Uses the Relay Identity Instead of the Admin Identity

**Status:** Proposed  
**Date:** 2026-05-26

### Context

The relay already supports NIP-42 for websocket authentication and can run relay-to-relay Negentropy synchronization. We need an operator-controlled way to restrict Negentropy sessions without introducing a second signing secret path just for sync.

Two candidate identities already exist:

1. `admin_pubkey`, used for operator-facing management APIs such as NIP-86.
2. `relay_information.pub_key`, the runtime relay identity that can already be derived from and signed with `relay_information.priv_key`.

### Decision

Use `relay_information.pub_key` as the sole authorized pubkey when `negentropy_auth=true`.

### Reasons

1. **Operational fit:** relay-to-relay sync is a relay function, so the relay identity is the least surprising principal.
2. **Secret minimization:** the process already owns `relay_information.priv_key`; reusing it avoids introducing an additional admin private key secret just for Negentropy auth.
3. **Trust separation:** `admin_pubkey` remains dedicated to management authorization semantics instead of becoming an overloaded transport identity.

### Consequences

- ✅ the sync CLI can authenticate against protected remote relays using existing runtime key material
- ✅ the websocket gate stays aligned with the relay's published identity
- ⚠️ operators must configure `relay_information.priv_key` whenever they expect outbound sync authentication to work

## ADR-034: Blossom Operations Use a Dedicated Internal Admin Workspace and Queue-Backed Media Pipeline

**Status:** Proposed  
**Date:** 2026-05-12

### Context

The relay already supports public Blossom/NIP-96 uploads and blob delivery, but operational control is still minimal: there is no paginated media browser, no first-class review queue, no uploader quota management, no BUD-04 mirroring workflow, and no durable audit trail for destructive media actions.

At the same time, the repository already has the pieces needed for a safe rollout: internal `/admin/*` transport, Redis-backed background jobs, object metadata persistence, and an embedded React admin dashboard.

### Decision

Implement Blossom management as one coordinated backend feature set behind the internal admin surface.

1. add `/admin/blossom/*` endpoints for overview, objects, review queue, uploader quotas, workers and audit reads
2. keep heavy media work out of request handlers and run it through the existing queue runtime
3. add public BUD-oriented endpoints `PUT /mirror`, `PUT /media` and `HEAD /media`
4. enrich object metadata with blurhash, derivatives, EXIF/privacy state, NIP-94 tags and usage counters
5. mirror critical admin actions into relational audit rows and Nostr `kind:24242` events

### Reasons

1. **Operational safety:** FFmpeg/image processing and purges must be asynchronous and observable.
2. **Protocol fit:** BUD-04, BUD-05 and BUD-08 map naturally onto the existing Blossom object model plus background jobs.
3. **Moderation clarity:** reported or AI-flagged files need a dedicated queue instead of overloading generic event moderation.
4. **Traceability:** storage deletions, purges and quota changes are high-risk actions and need durable audit evidence.

### Consequences

- ✅ the dashboard can manage media without calling public upload routes as an operator API
- ✅ SHA-256 remains the canonical identity across list, mirror and optimization flows
- ✅ derivative generation becomes observable through the same queue model already used elsewhere
- ⚠️ schema growth is non-trivial and needs careful migration sequencing around existing `objects` rows
- ⚠️ mirroring and media optimization increase disk and CPU pressure, so quotas and retention policies must ship together

---

## ADR-036: Blossom Plans Use a Dedicated Child Configuration Screen and Publish Route-Level Prometheus Metrics

**Status:** Proposed  
**Date:** 2026-05-25

### Context

The main Blossom workspace is already dense: browsing, review, users, workers, reports and analytics. Adding detailed quota-plan creation directly into the primary tab set would overload the route and bury important configuration UX behind generic form fields.

At the same time, Blossom now behaves like a standalone HTTP subsystem and needs explicit Prometheus visibility beyond the generic upload/download counters, especially for rejected or unauthenticated requests.

### Decision

1. add a dedicated child screen `/blossom/plans` for named plans, default assignments and quota modeling
2. keep the main `/blossom` route operational, and treat plans as a configuration drill-down
3. add Blossom-specific Prometheus counters and latency histograms with normalized route labels
4. categorize Blossom HTTP errors so auth and policy failures are visible independently from generic 5xx failures

### Reasons

1. **UX clarity:** plan design needs more explanation and editing space than a compact policy card can offer.
2. **Operator safety:** named plans reduce repeated raw-byte editing and accidental misconfiguration.
3. **Observability:** unauthenticated/rejected Blossom traffic is operationally important and should be queryable in Prometheus.

### Consequences

- ✅ the main workspace stays focused on operations, while deep quota modeling gets its own space
- ✅ MB/GB explanatory affordances can be designed without compromising the main route density
- ✅ Blossom auth failures become measurable, not just log-visible
- ⚠️ the backend needs an additional plan catalog surface instead of only one singleton policy record

---

## ADR-035: Blossom Upload Policy Is Runtime-Managed with Mode-Specific Default Quotas

**Status:** Proposed  
**Date:** 2026-05-15

### Context

The Blossom server now needs three operator-selectable behaviors:

1. every upload must be approved before public visibility
2. only enabled users may upload
3. free uploads for any authenticated user

At the same time, quota handling must be different depending on the selected mode. Free mode needs a default per-user plan or unlimited behavior, while enabled-user mode needs a different default plan plus optional per-pubkey overrides.

### Decision

Implement one effective Blossom server policy persisted behind the admin API.

1. keep per-pubkey overrides in `blossom_pubkey_quotas`
2. add a singleton policy record for upload mode and default plans
3. evaluate policy on every upload and mirror ingestion
4. when the mode is `mandatory_review`, new uploads start blocked until explicit admin approval

### Reasons

1. **Operator control:** the admin UI needs to change upload behavior without redeploying config files.
2. **Predictability:** per-mode defaults are easier to reason about than hard-coding mixed quota rules.
3. **Safety:** `mandatory_review` needs a first-class enforcement point, not only a UI convention.

### Consequences

- ✅ upload authorization and quota evaluation become explicit and inspectable
- ✅ free-mode and allowlist-mode can have different default plans
- ✅ moderation-required mode can safely accept uploads without immediately publishing them
- ⚠️ uploads now depend on one additional policy lookup path
- ⚠️ object approval must clear any temporary public-download block consistently

---

## ADR-036: BUD-04 Mirroring and BUD-08 Metadata Stay Queue-Backed and Extraction-Driven

**Status:** Proposed  
**Date:** 2026-05-16

### Context

The relay needs public `PUT /mirror` support, but remote downloads are untrusted and can stall the HTTP worker, inflate memory use or be abused for SSRF-style traffic amplification if performed inline. At the same time, mirrored files and locally optimized files must expose one canonical NIP-94 metadata shape instead of accumulating ad hoc JSON fragments over time.

### Decision

1. `PUT /mirror` only validates the signed Blossom authorization event (`kind:24242`) and request payload, then enqueues background work
2. the mirror worker performs the remote fetch and computes SHA-256 over the full body before any local persistence
3. a hash mismatch aborts the job and leaves no stored object behind
4. BUD-08 `nip94_tags` are generated only from extracted media facts produced by the BUD-05 routine, for both uploads and mirrored files
5. mirror/source URLs are persisted separately in `mirrors` and projected into repeated NIP-94 `fallback` tags during regeneration

### Reasons

1. **Abuse resistance:** queueing avoids tying remote fetch latency to public HTTP request latency.
2. **Correctness:** full-stream SHA-256 validation is the canonical acceptance gate for BUD-04.
3. **Consistency:** one regeneration path prevents drift between upload, mirror and media optimization metadata.
4. **Operational visibility:** failures and retries stay observable through the existing worker monitor.

### Consequences

- ✅ public mirroring reuses the current queue runtime instead of adding a parallel executor
- ✅ NIP-94 output becomes deterministic across uploads, mirrors and derivatives
- ✅ mirror origin URLs remain available without polluting canonical blob identity
- ⚠️ the public BUD-04 route returns an asynchronous queued response instead of a fully materialized descriptor in the first HTTP round-trip

---

## ADR-037: BUD-05 Uses Immediate Original Persistence Plus Queue-Backed Optimization

**Status:** Proposed  
**Date:** 2026-05-16

### Context

BUD-05 requires `PUT /media` and `HEAD /media`, plus heavy media processing such as image conversion, thumbnailing, blurhash generation and audio/video metadata extraction. Running these operations inline in the HTTP handler would increase latency, risk request timeouts and make abuse amplification easy.

### Decision

1. `PUT /media` stores the original uploaded body immediately and returns its blob descriptor in the first response
2. all heavy optimization steps run in the existing background jobs runtime against the stored original hash
3. `HEAD /media` uses the original hash as the canonical probe key and exposes derivative readiness through headers
4. audio/video inspection uses `ffprobe` and derivative generation uses `ffmpeg` via `os/exec`
5. image resizing and fallback transforms use `github.com/disintegration/imaging` in the first rollout to avoid adding a mandatory native `libvips` dependency
6. media worker features are toggled by explicit runtime flags under `store.media_processing`
7. defaults favor metadata extraction and lightweight preview UX (`blurhash` and image thumbnails on), while heavier video thumbnails and streaming manifests start disabled

### Reasons

1. **Latency control:** clients get a fast response with the stored canonical hash while heavy work continues asynchronously.
2. **Operational safety:** one queue model keeps retries, failures and worker visibility consistent with mirror jobs.
3. **Deployment pragmatism:** `imaging` avoids forcing `libvips` on every environment while still covering thumbnail generation and image fallback operations.
4. **Operational control:** media processing cost varies widely; feature flags must let operators disable expensive steps per environment.
5. **Protocol consistency:** the original object remains the anchor for later mirror and NIP-94 distribution flows.

### Consequences

- ✅ `PUT /media` becomes a trusted upload+optimize surface aligned with BUD-05
- ✅ image and video metadata feed the existing BUD-08 regeneration flow
- ✅ `HEAD /media` becomes a cheap readiness probe for clients and the dashboard
- ✅ operators can trade CPU/storage cost against richer derivatives without changing client contracts
- ⚠️ optimized derivatives may lag behind the first successful `PUT /media` response
- ⚠️ production environments that want HLS/DASH must provision `ffmpeg` binaries and enable the feature explicitly

---

## ADR-038: Optional Marmot MIP-00 Relay Module

**Status:** Proposed  
**Date:** 2026-05-20

### Context

The relay already stores arbitrary Nostr events and already implements generic replaceable and addressable semantics. Marmot `MIP-00` requires support for `kind:30443` KeyPackage events and `kind:10051` relay-list events, but most of the full protocol remains a client-side MLS responsibility.

We need an implementation path that is disabled by default, highly configurable, small in scope, and honest about the level of compatibility it provides.

### Decision

Implement Marmot `MIP-00` as an **optional relay-side module** with configuration-first activation and phased validation depth.

The design is:

1. Add a new `marmot` config block with nested `mip00` toggles.
2. Keep phase 1 storage in the existing `event` table.
3. Reuse the current addressable replacement path for `kind:30443` and replaceable behavior for `kind:10051`.
4. Add a focused event validator in `internal/policies` for `kind:30443`, `kind:10051`, and optional legacy `kind:443`.
5. Start with `basic` validation only: required tags, allowed values, relay URL shape, and payload bounds.
6. Reserve `strict` validation for a later phase that can parse MLS payloads and verify `KeyPackageRef` plus credential identity.

### Reasons

1. **Docs-first compatibility**: the project can describe exactly what it supports before introducing any code or external claims.
2. **Low-risk integration**: the relay already has the correct storage and replacement primitives.
3. **Operational safety**: disabled mode remains inert and existing relays do not change behavior accidentally.
4. **Pragmatism**: relay-side validation should stay at the Nostr boundary until the project adopts a mature MLS dependency.
5. **Future evolution**: the config model leaves room for later `MIP-01` to `MIP-03` work without prematurely coupling the relay core to MLS internals.

### Consequences

- ✅ Minimal first implementation surface: config, policies, tests, docs
- ✅ No database migration required for phase 1
- ✅ Honest compatibility statement: relay-aware `MIP-00`, not full Marmot
- ⚠️ `strict` mode cannot be claimed until a reliable MLS parser/validator exists in Go for this project
- ⚠️ Optional legacy `kind:443` support increases compatibility surface and should stay off by default

---
