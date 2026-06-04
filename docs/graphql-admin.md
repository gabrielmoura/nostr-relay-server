# Admin GraphQL Migration Spec

## Status

- Phase: documentation only
- Implementation status: not started
- Source of truth for the proposed contract: `docs/graphql-admin-schema.graphqls`
- Split SDL files: `docs/graphql-admin-schema-scalars.graphqls`, `docs/graphql-admin-schema-types.graphqls`, `docs/graphql-admin-schema-operations.graphqls`
- Current production transport is GraphQL on `/admin/graphql`

## Goal

Expose the internal admin surface only through GraphQL, backed by `gqlgen` and served by the internal `GoFiber` server.

## Endpoint Proposal

- `POST /admin/graphql`
- `GET /admin/graphql/schema` for SDL exposure in authenticated internal mode
- `GET /admin/graphql/playground` for authenticated internal exploration against the internal admin graph

## Authentication

- Preserve the current `X-Admin-Token` middleware behavior from the internal admin server
- The GraphQL endpoint must live on the internal server only
- All GraphQL operations inherit the current admin authorization boundary; no browser-side NIP-98 signing is introduced
- `schema` and `playground` must stay behind the same admin token boundary

## Development Surfaces

- `GET /admin/graphql/schema` returns the mounted GraphQL schema for internal tooling and frontend development
- `GET /admin/graphql/playground` serves an internal GraphQL IDE pointed to `/admin/graphql`
- both routes exist for operator and developer ergonomics only and are not public relay features

## Transport Status

- `/admin/graphql` is the only exposed admin API surface
- legacy `/admin/*` REST routes are intentionally not mounted anymore
- the backend may still reuse internal admin handler logic behind resolvers, but that logic is not exposed as HTTP REST API

## Design Rules

- Keep the existing persistence model; GraphQL is a transport replacement, not a storage rewrite
- Keep current admin semantics for normalization of `pubkey`, `event id`, `npub`, `note`, `nevent`, `nprofile`, and `naddr`
- Preserve queue-backed async behavior for sync, download, mirror, purge, and job operations
- Normalize response envelopes where GraphQL can remove REST-only ceremony
- Prefer typed fields for stable business data and `JSON` only for runtime snapshots or highly irregular payloads

## Domain Coverage

The proposed schema covers these admin domains already exposed by REST:

- dashboard overview and stream status
- active/authed connections and disconnect action
- logged, banned, searched, and detailed users
- manual NIP-05 administration
- event search, aggregates, timeline, detail, report drill-down, and fetch-from-relays
- reported events analytics from NIP-56 data
- NIP-32 labels list, summary, and creation
- Blossom admin operations: policy, plans, objects, users, reports, workers, analytics, audit
- NIP-86 admin datasets: allowlist, blocked IPs, banned events, relay metadata
- operational jobs, negentropy sync, and event downloads
- NIP-29 groups listing and WoT summary/trusted roots management

## Transport Mapping

REST pagination currently uses `limit` and `offset` on many endpoints. The GraphQL contract keeps offset pagination through `OffsetPageInput` and returns a shared `PageInfo` block:

```graphql
input OffsetPageInput {
  limit: Int = 100
  offset: Int = 0
}

type PageInfo {
  total: Int!
  limit: Int!
  offset: Int!
  hasMore: Boolean!
}
```

This is intentionally conservative so the admin SPA can migrate without changing its scrolling and virtualized list model first.

## GraphQL Conventions

- Queries expose reads only
- Mutations expose every admin write and async trigger
- `Upload` is reserved for multipart JSONL event import
- `Int64` and `JSON` must be declared as explicit gqlgen scalars during implementation
- `NostrTag.values` preserves variable-length tag arrays without flattening protocol semantics
- in the first implementation pass, RFC3339 timestamps are exposed as `String` instead of a dedicated GraphQL time scalar

## Normalization Policy

- Public-key and event-id inputs may accept NIP-19 forms at the resolver boundary when the REST handlers already do that normalization today
- Persisted values remain canonical hex or native DB forms
- Relay hints remain optional and are only attached where existing REST behavior already supports them

## Resolver Strategy

The GraphQL layer should wrap the current admin services and query packages instead of re-implementing the business rules.

Expected implementation shape after approval:

- `infra/net/router.go` mounts `/admin/graphql` under the existing admin middleware
- `graph/` stops being the sample todo schema and becomes the admin schema package
- the first implementation pass uses an internal in-memory Fiber bridge to the existing admin handlers to preserve current behavior and reduce migration risk
- targeted direct DB access is still used where the REST surface does not return enough data for the GraphQL contract

## REST To GraphQL Mapping

| REST surface | GraphQL field |
|---|---|
| `GET /admin/overview` | `adminOverview` |
| `GET /admin/stream/status` | `adminStreamStatus` |
| `GET /admin/connections/active` | `activeConnections` |
| `POST /admin/connections/:wsid/disconnect` | `disconnectConnection` |
| `GET /admin/users/*` | `loggedUsers`, `bannedUsers`, `searchUsers`, `userProfile`, `userBanStatus` |
| `GET/POST/DELETE /admin/nip05*` | `nip05Identities`, `userNip05`, `upsertNip05`, `deleteNip05` |
| `GET /admin/events/search*` | `events`, `eventAggregates`, `eventTimeline` |
| `GET /admin/events/:id*` | `eventDetail`, `eventReports`, `fetchEventFromRelays` |
| `GET /admin/events/reported*` | `reportedEvents`, `reportedEventsSummary` |
| `GET/POST /admin/labels*` | `labels`, `labelsSummary`, `createLabel` |
| `GET/PUT/POST/DELETE /admin/blossom/*` | `blossom*` queries and mutations |
| `GET/POST/DELETE /admin/nip86/*` | `nip86*` queries and mutations |
| `POST /admin/sync/negentropy` | `startNegentropySync` |
| `POST /admin/events/download` | `downloadEvents` |
| `GET/POST/DELETE /admin/jobs*` | `jobs`, `job`, `retryJob`, `cancelJob`, `resumeJob`, `deleteJobsHistory` |
| `GET /admin/groups` | `groups` |
| `GET/POST/DELETE /admin/wot/*` | `wotSummary`, `addTrustedPubkey`, `removeTrustedPubkey` |

## Scalars Required In gqlgen

| Scalar | Purpose |
|---|---|
| `Int64` | event counters, timestamps, byte sizes |
| `JSON` | job payload/result, runtime stats, irregular analytics blocks |
| `Upload` | JSONL event import |

## Known Non-Goals For This Phase

- no external/public GraphQL endpoint
- no federation
- no schema stitching
- no breaking storage migrations
- no replacement of NIP-86 JSON-RPC on the public relay root

## MCP Relevance For This Design

- `nostr`: relevant now, because NIP-19, NIP-32, NIP-56, and NIP-86 shape the admin contract
- `postgres`: relevant during implementation validation and resolver tests; current docs and live schema already align for this design
- `redis`: relevant during implementation for cached search responses, queue-backed jobs, and job observability; not needed to redefine the GraphQL contract itself

## Approval Gate

If this specification is approved, the next step is to replace the sample GraphQL schema under `graph/` with the approved admin SDL and then wire `gqlgen` into the internal Fiber router.
