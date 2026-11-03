# GraphQL Admin Audit

Status: implemented baseline, subscriptions pending
Scope: `/admin/graphql`, the admin panel under `/panel/`, and the REST handlers
it currently reuses.

## Executive summary

The admin GraphQL API is a single executable schema served by gqlgen over HTTP.
Most resolvers still delegate to Fiber handlers through an in-process adapter
(`executeAdminRequest`) instead of calling the database or domain services
directly. That kept the GraphQL rollout small, but it duplicates validation,
serialization, and authorization, and it makes every field resolution pay for an
HTTP request, a Fiber app, and a JSON round trip.

This iteration removed the highest-cost duplication in the Blossom plan path,
added direct resolvers for connection reads, exposed a browser-safe Blossom
preview URL, and replaced the REST plan adapter with a transactional service.
The remaining structural work is tracked at the end of this document.

## Request architecture

```text
Browser /panel/
  -> Apollo Client (HTTP link, X-Admin-Token)
  -> POST /admin/graphql
  -> admin auth middleware
  -> gqlgen transport
  -> Resolver
       -> direct DB / domain service            (preferred)
       -> executeAdminRequest -> Fiber handler  (legacy adapter)
```

Queries and mutations use HTTP. Subscriptions are not exposed yet; the target
transport is `graphql-transport-ws` on the same endpoint, with the admin token
sent in the connection payload rather than the URL.

Authorization is currently a static admin token checked by
`infra/handler/http/admin_core.go`. The GraphQL HTTP route is behind the same
middleware, so resolvers can assume an authenticated operator. WebSocket
connections need an equivalent check during `connection_init`.

## Inventory

The schema declares 69 fields across `Query`, `Mutation`, and `Subscription`.
The largest domains are:

| Domain | Representative fields | Current implementation |
| --- | --- | --- |
| Overview / presence | `adminOverview`, `activeConnections`, `authedConnections`, `loggedUsers` | Mixed: connection reads use listener state directly; overview still uses the REST adapter |
| Events | `events`, `eventDetail`, `eventAggregates`, `eventTimeline`, reports | REST adapter |
| Labels | `labels`, `labelsSummary`, `createLabel` | REST adapter |
| Blossom | policy, plans, objects, users, reports, workers, audit, analytics | Plans now use `internal/blossom`; objects use direct DB reads; the rest still use the adapter |
| Jobs / sync | `jobs`, `job`, `startNegentropySync`, `downloadEvents`, retry/cancel/resume | REST adapter plus Redis queue |
| Groups / NIP-86 / WoT | group reads, allow/block lists, trusted pubkeys | REST adapter |

Current subscriptions:

```graphql
type Subscription {
  blossomPlanChanged: AdminBlossomPlan!
  blossomObjectChanged: AdminBlossomObject!
}
```

They are declared in the schema but not served over WebSocket, and the existing
in-process broker in `graph/subscriptions.go` overwrites subscribers that use
the same id, closes channels while publishers may still hold them, and has no
authorization or per-subscription filtering.

## Diagnostic findings

### P0 - correctness and lifecycle

1. **No WebSocket transport.** gqlgen is mounted with `handler.NewDefaultServer`
   on an HTTP route. GraphQL subscriptions cannot run.
2. **Unsafe broker lifecycle.** Subscriber channels are stored under a single id
   key, so two operators watching the same plan replace each other. Publishing
   holds a read lock while sending, and cancellation closes a channel that a
   concurrent publisher may still write to.
3. **Preview URLs were unusable in private networks.** Blossom objects returned
   the configured media origin directly. When that origin is `.onion`, `.i2p`,
   or otherwise unreachable from the operator's browser, thumbnails and previews
   failed even though the relay can proxy the same blob.
4. **Missing generated field.** `AdminBlossomObject.previewURL` existed in the
   schema but not in the model or generated executor.

### P1 - architecture and performance

1. **REST adapter in the hot path.** `executeAdminRequest` creates a Fiber app,
   performs a fake HTTP request, decodes JSON, and re-maps it to GraphQL models.
   This is acceptable as a migration seam, not as the final implementation for
   high-traffic fields.
2. **Duplicate mapping and validation.** REST response structs and GraphQL model
   structs carry the same fields, so new fields must be added twice and can drift.
3. **Presence polling.** The panel refreshes presence and job data on timers
   because there is no push channel.
4. **Cache disabled.** `adminApolloClient` uses `fetchPolicy: "no-cache"` for
   queries and watch queries. This avoids stale data but prevents normalized
   cache updates and makes subscription write-through impossible without an
   explicit cache strategy.
5. **Static error banner.** The app shell renders a permanent presence refresh
   error instead of reflecting the actual subscription or query state.

### P2 - consistency and observability

1. **No uniform pagination contract.** Some domains use offset pages, others use
   ad-hoc limits and totals.
2. **No resolver-level metrics.** GraphQL latency is only visible through the
   generic HTTP middleware.
3. **No subscription metrics.** There is no count of active subscriptions,
   dropped events, or reconnect attempts.

## Implemented in this iteration

- Added `internal/blossom/urls.go` and `BrowserURL(hash)` to resolve previews
  through the relay's `/blob/:id` proxy.
- Populated `AdminBlossomObject.previewURL` from the direct DB resolver and the
  REST mapper. Thumbnail and optimized URLs also use the browser-safe proxy.
- Updated the frontend object type, normalization, queries, detail sheet, and
  library grid to prefer `preview_url` and keep `direct_url` as metadata.
- Added the direct `UpsertBlossomPlan` service and transactional default-plan
  handling in `internal/blossom/plan.go` and `infra/db/blossom_plan_query.go`.
- Added direct connection resolvers in `graph/admin_queries_core.go`.
- Regenerated gqlgen models and executor code for `previewURL`.
- Added structured validation errors for Blossom plans.

## Target subscription architecture

```text
Domain transaction / job state change
  -> publish event after commit
  -> in-process event bus (bounded, non-blocking)
  -> optional Redis Pub/Sub bridge (ephemeral fan-out only)
  -> GraphQL subscription resolver
  -> graphql-transport-ws
  -> Apollo cache update
```

Rules:

- Publish only after the transaction commits or the job transition is durable.
- Redis Pub/Sub is fan-out, not a source of truth. A reconnect must be able to
  recover from a normal query.
- Every subscriber gets its own bounded channel. Slow consumers are dropped and
  counted; they must never block the producer.
- Subscriptions are authorized at `connection_init` and filtered per operator.
- Tokens are sent in the connection payload, never in the URL.

Planned subscription surface:

| Subscription | Producer | Payload |
| --- | --- | --- |
| `adminOverviewChanged` | listener/connection lifecycle | compact overview counters |
| `connectionsChanged` | listener connect/disconnect/auth | connection event or invalidate signal |
| `jobChanged` | Redis queue worker and tracker | job id, queue, status, progress, error |
| `syncProgress` | relay sync runtime | job id, stage, counters, done flag |
| `downloadProgress` | download runtime | job id, current/total, done flag |
| `blossomPlanChanged` | Blossom plan service | full plan |
| `blossomObjectChanged` | Blossom review/processing | full object or hash plus state |

`jobChanged`, `syncProgress`, and `downloadProgress` should share one job event
envelope so the frontend can reconcile queue state and progress without polling.

## Frontend cache strategy

The panel currently uses a small React-Query-compatible layer over imperative
Apollo calls (`src/hooks/query-compat.ts`). It must be extended before
subscriptions are wired:

1. Add a query-key-aware `setQueryData`/`updateQueryData` operation to the
   compat client.
2. Keep normalized Apollo cache entries for entities that subscriptions mutate
   (`AdminBlossomObject`, `AdminJob`, plans), and switch only those operations
   to a cache-aware fetch policy.
3. On a subscription event, write the entity once, then patch list pages and
   counters in place. Do not invalidate the whole domain.
4. Reconnect with exponential backoff, re-run one normal query on reconnect, and
   resume the subscription. Do not replay missed events.
5. Tear down the socket when the panel unmounts or the token changes.

## Performance opportunities

1. Replace `executeAdminRequest` with domain services in this order: jobs,
   events, labels, Blossom users/reports, NIP-86, groups.
2. Add dataloader-style batching for profile and author lookups in event pages.
3. Reuse one Fiber app per process for any remaining adapter calls instead of
   allocating one per resolver invocation.
4. Add persistence for selected GraphQL fields (offset pages, filters) so a
   reconnect does not reset the operator's view.
5. Measure p50/p95 resolver latency before and after each adapter removal.

## Test plan

No new tests were added in this iteration by design. Future coverage should be:

- Unit: broker fan-out, slow-consumer drop, unsubscribe, duplicate subscribers.
- Unit: `BrowserURL` and preview selection for public, `.onion`, and `.i2p`
  media origins.
- Integration: `graphql-transport-ws` handshake with valid, missing, and
  invalid admin tokens.
- Integration: publish after commit (plan upsert) and after durable job
  transition; verify no event is emitted on rollback.
- Integration: Redis bridge with two relay instances.
- Frontend: subscription event updates the detail sheet and list page without a
  refetch; reconnect restores the subscription exactly once.
- Frontend: filters and pagination survive a reconnect.

## Priorities

**P0**

- Implement the WebSocket transport and a safe bounded event bus.
- Replace the current broker before enabling any subscription in production.
- Keep preview URLs browser-safe.

**P1**

- Add job and presence event producers.
- Add compat cache write-through and remove presence/job polling.
- Remove the static app-shell error banner.

**P2**

- Continue replacing the REST adapter domain by domain.
- Add resolver and subscription metrics.
- Standardize pagination and error payloads.
