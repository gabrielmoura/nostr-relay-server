# Frontend Decisions

## ADR-F011: Admin Dashboard Migrates From REST + TanStack Query To Internal GraphQL + Apollo Client

**Status:** Proposed  
**Date:** 2026-06-03

### Context

The admin dashboard currently consumes many internal REST endpoints through `services/admin.ts` and `use-admin-data.ts`, with server-state orchestration handled by TanStack Query. The backend now has an internal GraphQL admin surface, making the current REST fan-out unnecessarily verbose for route screens that combine overview, lists, aggregates, modals, and mutations.

The route structure, visual language, and URL-state design are already good enough and do not need a UI reset.

### Decision

1. keep **TanStack Router** for routing and URL-state
2. adopt **Apollo Client 4.x** as the admin GraphQL client and remove REST as the dashboard transport
3. keep the current route-facing hook surface temporarily so migration can be incremental
4. keep dumb components transport-agnostic
5. colocate GraphQL fragments and operations by route or feature container
6. move request normalization, error normalization, and request-id propagation into a GraphQL integration layer

### Reasons

1. **Transport fit:** the backend admin contract is now GraphQL-first
2. **Composition:** route screens can fetch exactly the fields they need without multiplying REST helpers
3. **Boundary clarity:** Apollo becomes the single GraphQL client while current hooks can be migrated route by route
4. **Low visual risk:** route tree and UI primitives remain intact while only the data layer changes

### Consequences

- ✅ fewer transport-specific REST helpers in the dashboard
- ✅ route queries can be composed from colocated fragments
- ✅ the admin graph becomes the single source of truth for frontend admin data
- ⚠️ TanStack Query may remain temporarily as an orchestration layer until route hooks are fully migrated
- ⚠️ current mock fallback behavior must be re-homed into GraphQL adapters or development-only tooling

---

## ADR-F009: Rich Event Visualization Uses Protocol-Aware Cards and One Shared Media Interpreter

**Status:** Proposed  
**Date:** 2026-05-08

### Context

The admin dashboard already exposes event search and event detail, but many Nostr events are media-first or protocol-descriptive rather than text-first. Today the operator often sees empty content, raw links or large unconstrained media blocks. This is especially problematic for `kind:6`, `kind:4550`, `kind:10050`, NIP-68 picture posts and text notes with multiple media assets in `imeta`.

The project already includes `react-resizable-panels`, `recharts`, `embla-carousel-react`, `@nostrify/nostrify` and `@nostrify/react`, but the event visualization flow does not fully exploit them yet.

### Decision

1. keep the existing route structure for `/panel/events/search` and `/panel/events/$eventId`
2. centralize media extraction and protocol summaries in `lib/event-parser.ts`
3. use protocol-aware visual cards for `kind:6`, `kind:4550` and `kind:10050`
4. use direct render for a single image and carousel for multiple images or mixed media in the same post context
5. keep videos lazy in search results via click-to-load previews
6. use `@nostrify/react` only for optional read enrichment of referenced events, not as the primary admin data source

### Reasons

1. **Operator speed:** empty-text events become readable without raw JSON inspection.
2. **Consistency:** search and detail stop interpreting media differently.
3. **Safety:** lazy video rendering avoids heavy feeds and noisy autoplay behavior.
4. **Architecture fit:** the admin API remains authoritative while Nostr-native enrichment stays isolated.

### Consequences

- ✅ better moderation ergonomics for community approvals, reposts and DM relay lists
- ✅ shared parser logic reduces duplicated heuristics across components
- ✅ event detail becomes more resilient against overflow and media blowout
- ⚠️ frontend complexity in media parsing increases and must stay well-contained in adapters
- ⚠️ `@nostrify/*` enrichment must remain optional so detail pages still work when remote lookups fail

## ADR-F004: Generic Operational Jobs Board Reuses Existing Routes

**Status:** Proposed  
**Date:** 2026-04-28

### Context

The backend now exposes one shared queue model for download, sync and cron work. The dashboard currently treats download jobs with feature-specific cards and treats sync only as a toast-triggered background action. We need a frontend model that matches the durable backend job system without fragmenting the operator experience.

### Decision

Use one reusable jobs module and embed it into the existing operational routes first.

1. `/download` shows generic jobs filtered to `download.events`
2. `/sync` shows generic jobs filtered to `sync.negentropy`
3. a dedicated global `/jobs` route is deferred until there is clear operator demand

### Reasons

1. **Low-risk migration:** preserves familiar entry points for operators
2. **Reuse:** one board, one dialog, one mutation model for retry/cancel
3. **Truthful UX:** the UI reflects the backend queue lifecycle instead of per-feature simulations
4. **Docs-first scalability:** future cron/dead-letter views can reuse the same contracts

### Consequences

- ✅ download and sync converge on one jobs presentation model
- ✅ service layer and hooks stay compact and typed
- ✅ route-level orchestration stays aligned with current Smart/Dumb rules
- ⚠️ `/download` and `/sync` pages become denser and require careful visual hierarchy
- ⚠️ cron jobs will still be less visible until a broader `/jobs` route is added later

---

## ADR-F005: NIP-32 Labels Use a Dedicated Admin Route and Backend-Signed Mutation Flow

**Status:** Proposed  
**Date:** 2026-05-06

### Context

The labels experience described in the reference project depends on browser-side Nostr queries and a separate publishing worker. The admin dashboard in this repository already standardizes on typed internal API calls plus TanStack Query.

We need a labels screen that fits the existing dashboard architecture and can create NIP-32 events without exposing relay signing to the browser.

### Decision

Add a dedicated `/labels` route in `infra/dash` and consume backend-signed admin endpoints.

1. `LabelsPage` stays the smart route entrypoint
2. `LabelsWorkspace` is the only feature-level smart container
3. all rendering blocks remain dumb and reusable
4. label creation uses TanStack Query mutation, not browser-side Nostr publish
5. optional ban chaining reuses the existing ban mutation after label creation succeeds

### Reasons

1. **Consistency:** matches the rest of the dashboard service layer.
2. **Traceability:** preserves `x-request-id` and server validation errors.
3. **Security:** keeps relay private-key usage on the backend.
4. **Reusability:** the same dialog can later appear in reported events and user detail flows.

### Consequences

- ✅ labels gain a native admin experience instead of a bolt-on external UI
- ✅ Smart/Dumb separation stays explicit
- ✅ server state remains cacheable and invalidatable through TanStack Query
- ⚠️ label creation cannot be optimistic because the signed event id is only known after backend success

---

## ADR-F006: Treat Sync Cancel as Terminal and Resume as Explicit Operator Action

**Status:** Proposed  
**Date:** 2026-05-06

### Context

The generic jobs board now exposes cancel actions on `/sync`, but the observed runtime behavior allows some canceled items to resume automatically later. This breaks operator trust because the UI communicates a terminal action while the backend queue may still continue work.

### Decision

1. sync cancel must be treated as terminal state in both backend and frontend semantics
2. a separate explicit `resume` action must be introduced for canceled sync jobs
3. `NostrFilterBuilder` inputs should accept either hex or NIP-19 when the domain concept allows both, reducing operator friction across search and moderation flows

### Reasons

1. **Trust:** operators need cancellation to mean stop
2. **Auditability:** resume becomes an intentional new action instead of hidden queue behavior
3. **Consistency:** NIP-19-aware inputs align operational tools with how Nostr operators actually copy identifiers

### Consequences

- ✅ sync job lifecycle becomes easier to reason about
- ✅ frontend actions map more honestly to backend state
- ✅ filter and target fields become friendlier to real Nostr workflows
- ⚠️ backend queue semantics may need changes beyond the dashboard layer

---

## ADR-F007: Expand Operator Inputs and Rich Event Context Instead of Requiring Canonical Raw Data

**Status:** Proposed  
**Date:** 2026-05-06

### Context

Operators naturally work with copied Nostr identifiers like `npub`, `note`, `nevent`, `nprofile` and `naddr`, not only raw hex. They also need richer context for community events (`kind:34550`) and queued sync work without drilling into raw JSON every time.

### Decision

1. `NostrFilterBuilder` should accept NIP-19 or hex wherever the concept allows both
2. labels target input keeps normalizing profile/event identifiers before submission
3. `/events/search` and `/events/$eventId` should render `kind:34550`-specific metadata (`d`, `description`, `image`, moderators)
4. sync job cards/details should expose the filters used and allow reenqueue/resume paths explicitly

### Reasons

1. **Operator ergonomics:** matches real Nostr workflows
2. **Moderation speed:** reduces hops to inspect relevant metadata
3. **Safety:** normalization still preserves canonical backend contracts

### Consequences

- ✅ search, sync and labels inputs become easier to use
- ✅ community events become more interpretable in moderation flows
- ✅ job actions become more auditable and explicit
- ⚠️ frontend adapters and backend contracts must stay tightly aligned on normalization rules

---

## ADR-F008: Sync Job Modal Uses Curated Panels for Filter and Relay Rejections

**Status:** Proposed  
**Date:** 2026-05-08

### Context

The generic jobs dialog already exposes raw payload/result JSON, but operators reviewing sync failures need two answers immediately: which filter was executed and which relay rejections happened. Requiring manual JSON inspection slows triage and hides important diagnostics in noisy blobs.

### Decision

1. keep the generic dialog shell in `JobsBoard`
2. add curated sync-only panels inside the modal for executed filter and rejection diagnostics
3. keep raw payload/result JSON panels as secondary, still-visible debug surfaces
4. prefer structured backend data from `job.result` over reconstructing meaning from `last_error`

### Reasons

1. **Speed:** the operator sees the important sync context first.
2. **Safety:** the UI reflects persisted backend truth instead of re-deriving diagnostics heuristically.
3. **Reuse:** the generic jobs route structure remains intact while only the modal body gains sync-specific branches.

### Consequences

- ✅ `/panel/sync` gets a clearer operational drill-down without a new route
- ✅ raw JSON remains available for deep debugging
- ⚠️ sync-specific modal content must stay isolated so `/download` and other jobs do not inherit irrelevant UI noise

---

## ADR-F009: Event Search Headlines Must Stay Operator-Explicit and Community-Aware

**Status:** Proposed  
**Date:** 2026-05-08

### Context

The first rich-event refinement improved media and protocol cards, but operators still need stricter textual cues in search results. Two gaps remain important in moderation workflows:

1. empty-text events must visibly preserve the exact `(sem conteudo textual) ${alt}` pattern when `alt` exists
2. NIP-72/NIP-22 community-related events should expose the associated community context, especially `kind:4550` approvals and `kind:1111` posts/replies tagged with a `34550:*` community address

There is also a known residual horizontal overflow case on `/panel/events/$eventId` for some long protocol payloads.

### Decision

1. keep the explicit `(sem conteudo textual) ${alt}` headline pattern for empty-text events
2. enrich `kind:4550` search cards with more community moderation context
3. resolve the associated community identifier/name for `kind:1111` when an `a`/`A` tag targets `kind:34550`
4. harden detail-page protocol cards and wrappers against residual horizontal overflow

### Reasons

1. **Moderation clarity:** operators need to identify non-text events quickly without opening JSON.
2. **Community context:** NIP-72 workflows are much easier to understand when the community is visible inline.
3. **Layout safety:** long protocol identifiers must never break the admin page structure.

### Consequences

- ✅ search results become more explicit for empty-text and community-scoped events
- ✅ `kind:1111` community posts gain meaningful inline context
- ✅ detail pages are more resilient to long identifiers and protocol payloads
- ⚠️ community resolution may sometimes rely on fallback identifiers when richer metadata is unavailable

For the current follow-up, community-aware list rows should also expose a visible badge and thumbnail when resolved, because text-only context proved too subtle for operators in dense search results.

---

## ADR-F010: Reported Events Route Becomes an Analytics-First Moderation Workspace

**Status:** Proposed  
**Date:** 2026-05-08

### Context

The current `/panel/events/reported` route already exposes filters, a KPI trio and a list of reported targets, but the KPI layer is static and visually secondary. Operators need faster pattern recognition around NIP-56 report pressure, report types and time spikes without reading each list row.

The frontend already ships `recharts`, so the route can be upgraded without adding a new charting dependency.

### Decision

1. keep the existing `/panel/events/reported` route
2. turn it into an analytics-first moderation workspace
3. replace the static KPI strip with Recharts-based compact KPI cards
4. add at least two analytical charts in the first pass:
   - report volume trend over time
   - report type distribution
5. keep the existing list and modal as drill-down layers below the analytics section

### Reasons

1. **Operator speed:** anomalies and dominant report categories become visible immediately.
2. **Low-risk change:** preserves the current route and service contracts.
3. **Architectural fit:** aggregation can be derived client-side from the current fetched moderation slice.

### Consequences

- ✅ moderation telemetry becomes readable at a glance
- ✅ `recharts` is used for real operational value, not decorative charts
- ✅ current row-level workflow remains intact
- ⚠️ first-pass analytics are based on the fetched slice, not a dedicated full-dataset aggregate endpoint

---

## ADR-F011: Reports Analytics, Geohash Search and Media UX Need Shared Global UI State

**Status:** Proposed  
**Date:** 2026-05-08

### Context

The dashboard now needs three cross-cutting capabilities that no longer fit comfortably as isolated local state:

1. chart-driven filters and persisted view preferences on `/panel/events/reported`
2. geohash-aware search state for Nostr `g` tags
3. richer media-player behavior across event search/detail flows

At the same time, the reports charts must reflect global server totals, not only the currently loaded list slice.

### Decision

1. introduce `zustand` + `immer` for global UI/session stores with scoped `localStorage` sync
2. keep TanStack Query for server state and add a dedicated `/admin/events/reported/summary` query for global charts
3. adopt `ngeohash` for geohash normalization and operator-facing geohash search features
4. replace the plain `<video>` path with `@vidstack/react` using `hls.js` and `dashjs` for richer stream support

### Reasons

1. **Consistency:** chart interactions and persisted preferences should survive route churn.
2. **Correctness:** moderation charts must match full-server totals, not viewport or virtualized slice size.
3. **Extensibility:** geohash and richer video playback become reusable capabilities instead of one-off hacks.

### Consequences

- ✅ reports analytics can become globally correct and interactively filterable
- ✅ global UI state becomes explicit and reusable
- ✅ geohash and richer media support align the dashboard better with Nostr protocol domains
- ⚠️ adds new dependencies and requires strict discipline to keep server state out of Zustand stores

---

## ADR-F012: Event Search Analytics Stay in a Modal Instead of Expanding the Main Route Permanently

**Status:** Proposed  
**Date:** 2026-05-08

### Context

`/panel/events/search` already exposes search filters, KPIs, result list, aggregates and timeline tabs. Operators now want a dedicated button that opens a modal containing charts related to the current active search filters.

### Decision

1. keep `/panel/events/search` as the main operational route
2. add a header action button to open an analytics modal
3. reuse existing event-search analytical queries and Recharts components where practical
4. keep the modal as a secondary drill-in workspace instead of permanently expanding the base page layout

### Reasons

1. **Low-risk UX change:** preserves the current route structure and list workflow.
2. **Reusability:** existing aggregates/timeline logic can be reused.
3. **Operator speed:** charts become accessible immediately without navigating away from the current search context.

### Consequences

- ✅ event-search analytics become more discoverable
- ✅ the route remains focused on search + results by default
- ✅ implementation can stay compact and incremental
- ⚠️ the route now has two analytics entry patterns (tabs and modal) that must remain visually coherent

For the next refinement, the modal should gain its own top-line KPI strip, additional charts and tab targeting so it feels like a real analytical workspace rather than a thin wrapper around existing panels.

For the current follow-up, the modal must also treat server-backed event-search aggregates as authoritative totals, while the search list remains only a virtualized drill-down projection.

For the next refinement, `/panel/events/reported` becomes explicitly URL-driven, so chart interactions must synchronize with route search params instead of living only in persisted UI state.

---

## ADR-F013: Blossom Management Uses One Tabbed Route with Drawer-Based Drill-Down

**Status:** Proposed  
**Date:** 2026-05-12

### Context

The requested Blossom management scope is broad: KPI monitoring, file browsing, review queue, user quotas, mirroring, worker health, EXIF/privacy visibility and immutable audit logs. Splitting each concern into isolated routes would fragment operator context and make cross-checking a file, its uploader and its background jobs unnecessarily slow.

### Decision

1. add one `/blossom` route in the dashboard
2. keep route-level navigation inside the page using tabs/sections instead of multiple top-level routes
3. use table/grid toggle for the media browser and a right-side sheet for object/user drill-down
4. keep data fetching in the route/workspace layer and rendering blocks dumb
5. use TanStack Query mutations for bulk review, quotas, mirror and purge actions

### Reasons

1. **Operator speed:** one workspace reduces navigation churn during moderation and storage triage.
2. **Visual coherence:** the current admin shell already supports dense operational pages well.
3. **Architecture fit:** drawers isolate detail complexity without promoting every drill-down to a new route.
4. **Performance:** only the active tab needs heavy server-state polling.

### Consequences

- ✅ the page can show KPIs, list/grid browsing and worker state in one coherent flow
- ✅ the Smart/Dumb split remains explicit and testable
- ✅ object inspection stays contextual instead of bouncing to a dedicated page
- ⚠️ the route becomes data-dense and requires strict hierarchy to avoid operator overload
- ⚠️ polling scope must be limited so inactive tabs do not create unnecessary network load

---

## ADR-F014: Blossom Uses Modal Overlays for Workers and Analytics, While Reports Stay as a Tab

**Status:** Proposed  
**Date:** 2026-05-15

### Context

Operators need two new interaction styles on top of the existing Blossom route:

- a `Workers` shortcut that exposes queue state from anywhere in the workspace
- an analytics surface with charts and operational summaries
- a BUD-09 report workflow that is important enough to deserve persistent filtering and drill-down

If all three concerns become tabs, quick worker inspection becomes slower. If all three become modals, reports lose URL-driven discoverability and long-form moderation ergonomics.

### Decision

1. keep `reports` as a first-class Blossom tab
2. open workers through a modal dialog from the header shortcut
3. open analytics through a separate modal dialog
4. keep both modal overlays smart and query-backed, while list and chart primitives remain dumb

### Reasons

1. **Fast operations:** workers are often checked briefly, not navigated to for long sessions.
2. **Cognitive load:** analytics benefits from focused modal context without permanently occupying tab space.
3. **Moderation depth:** reports need filtering, pagination and drill-down that work better as a stable tab.

### Consequences

- ✅ operators can inspect workers without losing the current Blossom tab context
- ✅ analytics can load lazily only when requested
- ✅ reports remain bookmarkable and URL-driven inside the route
- ⚠️ overlay state must not fight with tab state or sheet state

---

## ADR-F016: Detailed Blossom Quota Plan Editing Lives in a Child Route, Not in the Main Workspace Tabs

**Status:** Proposed  
**Date:** 2026-05-25

### Context

Operators now need richer quota-plan authoring: named plans, default assignments, editable storage/egress limits, explanatory help for byte units and clearer destructive actions. This is deeper than the compact operational controls already living in `/blossom`.

### Decision

1. add `/blossom/plans` as a lower-level configuration route
2. keep `/blossom` as the operational hub and link to plans from there
3. use a two-pane configuration UX: plan list/grid plus focused detail editor
4. use icon-triggered tooltips for storage-unit help near MB/GB inputs

### Reasons

1. **Hierarchy:** advanced configuration should feel one level deeper than daily operations.
2. **Cognitive load:** quota modeling needs more space, more copy and safer affordances.
3. **Buildability:** a dedicated child route keeps the Smart/Dumb split clean and easier to maintain.

### Consequences

- ✅ operators get a clearer plan-management UX with room for explanation and guardrails
- ✅ `/blossom` avoids becoming an overstuffed mega-page
- ⚠️ cross-route invalidation between plans and overview must stay explicit

---

## ADR-F015: Dashboard Persistence Uses Zustand Stores, with localStorage for Small Preferences and IndexedDB for Larger Operator History

**Status:** Proposed  
**Date:** 2026-05-25

### Context

The admin dashboard already depends on `zustand` and uses it correctly in some places, but persistence is inconsistent:

- some preferences already use `zustand` + `localStorage`
- relay presets still use manual `localStorage` helpers
- there is no typed IndexedDB-backed persistence for larger operator-side artifacts

The next refinement needs a more explicit storage policy before adding more persistence around Blossom workflows.

### Decision

1. standardize client persistence behind `zustand` stores whenever the data belongs to app state
2. keep URL-search params as the source of truth for shareable filters
3. keep `localStorage` for compact preferences and recent values
4. introduce IndexedDB only where payload size or append-oriented history makes it materially useful

### Reasons

1. **Type safety:** stores make persisted shapes explicit and reviewable.
2. **Consistency:** UI state should not be split between ad-hoc helpers and store middleware.
3. **Pragmatism:** IndexedDB is more complex and should only be used where `localStorage` is a poor fit.

### Consequences

- ✅ relay preset persistence can move out of raw helpers into one store
- ✅ existing zustand usage remains aligned with current architecture
- ✅ larger operator history such as Blossom mirror submissions can be kept outside `localStorage`
- ⚠️ a small storage adapter layer is needed for IndexedDB-backed stores
