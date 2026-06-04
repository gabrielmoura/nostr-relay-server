# Frontend Architecture

## Overview

The admin dashboard (`infra/dash/`) is a React 19 + TypeScript SPA built with TanStack Router and i18next. It provides operational controls for the Nostr relay server.

## Planned GraphQL Frontend Migration

The admin dashboard will migrate from the current REST service layer to the internal GraphQL surface exposed by the relay backend.

Target transport:

- `POST /admin/graphql`
- authenticated internal helpers for development: `GET /admin/graphql/schema` and `GET /admin/graphql/playground`

Target frontend data stack:

- **Apollo Client 4.x** for GraphQL transport, cache, and mutations
- **TanStack Router** remains the route and URL-state layer
- the current compact operations-dashboard language remains intact

This is a data-layer refactor, not a visual redesign.

Implementation note for the first migration pass:

- existing route-facing hooks in `use-admin-data.ts` remain in place temporarily
- those hooks now sit on top of a GraphQL-backed service layer instead of calling REST directly
- this keeps route churn low while removing the admin REST dependency from the dashboard

## Planned NIP-86 Dashboard Extension

The dashboard will remain on the internal admin API and will not call the external NIP-86 JSON-RPC endpoint directly from the browser.

Reasons:

- keeps the admin browser flow aligned with the existing `X-Admin-Token` trust model
- avoids exposing NIP-98 signing responsibilities to the SPA
- reuses the current typed `services/admin.ts` integration pattern
- lets the backend translate UI actions into the same persistence/runtime side effects used by NIP-86

The internal admin API should gain dedicated endpoints for:

- allowed pubkeys
- banned events
- blocked IPs
- relay metadata overrides

## Planned NIP-32 Labels Dashboard

The dashboard also needs a dedicated labels workspace for `kind:1985` events.

### Product goal

- inspect existing labels already stored on the relay;
- create new labels from the internal admin UI;
- support NIP-32 targets `e`, `p`, `a`, `r`, and `t`;
- normalize applicable NIP-19 input into canonical target values before mutation;
- optionally chain a pubkey ban after a successful label mutation.

### API strategy

The SPA must stay on the internal admin API and must not publish labels directly over browser WebSocket connections.

Planned service functions in `services/admin.ts`:

- `getLabels(filters)`
- `getLabelsSummary(filters)`
- `createLabel(payload)`

Optional ban chaining keeps using the existing ban service:

- `banUser(payload)`

### Visual direction

Using `ui-ux-pro-max`, we keep the existing compact operations-dashboard language and apply only the useful structural guidance:

- **Pattern:** data-dense + drill-down
- **Layout:** KPI strip + filter bar + dual content views
- **Interaction:** row highlight, compact chips, quick actions, strong empty/error states
- **Typography:** current dashboard typography, with monospaced treatment only for ids/namespaces/targets
- **Important:** preserve the current dashboard palette and avoid a generic purple-heavy visual reset

## Visual Direction

Using `ui-ux-pro-max`, the recommended direction for the new moderation area is a data-dense operational dashboard:

- **Pattern:** data-dense + drill-down
- **Style:** compact KPI cards, dense tables, low-friction filters, clear status badges
- **Palette:** security blue base with green success accents and pale blue surfaces
- **Typography:** `Fira Code` for headings and `Fira Sans` for body copy
- **Interaction:** row highlighting, compact toolbars, fast filter feedback, visible focus states

This should refine the current dashboard instead of replacing it with a marketing-style interface.

## Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Framework** | React 19 | UI library with new hooks (useActionState, useOptimistic) |
| **State Management** | Apollo Client 4.x | GraphQL server state, cache, fragments, and mutations |
| **Routing** | TanStack Router | File-based routing with type-safe navigation |
| **i18n** | i18next | Internationalization (English/Portuguese) |
| **Build** | Vite | Fast development and optimized production build |
| **UI** | Radix UI primitives | Accessible component foundation |
| **Styling** | Tailwind CSS | Utility-first styling |

## Project Structure

```
infra/dash/src/
├── main.tsx                 # App entry point
├── router.tsx              # TanStack Router configuration
├── App.tsx                 # Root component with Error Boundary
├── locales/                # i18n translation files
│   ├── en.json
│   └── pt-BR.json
├── lib/                    # Utility layer (adapters, parsers)
│   ├── event-parser.ts    # Event tag parsing (imeta, media, references)
│   ├── event-search.ts    # Search result parsing
│   └── router.ts          # Navigation hooks
├── services/               # API integration layer (future)
├── components/
│   ├── ui/                 # Base UI primitives (Button, Card, Dialog...)
│   ├── shared/             # Shared components (PageHeader, MetricCard...)
│   ├── features/          # Feature-specific components
│   │   ├── event-detail/
│   │   ├── event-search/
│   │   └── ban-user-dialog/
│   └── layout/             # App shell and layout components
└── routes/                 # Page components (TanStack Router)
    ├── event-detail-page.tsx
    ├── event-search-page.tsx
    ├── overview-page.tsx
    ├── sync-page.tsx            # Negentropy sync controls
    ├── download-page.tsx        # Bulk event download
    ├── groups-page.tsx          # NIP-29 management
    ├── wot-page.tsx             # Web of Trust / Trusted Pubkeys
    └── ...
```

## Design Patterns

### Smart vs. Dumb Components

- **Dumb (Visuals)**: Pure presentational components in `components/ui/` and `components/shared/`. Receive data via props, emit events via callbacks. No API calls, no business logic.

- **Smart (Containers)**: Route components in `routes/` and feature components in `components/features/`. Handle state, call APIs, orchestrate data flow.

### Separation of Concerns

```
┌─────────────────────────────────────────────────────────────┐
│                      Route (Smart)                          │
│  - Fetches data                                             │
│  - Manages state                                            │
│  - Orchestrates sub-components                              │
└─────────────────────────┬───────────────────────────────────┘
                          │ props / callbacks
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Feature Components (Smart/Dumb)                │
│  - Composes visual blocks                                  │
│  - Transforms data for presentation                        │
└─────────────────────────┬───────────────────────────────────┘
                          │ props
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              UI Components (Dumb)                           │
│  - Pure rendering                                           │
│  - No business logic                                        │
└─────────────────────────────────────────────────────────────┘
```

### Service Layer and GraphQL Access

All backend communication must continue to go through a typed integration layer:

```
┌─────────────┐     ┌──────────────────┐     ┌────────────────┐
│  Component  │────►│ GraphQL module   │────►│ Apollo Client  │
│             │     │ (typed docs +    │     │ + /admin/graphql │
│             │     │ adapters)        │     │                │
└─────────────┘     └──────────────────┘     └────────────────┘
```

Rules for the migration:

- route-level smart components own page queries and mutations
- dumb components receive already-shaped props and callbacks only
- GraphQL documents must be colocated by route or feature, but transport setup stays centralized
- `services/admin.ts` must stop being a REST fetch bag and become a typed GraphQL adapter boundary or be split by domain into GraphQL modules
- no route component should call `fetch` directly

Preferred structure after approval:

```text
infra/dash/src/
  graphql/
    client.ts
    errors.ts
    fragments/
    queries/
    mutations/
    adapters/
  hooks/
    use-admin-graphql.ts
```

Current implementation state:

- `src/graphql/client.ts` provides the shared Apollo client
- `src/graphql/admin-api/` now splits the admin transport by domain:
  - `core.ts`
  - `users.ts`
  - `events.ts`
  - `nip05-nip86.ts`
  - `jobs-wot.ts`
  - `blossom.ts`
  - `index.ts`
- `src/hooks/use-admin-data.ts` now exposes the existing route-facing hook API through an Apollo-backed compatibility layer, reducing route churn during migration
- `src/graphql/documents.ts` is the source-of-truth for frontend GraphQL operations used by codegen
- `infra/dash/codegen.ts` and `pnpm graphql:codegen` generate typed operation artifacts in `src/graphql/generated/operations.ts`

## Apollo Client Strategy

- `ApolloProvider` at app root, alongside Router and current global providers
- one page query per route where practical, composed from colocated fragments
- mutations via `useMutation`
- typed adapters convert GraphQL camelCase payloads into route-friendly view models when needed
- current mock fallback behavior should be isolated behind the GraphQL adapter layer instead of leaking into route components
- `x-request-id` must be extracted from GraphQL transport failures and surfaced through typed frontend errors

Incremental adoption rule:

- while Apollo is now the GraphQL client, the first pass may keep existing TanStack Query hooks as route-facing orchestration wrappers until route-by-route Apollo hook migration is complete
- the current implementation has already removed direct TanStack Query usage from `use-admin-data.ts`; remaining route compatibility is preserved through local hook adapters instead of React Query primitives

## Error Handling and Boundaries

- current global error boundaries remain
- Apollo transport and GraphQL errors must be normalized into one frontend error shape
- request-id propagation remains mandatory for debugging admin failures
- route-level retries should use Apollo refetch or mutation retry UX, not ad-hoc fetch retries

## Smart vs Dumb Impact

No visual component should become GraphQL-aware.

- Smart: route pages, modal containers, board/workspace containers
- Dumb: cards, rows, forms, charts, badges, summaries, lists

The migration must not push GraphQL hooks into dumb components.

## Persistence Strategy

Current dashboard persistence survey (`infra/dash/src`):

- `zustand` + `localStorage` already back:
  - `stores/media-player-store.ts`
  - `stores/geohash-search-store.ts`
  - `stores/reported-events-store.ts`
- manual `localStorage` is still used in `lib/relay-presets.ts`
- i18n language detection also uses `localStorage`, which should remain external to app-state stores

Decision for the next refinement:

- keep **small UI/session preferences** in `zustand` persisted to `localStorage`
- migrate **manual relay preset persistence** into a dedicated `zustand` store instead of raw helper writes
- use **IndexedDB** only for operator data that is materially larger or benefits from structured client-side caching/history, such as Blossom mirror submission history or larger route-scoped cached drafts
- do **not** move TanStack Query server-state into `localStorage` persistence by default

Practical rule:

- `localStorage`: booleans, small arrays, route/workspace preferences, recent compact identifiers
- `IndexedDB`: larger lists, structured operator history, recoverable drafts that may outgrow safe `localStorage` usage
- URL state: active filters that should remain shareable/bookmarkable

## Planned NIP-86 Feature Modules

New feature components should be grouped under:

```text
components/features/nip86/
  allowed-pubkeys-panel.tsx
  banned-events-panel.tsx
  blocked-ips-panel.tsx
  relay-metadata-form.tsx
```

Suggested route split:

```text
routes/
  nip86-page.tsx                # overview / command center
  nip86-allowed-page.tsx        # allowlist management
  nip86-banned-events-page.tsx  # event moderation state
  nip86-blocked-ips-page.tsx    # network blocking and disconnect actions
```

## Planned Labels Feature Module

Suggested split:

```text
components/features/labels/
  labels-workspace.tsx          # smart orchestrator used by the route
  labels-help-dialog.tsx        # dumb/help content dialog
  labels-stats-strip.tsx        # dumb KPI cards
  labels-filter-bar.tsx         # dumb URL-driven filters
  labels-timeline.tsx           # dumb event list
  labels-targets-table.tsx      # dumb aggregated-by-target view
  create-label-dialog.tsx       # smart mutation container
  label-form-fields.tsx         # dumb form body
  label-category-picker.tsx     # dumb category + custom label selector
```

Planned route:

```text
routes/
  labels-page.tsx
```

## Related operational refinements

- `JobsBoard` should support a clear-history interaction for noisy operational queues like `/download` and `/sync`
- `/events/reported` and `/users/search` should expose compact KPI strips derived from the current filtered dataset
- `/events/$eventId` should become a richer moderation workspace by combining event detail with labels, reports, reply authors and associated event references
- `NostrFilterBuilder` should normalize NIP-19 or hex input consistently across search and operational forms
- canceled sync jobs must remain terminal until an operator explicitly resumes them
- completed jobs should expose a reenqueue/retry action from the board when the backend already supports safe retry semantics
- `/events/search` should move KPI cards above filters and surface kind `34550` metadata inline

## Planned Rich Event Visualization

### Scope

This refinement targets:

- `/panel/events/search`
- `/panel/events/$eventId`
- `components/features/event-detail/*`
- `components/features/event-search/*`

### Product goal

Operators should understand media-heavy and non-text events without opening raw JSON.

Required interpretation rules from the current relay domain:

- `kind:6` must be rendered as a repost of the referenced event from NIP-18.
- `kind:4550` must be rendered as a NIP-72 community approval, highlighting community pointer, approved event, approved kind and embedded approved payload when present.
- `kind:10050` must be rendered as a NIP-51/NIP-17 DM relay list, prioritizing relay badges and counts over empty content.
- any event displayed as `(sem conteudo textual)` must append its `alt` tag when present, using the exact operator-facing pattern `(sem conteudo textual) ${alt}`.
- `kind:1111` should expose the associated community name or identifier when an `a`/`A` tag points to a `kind:34550` community.
- when the associated community can be resolved, `/panel/events/search` should also show the community thumbnail and a visible community-context badge.
- when the event is `kind:1111` and belongs to a community, `/panel/events/search` should keep the community indicator and also render a compact preview of its textual content plus associated tags when present.
- media-only events must render media from `imeta`, `r`, `url`, content URL and recognized MIME metadata instead of showing only raw links.

### Visual direction

Using `ui-ux-pro-max`, preserve the existing admin dashboard language and apply these feature-specific rules:

- data-dense moderation layout, not a marketing layout
- compact semantic cards for protocol-specific kinds
- `kind:4550` in `/panel/events/search` should surface richer moderation context than a generic badge-only summary
- community-aware events in `/panel/events/search` should render one compact context strip with thumbnail + semantic badge before the protocol-specific body
- `K:1111` in `/panel/events/search` should expose a tooltip with the event-kind description sourced from NIP-22 / kind metadata
- constrained media containers with `min-w-0`, `overflow-hidden` and capped viewport heights to prevent page blowout
- carousel only when there is more than one asset of the same post context
- video should be click-to-load in search results to avoid auto-heavy rendering in long lists
- keep light/dark contrast high and avoid layout shifts on hover

### Component strategy

Route containers remain smart:

- `routes/event-search-page.tsx`
- `routes/event-detail-page.tsx`

Feature visual blocks remain dumb or near-dumb:

- `EventSearchItem`
- `EventMedia`
- `MediaCarousel`
- `CommunityApprovalCard`
- `DMRelayListCard`
- new protocol cards and media summary blocks where needed

Light adapters and parsers live in `lib/`:

- extend `event-parser.ts` for MIME-aware media extraction, alt fallback rules and specialized protocol summaries
- keep route components responsible for query orchestration only

### `@nostrify/*` usage plan

The dashboard already depends on `@nostrify/nostrify` and `@nostrify/react` but does not consume them yet.

Planned usage:

- add a small Nostr provider boundary near the app root or event feature boundary
- use `@nostrify/react` hooks to resolve referenced events and author context needed by richer repost/community visualization without coupling UI components to raw websocket logic
- keep admin REST endpoints as the primary source of truth for indexed data; `@nostrify/*` is an auxiliary read layer for protocol-native enrichments only

This preserves the service-first admin architecture while finally using the installed Nostr client stack in a controlled way.

### Layout safety rules for `/panel/events/$eventId`

- `PanelGroup` panels must allow shrinking but all inner content containers must use `min-w-0`
- long identifiers, URLs and tags must render with `break-all` or `truncate` depending on context
- media panels must cap height around viewport units and avoid unconstrained width growth
- multi-card sections below the split panel must use responsive grids that collapse cleanly on narrow widths
- no event content block may force horizontal scroll on the page
- protocol cards, metadata chips and nested preview blocks must not create horizontal overflow for long `a`, `e`, `p`, relay and media URLs, including the current problematic detail case `/panel/events/625b9578996ccccd0cc381b7f133a4dba94f3a7aae8851a6734f3870a24c6621`

## Planned Event Search Analytics Modal

### Scope

This refinement targets:

- `/panel/events/search`
- `routes/event-search-page.tsx`
- new `components/features/event-search/*` analytical modal blocks

### Product goal

Operators need a dedicated analytical surface for event search without leaving the main search route. The route should expose a button that opens a modal and renders event-search charts related to the current active filters.

The modal should answer:

- top-line KPI values for the current filtered relay dataset
- dominant kinds in the current filtered relay dataset
- event activity over time with month/year-visible labels
- most active authors, including resolved display names when possible
- most common tags
- trend-oriented insights such as top tag in the month/year when the aggregate contract supports them

### Visual direction

Using `ui-ux-pro-max`, the modal should feel like a compact operations analytics workspace:

- opened from a clear button in the page header actions
- analytical cards arranged in a dense modal grid
- KPI strip should sit at the top of the modal before the charts
- reuse `recharts` components already aligned to the dashboard language
- keep charts compact and readable inside the modal; no oversized hero charts
- preserve the existing search route as the primary surface and keep the modal as a drill-in analytics layer
- clicking a kind or tag in the modal should refine the modal charts to the chosen selection
- author surfaces in the modal should support both filtering and navigation to the user detail route

### Component strategy

Smart orchestration stays in `routes/event-search-page.tsx`.

New dumb modal-focused components may include:

- `EventSearchAnalyticsModal`
- `EventSearchAnalyticsSummary`
- `EventSearchAnalyticsKpiStrip`
- `EventSearchTopAuthorsChart`
- `EventSearchTopTagsChart`
- reuse of existing `EventSearchAggregates` and `EventSearchTimeline` where possible

### State strategy

- modal open/close can remain local route state
- analytics data continues to come from TanStack Query hooks already tied to the active search filters
- the modal should support a relay-overview mode where aggregates are not interpreted as list-local counters
- the modal should support opening on a specific initial analytical tab when triggered by the route
- if the modal later gains persisted view preferences, they can move into the existing Zustand global UI layer

### Reported events routing strategy

The `/panel/events/reported` route must now be URL-driven through TanStack Router.

Rules:

- `q`, `type`, `targetEventId`, `targetPubkey`, `since` and `until` should be reflected in route search params
- chart-driven filtering must update the URL, not only local/global UI state
- restoring the URL must restore the moderation slice exactly

### Labels analytics modal strategy

`/panel/labels` should expose a secondary analytics modal with KPIs and Recharts-based charts derived from `labels/summary`.

Goals:

- keep `LabelsWorkspace` primary for operations
- open analytics on demand from a header action
- surface namespace, label and target-type distributions without leaving the route

## Planned Report Analytics Workspace

### Scope

This refinement targets the existing moderation route:

- `/panel/events/reported`
- `routes/reported-events-page.tsx`
- new `components/features/reported-events/*`

### Product goal

Operators should move from a report list with static KPI cards to an analytical moderation workspace that answers these questions immediately:

- how many reported events and total report events exist in the current filter slice
- which report types dominate the current dataset
- when report activity spikes over time
- which authors or targets accumulate the most moderation pressure

The protocol context comes from NIP-56 (`kind:1984`) and should be treated as moderation telemetry, not only as individual row items.

### Visual direction

Using `ui-ux-pro-max`, the reports page should stay inside the existing admin dashboard language and become a compact analytics screen:

- KPI strip built with `recharts`, not plain static cards
- one trend chart for report volume over time
- one categorical chart for report types
- optional author/target concentration chart when the dataset is large enough
- filters stay above charts so every chart reflects the active moderation slice
- charts must remain compact, readable and responsive, not marketing-sized hero graphics

### Component strategy

Smart route/container:

- `routes/reported-events-page.tsx`
- optional `ReportedEventsWorkspace` smart orchestrator if the route becomes too dense

Dumb analytical components:

- `ReportedEventsKpiStrip`
- `ReportedEventsTrendChart`
- `ReportedEventsTypeChart`
- `ReportedEventsTopAuthorsChart`
- `ReportedEventsSummaryCard`

Existing list and modal behavior remain, but become secondary drill-down surfaces below the analytics summary.

### API strategy

The reports analytics workspace now requires **global server-backed aggregates**.

The list route and the analytics route must be separated conceptually:

- `/admin/events/reported` continues to return paginated target rows for the virtualized list
- a new summary endpoint must return analytics computed from the full filtered dataset on the server, independent of the currently loaded list slice

Planned summary endpoint:

- `GET /admin/events/reported/summary`

The dashboard must not derive KPI or chart totals from the virtualized list when global moderation telemetry is required.

### Global state strategy

This refinement also introduces a lightweight global client-state layer using:

- `zustand`
- `immer`
- persistence/sync via `localStorage`

Planned use cases:

- reports dashboard filters and chart-driven drill-down state
- persisted analytics view preferences
- reusable geohash search input state if promoted beyond one route
- media/player preferences that belong to the operator session, not a single component subtree

Rules:

- TanStack Query remains the source of truth for server state
- Zustand stores only global UI/session state
- persistence is explicit and scoped; do not dump entire fetched datasets into localStorage

### Geohash search strategy

The dashboard now also needs geohash-aware search support using `ngeohash`.

First-pass scope:

- normalize geohash input for search/filter workflows
- decode geohash labels for operator context where useful
- support exact geohash search and prefix-based grouping when the route domain allows the Nostr `g` tag

This aligns with the Nostr `g` tag used by NIP-52 and other location-aware events.

### Video player strategy

The current native `<video>` rendering path should be upgraded to:

- `@vidstack/react`
- `hls.js`
- `dashjs`

Goals:

- support progressive video plus HLS/DASH manifests in one operator-grade player
- preserve lazy load behavior in search results
- keep the player wrapper dumb and driven by parsed media metadata
- maintain layout safety and avoid autoplay-heavy behavior in virtualized contexts
- keep Vidstack debug/log output enabled while this player migration is being verified in the admin dashboard

### Search analytics modal strategy

The search analytics modal must reflect the current **full filtered relay dataset**, not only the items loaded in the virtualized list.

Rules:

- KPI values at the top of the modal must come from server-backed aggregates
- modal charts must be tied to the same aggregate/timeline sources already computed for the full filtered dataset
- `loadedItems` from the current virtualized list may be shown only as a secondary local indicator, never as the primary global total

## Current known gap

The current sync queue UX exposes `cancel`, but the backend/runtime behavior can still let a canceled item resume automatically later. The intended fix is:

- preserve `canceled` as a stable terminal state
- add an explicit `resume` action for canceled jobs
- keep the frontend board aligned with real backend semantics instead of simulating cancellation locally

For labels, the intended target-value behavior is:

- `pubkey` targets accept `hex`, `npub` and `nprofile`
- labeling a profile is a first-class supported workflow through `target.type = pubkey`
- API traffic should remain normalized to canonical hex before submission

## Planned Relay Workflow Refinement

The dashboard now needs a shared relay-selection UX for operational screens that query external relays.

Shared frontend rules:

- relay lists must be managed through a reusable modal, not raw comma-separated text only
- the modal must support:
  - selecting from common relays
  - adding one relay at a time
  - importing a comma-separated relay list in one action
  - removing already-added relays
- the chosen relay list must be persisted in `localStorage` and reused by:
  - `/download`
  - event-detail relay search flows
  - future relay-driven screens

Implementation split:

- `lib/relay-presets.ts` - localStorage adapter + normalization helpers
- `components/shared/relay-list-modal.tsx` - reusable modal UI
- route/feature containers remain responsible for passing current relays and handling submit actions

## Planned Download Job UX

The current `/download` page starts a background backend process but does not expose meaningful progress or completion states to the user.

Refinement strategy:

- introduce backend-backed download jobs with in-memory runtime status
- start action returns a `job_id`
- frontend polls job status and renders a work queue
- work queue should expose:
  - pending / running / completed / failed state
  - filter summary
  - relay count and timeout
  - result summary (`events_received`, `inserted_events`, `duplicate_events`, `pages`)
  - “Ver filtros” and “Ver detalhes” actions

This keeps the visual queue truthful to real backend execution instead of simulating success after the request returns.

## Planned Generic Operational Jobs UX

The queue backend now supports durable jobs for download, sync and cron. The dashboard should stop treating these as feature-specific background toasts and instead present them through one shared operational jobs experience.

### UX direction applied from `ui-ux-pro-max`

Because this lives inside an existing admin dashboard, the new UI should reuse the current visual language instead of switching to a radically different theme. We will only borrow the useful parts of the generated system:

- **Pattern:** real-time monitoring
- **Layout:** dense operational cards + drill-down table
- **Typography emphasis:** keep existing app typography, but use monospace treatment for job ids, queues and timing values
- **Feedback:** compact status chips, subtle activity pulse for running jobs, strong error panels for terminal failures
- **Avoid:** fake progress bars, noisy glow effects, or marketing-style hero layouts

### Architecture refinement

The generic jobs flow should be split into four frontend layers:

1. **Service layer** in `services/admin.ts`
   - `getJob(jobId)`
   - `getJobs(filters?)`
   - `retryJob(jobId)`
   - `cancelJob(jobId)`

2. **Types layer** in `types/admin.ts`
   - `AdminJob`
   - `AdminJobStatus`
   - `AdminJobListResponse`
   - `AdminJobResult`

3. **TanStack Query hooks** in `hooks/use-admin-data.ts`
   - `useJobsQuery`
   - `useJobQuery`
   - `useRetryJobMutation`
   - `useCancelJobMutation`

4. **Feature UI** in `components/features/jobs/`
   - reusable queue board, cards, details drawer/dialog and action toolbar

### Screen strategy

Instead of adding one more isolated page immediately, the first rollout should integrate the generic jobs board into the existing operational routes:

- `/download` shows the shared jobs board filtered to `job_name=download.events`
- `/sync` shows the shared jobs board filtered to `job_name=sync.negentropy`
- a later follow-up may add `/jobs` as a global operator command center once cron and dead-letter workflows need broader inspection

### Error handling strategy

- route-level error states stay in the route containers
- query failures render inline operational panels instead of relying only on toast messages
- retry/cancel mutations must surface backend `x-request-id` when available through the shared `ApiError`
- dialogs/drawers remain dumb; mutation and polling logic stays in smart containers/hooks
- sync detail dialogs must prioritize structured `job.result` diagnostics over raw `job.last_error`
- the sync modal must render the executed filter in a dedicated panel instead of burying it only inside raw payload JSON

### React 19 decision

This feature should continue to use **TanStack Query** as the primary server-state mechanism because:

- jobs are long-lived server state, not simple form submissions
- polling, targeted invalidation and mutation side effects are already standardized in the dashboard
- `useActionState` would add ceremony without helping cache coordination here

## Error Handling

### Error Boundaries

| Level | Location | Purpose |
|-------|----------|---------|
| App | `App.tsx` | Catches fatal errors, shows recovery UI |
| Route | Route components | Feature-specific error states |
| Component | Risky components | Isolated failure containment |

### Error Recovery Flow

1. Error Boundary catches the error
2. Fallback UI displayed with error message
3. User can retry (resets the component tree)
4. Error is logged (console for now, future: sentry/logging service)

### x-request-id Propagation

Backend responses include `x-request-id` header. Currently not propagated to frontend. Future: capture and display in error messages for traceability.

For the labels workspace this becomes mandatory on mutation failures because operators may need to trace rejected label creation or chained ban actions.

## Sync Modal Refinement

Specific UX rules for the `/sync` details modal:

- keep `JobsBoard` as the only smart component; no new API calls move into dialog leaf components
- add one dedicated filter panel that prefers parsed `job.result.filter`, then falls back to `job.payload.filter`, `job.payload.request.filter`, or `job.payload.filter_json`
- add one diagnostic panel for `sync.negentropy` that renders:
  - aggregated result error message
  - per-event rejection rows (`event_id`, `reason`)
  - optional raw relay frame for copy/paste debugging
- keep the existing generic raw payload/result JSON panels for deep inspection, but make them secondary to the curated sync panels

## State Management

### Local State

- React `useState` for form inputs and UI state
- TanStack Router for URL state (query params, path params)

### Server State

Adopted **TanStack Query** (v5) for all administrative data fetching and mutations.

- **Queries**: Dashboard metrics, user lists, event search, NIP-29 groups, WoT summary.
- **Mutations**: Sync start, download start, group moderation, WoT trusted pubkeys management.
- **Caching**: Automatic background refetching and cache invalidation after mutations.

### State Patterns

```tsx
// Loading state
const [isLoading, setIsLoading] = useState(false);

// Error state  
const [error, setError] = useState<Error | null>(null);

// Form state
const [formData, setFormData] = useState(initialData);
```

## React 19 Usage

### useActionState (Forms)

Use for form submissions with pending/success/error states:

```tsx
const [state, formAction, isPending] = useActionState(
  async (prev, formData) => {
    // submit logic
    return result;
  },
  initialState
);
```

### useOptimistic (Feedback)

Use for immediate UI feedback before server confirmation:

```tsx
const [optimisticValue, setOptimistic] = useOptimistic(
  serverValue,
  (state, newValue) => newValue
);
```

### When NOT to Use

- Don't use React 19 hooks just for fashion
- Apply when they genuinely improve:
  - Form handling clarity
  - User feedback immediacy
  - Code readability

## Internationalization

### Structure

```
locales/
├── en.json      # English (default)
└── pt-BR.json   # Portuguese (Brazil)
```

### Usage

```tsx
const { t } = useTranslation('namespace');
return <span>{t('key')}</span>;
```

### Keys Pattern

`category.element.action` (e.g., `eventDetail.metadata.show`)

## Build and Deployment

```bash
pnpm install    # Install dependencies
pnpm dev        # Development server
pnpm build      # Production build
pnpm preview    # Preview production build
```

The built assets are embedded into the Go binary via `embed.FS` in `infra/dash/dist/`.

## Conventions

### File Naming

- **Components**: `kebab-case.tsx` (e.g., `event-detail-page.tsx`)
- **Utilities**: `kebab-case.ts` (e.g., `event-parser.ts`)
- **Types/Interfaces**: PascalCase in same file or `types.ts`

### Props Interface Pattern

```tsx
interface MyComponentProps {
  title: string;
  onSubmit: (data: Data) => void;
  isLoading?: boolean;  // optional, with default
}

export function MyComponent({ title, onSubmit, isLoading = false }: MyComponentProps) {
  // ...
}
```

### Import Order

1. React/Next imports
2. External libraries
3. Internal components
4. Local utilities
5. Types

```tsx
import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { Button } from '@/components/ui/button';
import { EventCard } from '@/components/features/event-detail/event-card';
import { parseEvent } from '@/lib/event-parser';
import type { Event } from '@/lib/types';
```

## Planned Blossom Admin Workspace

### Product goal

- give operators one dense operational route for media inventory, review, quotas, mirroring, worker health and audit trail
- preserve the current admin visual language instead of introducing a separate product shell
- make media-heavy workflows legible with thumbnails, blurhash placeholders and a side inspection drawer
- add fast-access overlays for workers and analytics without forcing a tab switch
- expose BUD-09 blob reports as a first-class moderation surface
- add a deeper plans/quotas management screen for default plans and named quota presets
- move `review`, `reports` and `audit` into lower-level child screens so the main Blossom route stays lighter

### Visual direction

Using `ui-ux-pro-max`, keep the current compact admin palette and apply these route-specific rules:

- **Pattern:** analytics-first operational dashboard with media drill-down
- **Layout:** KPI strip -> alert rail -> filter/policy bar -> compact hub tabs -> child-route drill-downs -> side inspection sheet + modal overlays
- **Views:** table/list for dense auditing and optional thumbnail grid for browsing
- **Accessibility:** all icon-only actions need labels; quick actions must stay keyboard accessible; danger actions require confirmation
- **Performance:** grid previews use reserved aspect ratios, lazy thumbnails and blurhash-first placeholders to avoid layout shift

For the dedicated plans screen, apply a more focused configuration UX while preserving the dashboard language:

- **Pattern:** configuration cockpit with plan cards + detailed editor panel
- **Layout:** summary rail -> default mode selector -> plan grid -> sticky detail form
- **Hierarchy:** named plans first, low-level byte values second
- **Tooltip rule:** storage size fields show a help icon with a tooltip explaining MB/GB meaning and default-plan consequences
- **Interactions:** quick duplicate/edit/set-default actions, explicit empty/unlimited states, strong destructive confirmations for deletion

### API strategy

The SPA continues to call only the internal admin API via `services/admin.ts`.

Planned service functions:

- `getBlossomOverview()`
- `getBlossomPolicy()`
- `updateBlossomPolicy(payload)`
- `getBlossomPlans()`
- `upsertBlossomPlan(payload)`
- `deleteBlossomPlan(id)`
- `getBlossomObjects(filters)`
- `getBlossomObjectDetail(hash)`
- `reviewBlossomObjects(payload)`
- `getBlossomUsers(filters)`
- `getBlossomUserDetail(pubkey)`
- `upsertBlossomWhitelistEntry(payload)`
- `purgeBlossomUser(pubkey)`
- `createBlossomMirrorJob(payload)`
- `getBlossomWorkers(filters)`
- `getBlossomReports(filters)`
- `resolveBlossomReport(payload)`
- `getBlossomAnalytics(filters?)`
- `getBlossomAudit(filters)`

### Suggested module split

```text
components/features/blossom/
  blossom-workspace.tsx
  blossom-plans-page.tsx
  blossom-review-page.tsx
  blossom-reports-page.tsx
  blossom-audit-page.tsx
  blossom-kpi-strip.tsx
  blossom-alert-rail.tsx
  blossom-filters-bar.tsx
  blossom-view-toggle.tsx
  blossom-objects-table.tsx
  blossom-objects-grid.tsx
  blossom-object-sheet.tsx
  blossom-analytics-dialog.tsx
  blossom-review-queue.tsx
  blossom-bulk-actions-bar.tsx
  blossom-users-table.tsx
  blossom-plan-modal.tsx
  blossom-delete-plan-modal.tsx
  blossom-assign-plan-modal.tsx
  blossom-mirror-panel.tsx
  blossom-workers-board.tsx
  blossom-workers-dialog.tsx
  blossom-reports-table.tsx
  blossom-report-sheet.tsx
  blossom-audit-table.tsx
```

Planned route:

```text
routes/
  blossom-page.tsx
  blossom-plans-page.tsx
  blossom-policy-page.tsx
  blossom-review-page.tsx
  blossom-reports-page.tsx
  blossom-audit-page.tsx
```

Interaction rules for this follow-up:

- the header `Workers` button opens `BlossomWorkersDialog` even when the current tab is not `workers`
- the MIME filter becomes an editable combobox: operators can pick a known MIME or type an arbitrary MIME string
- the uploader filter accepts name, display name, `nip05`, `npub`, or hex pubkey and remains URL-driven
- `BlossomObjectSheet` gets a `Copiar Blossom ID` action that emits a BUD-10 URI
- `BlossomAnalyticsDialog` shows charts and operational summaries without replacing the main tabbed workspace
- `reports`, `audit` and `review` become lower-level Blossom routes linked from the main workspace instead of living in the primary tab strip
- `review` navigation is only rendered when the effective Blossom policy indicates manual review is enabled
- plans/quotas configuration becomes a lower-level route under Blossom, linked from the main workspace as a drill-down instead of another heavy top-level tab
- the overview tab shows only a read-only policy summary with a link to `/blossom/policy`
- `BlossomUsersTable` owns TanStack Table state, backend sorting and `@tanstack/react-virtual` pagination
- `/blossom/plans` no longer embeds policy controls; it is dedicated to plan catalog, modal CRUD and plan-to-user association
