# Frontend TODO

## Completed ✓

### 1. i18n Coverage
- [x] Add `eventDetail.*` translation keys to `en.json`
- [x] Add `eventDetail.*` translation keys to `pt-BR.json`

### 2. Event Detail Page Refactoring (944 → 190 lines)
- [x] Extract parsing logic to `lib/event-parser.ts`
- [x] Create `EventMetadata` component
- [x] Create `EventMedia` component
- [x] Create `EventImageGrid` component
- [x] Create `EventVideoPlayer` component
- [x] Create `EventRepostCard` component
- [x] Create `ReactionTargetEvent` component
- [x] Create `EventListItems` component
- [x] Create `NostrReferences` component
- [x] Create `ListRefSyncCard` component
- [x] Create `RelaySearchModal` component
- [x] Create `EventDetailErrorState` component
- [x] Fix TypeScript build errors
- [x] Verify build passes

### 3. Event Search Page Refactoring (474 → 158 lines)
- [x] Extract parsing logic to `lib/event-search.ts`
- [x] Create navigation hook in `lib/router.ts`
- [x] Create `EventSearchForm` component
- [x] Create `EventSearchItem` component
- [x] Create `EventSearchAggregates` component
- [x] Create `EventSearchTimeline` component
- [x] Create `EventImportModal` component
- [x] Fix TypeScript build errors
- [x] Verify build passes

### 4. Admin Dashboard Enhancements
- [x] Adopt TanStack Query for server state management
- [x] Atualizar Vite para v8 e Rolldown
- [x] Implementar `NostrFilterBuilder` avançado (NIP-01, 24, 29, 34, 35, 39, 50, 52, 73)
- [x] Substituir formulário de busca legado em `/panel/events/search`
- [x] Integrar construtor de filtros em `SyncPage` e `DownloadPage`
- [x] Correção de erros críticos de nulos no WoT
- [x] Implementar Error Boundaries e Suspense em todas as páginas administrativas
- [x] Criar tela de "Funcionalidade Desabilitada" (NIP-86, WoT, NIP-29)
- [x] Create `sync-page.tsx` for Negentropy synchronization
- [x] Create `download-page.tsx` for bulk event download
- [x] Create `groups-page.tsx` for NIP-29 group management
- [x] Create `wot-page.tsx` for WoT & Trusted Pubkeys management
- [x] Add i18n coverage for all new features
- [x] Register new routes in `router.tsx`

### 5. Documentation
- [x] Create/Update `docs/frontend-architecture.md`
- [x] Create/Update `docs/components-tree.md`
- [x] Create/Update `docs/state-management.md`

---

## In Progress

### Documentation Update
- [ ] Review and finalize `docs/frontend-architecture.md`
- [ ] Review and finalize `docs/components-tree.md`
- [ ] Review and finalize `docs/state-management.md`

### Rich Event Visualization
- [ ] Extend `lib/event-parser.ts` with one shared media interpretation model for search and detail
- [ ] Add alt-aware headline fallback so `(sem conteudo textual)` always appends the `alt` tag when present in list rendering
- [ ] Interpret `kind:4550` visually in `/panel/events/search` and `/panel/events/$eventId` using NIP-72 semantics
- [ ] Interpret `kind:6` visually as repost context even when embedded content is absent
- [ ] Interpret `kind:10050` visually as DM relay list with badges and counts instead of empty content emphasis
- [ ] Render one image directly and multiple images as carousel in event detail
- [ ] Render one video as click-to-load preview in search results when inferred from `imeta` MIME or URL
- [ ] Add carousel support for multi-video or mixed-media posts in kinds `1`, `20`, NIP-68 picture-first flows and NIP-51 list-related media previews when multiple assets are present
- [ ] Prevent `/panel/events/$eventId` content from overflowing horizontally by enforcing `min-w-0`, capped media sizes and wrapped identifier blocks
- [ ] Introduce `@nostrify/react` enrichment path for referenced event context without replacing admin REST as the main source
- [ ] Verify `pnpm build` for `infra/dash`

### NIP-86 Dashboard Extension
- [ ] Add NIP-86 dashboard information architecture to docs
- [ ] Define new internal admin service endpoints consumed by the SPA
- [ ] Add visual system updates for compact moderation workflows

### Relay Workflow UX
- [ ] Add reusable relay selection modal with localStorage persistence
- [ ] Replace raw comma-separated relay editing on `/download` with modal-assisted workflow
- [ ] Reuse persisted relay selections in event-detail relay recovery

### Download Queue UX
- [ ] Add backend-backed download job queue cards
- [ ] Add “Ver filtros” and “Ver detalhes” dialogs for each download job
- [ ] Surface running/completed/failed visual states without relying only on toast messages

### Generic Operational Jobs UX
- [ ] Extend `types/admin.ts` with generic job contracts
- [ ] Extend `services/admin.ts` with generic jobs endpoints and error/request-id propagation
- [ ] Add TanStack Query hooks for generic jobs polling, retry and cancel
- [ ] Create `components/features/jobs/` smart/dumb split
- [ ] Refactor `/download` to consume generic jobs board filtered by `download.events`
- [ ] Refactor `/sync` to consume generic jobs board filtered by `sync.negentropy`
- [ ] Add i18n copy for generic queue states, actions and empty/error panels
- [ ] Verify `pnpm build` for `infra/dash`

### NIP-32 Labels Dashboard
- [ ] Add `/labels` route in `router.tsx`
- [ ] Add nav entry in `components/layout/app-shell.tsx`
- [ ] Extend `types/admin.ts` with `AdminLabelEvent`, `AdminLabelTarget`, `AdminLabelsSummary`, and create payload types
- [ ] Extend `services/admin.ts` with `getLabels`, `getLabelsSummary`, and `createLabel`
- [ ] Add TanStack Query hooks for labels list, summary and create mutation
- [ ] Create `components/features/labels/` smart/dumb split
- [ ] Implement timeline view for `kind:1985`
- [ ] Implement grouped `By Target` view
- [ ] Implement label creation dialog with target types `event`, `pubkey`, `address`, `reference`, `topic`
- [ ] Accept NIP-19 values in label target input when applicable
- [ ] Add help button + field glossary modal to `/labels`
- [ ] Chain optional pubkey ban after successful label creation
- [ ] Add i18n copy for labels filters, states and actions
- [ ] Verify `pnpm build` for `infra/dash`

### Operational UX Follow-ups
- [ ] Add clear-history interaction to `/download`
- [ ] Add clear-history interaction to `/sync`
- [ ] Add KPI cards to `/events/reported`
- [ ] Add KPI cards to `/users/search`
- [ ] Expand `/events/$eventId` with labels, reports, replies and related actors
- [ ] Extend `NostrFilterBuilder` to accept NIP-19 or hex where applicable
- [ ] Fix `/sync` cancel flow so canceled jobs do not auto-resume
- [ ] Add explicit resume action for canceled sync jobs

### New operator refinements
- [ ] Make labels filter support multiple simultaneous labels
- [ ] Replace UI-only job history clearing with backend deletion flow
- [ ] Show filters used in each sync job card/detail
- [ ] Show the full normalized filter inside the sync job details modal
- [ ] Render sync rejection details from `job.result` before falling back to `last_error`
- [ ] Expose `Reenfileirar` on completed jobs when retry semantics are safe
- [ ] Render kind `34550` metadata inline in `/events/search`
- [ ] Reorder `/events/search` as KPIs -> filters -> results
- [ ] Render moderators and richer reply/responder cards in `/events/$eventId` for community events

### Event Search Analytics Modal
- [ ] Add a button in `/events/search` to open an analytics modal
- [ ] Add an analytics modal that renders charts related to the current event-search filters
- [ ] Reuse or adapt existing Recharts-based event aggregates and timeline components inside the modal
- [ ] Keep the list route primary and the modal secondary as a drill-in analytics surface
- [ ] Add KPI strip at the top of the event-search analytics modal
- [ ] Add dedicated top-authors and top-tags charts inside the modal
- [ ] Allow opening the modal with a specific initial analytical tab
- [ ] Ensure modal KPIs/charts reflect full filtered relay totals, not only the virtualized list items
- [ ] Add click-driven kind/tag filtering inside the event-search analytics modal
- [ ] Show month/year in timeline labels
- [ ] Make active authors in the modal filterable and navigable to the user detail route
- [ ] Add a trends tab with month/year tag highlights when supported by current aggregates

### Event Search Community and Kind UX
- [ ] Render community post preview + associated tags for `kind:1111` rows in `/events/search`
- [ ] Add NIP-based tooltip to `K:1111` badge in `/events/search`
- [ ] Make event-reference copy badges use NIP-19 when copying applicable event identifiers
- [ ] Show resolved user display name in search rows when available for linked authors
- [ ] Enable Vidstack logs during dashboard verification
- [ ] Render the approved event below the `CommunityApprovalCard` on `kind:4550` detail pages

### Reported Events Analytics
- [ ] Refactor `/events/reported` into analytics-first layout
- [ ] Replace static KPI cards with a Recharts-based KPI strip
- [ ] Add report-volume trend chart using the fetched report slice
- [ ] Add report-type distribution chart using NIP-56 types
- [ ] Add top authors or top targets chart for moderation concentration
- [ ] Keep reported-event list and drill-down modal below the analytics summary
- [ ] Verify `pnpm build` for `infra/dash`

### Reported Events Global Analytics
- [ ] Add server-backed `/admin/events/reported/summary` contract for totals independent of virtualized list length
- [ ] Add typed frontend service + hook for reports summary query
- [ ] Refactor reports analytics to consume summary data instead of the loaded list slice
- [ ] Add chart-click filtering interactions for type, timeline and concentration charts
- [ ] Add top-targets chart alongside top-authors chart
- [ ] Add semantic chart colors for NIP-56 report types
- [ ] Make `/events/reported` filters URL-driven with TanStack Router search params
- [ ] Ensure `type=nudity` and other moderation filters work end-to-end without backend SQL errors

### Labels Analytics Modal
- [ ] Add analytics modal entry action in `/labels`
- [ ] Render KPIs and Recharts-based charts inside the labels modal using labels summary data

### Global State and Media Platform
- [ ] Add `zustand` + `immer` for global UI stores with scoped localStorage persistence
- [ ] Add persisted reports analytics store for chart selections and view preferences
- [ ] Add `ngeohash` support for geohash-aware searches using Nostr `g` tags
- [ ] Replace native video rendering with `@vidstack/react` + `hls.js` + `dashjs`
- [ ] Preserve lazy-load behavior for video previews in virtualized/search contexts


---

## Backlog (Future Refactoring)

### Large Route Components (>150 lines)

| Route | Current Lines | Priority | Notes |
|-------|---------------|----------|-------|
| `overview-page.tsx` | ~178 | Medium | Dashboard, may split |
| `reported-events-page.tsx` | ~154 | Medium | NIP-56 reports |
| `ban-user-dialog.tsx` | 229 | High | Consider split |
| `stream-status-page.tsx` | ~127 | Low | Status page |
| `user-search-page.tsx` | ~133 | Low | User search |
| `user-detail-page.tsx` | ~143 | Low | User profile |
| `logged-connections-page.tsx` | - | Low | Connection logs |
| `active-connections-page.tsx` | - | Low | Active connections |
| `logged-users-page.tsx` | - | Low | Auth users |
| `banned-users-page.tsx` | - | Low | Ban list |

### Refactoring Pattern

For each large file, follow the same pattern:

1. **Extract parsing**: Move utility functions to `lib/`
2. **Identify dumb components**: Create in `components/features/[feature]/`
3. **Identify smart components**: Keep in route or create container
4. **Fix imports**: Ensure no circular dependencies
5. **Run build**: Verify TypeScript compiles

### Priority Guidelines

**High**: Files with >200 lines, complex logic, or frequent changes
**Medium**: Files with 150-200 lines, moderate complexity
**Low**: Files with <150 lines, stable functionality

---

## Architecture Improvements (Future)

### Service Layer
- [ ] Create `services/api.ts` - centralized API client
- [ ] Create `services/events.ts` - event-specific API calls
- [ ] Create `services/users.ts` - user-specific API calls

### NIP-86 Admin UX (Phase 1)
- [x] Add `sync-page.tsx` and `download-page.tsx`
- [x] Add NIP-29 and WoT management routes
- [x] Extend `services/admin.ts` and `types/admin.ts` for new responses and mutations
- [ ] Add relay metadata override form to dashboard

### State Management
- [x] Evaluate and Adopt TanStack Query for server state
- [ ] Add React 19 Actions for form submissions where applicable
- [ ] Add useOptimistic for mutations

### Error Handling
- [ ] Add global error tracking (Sentry)
- [ ] Propagate `x-request-id` in error messages
- [ ] Create shared error fallback components

### Testing
- [ ] Add unit tests for parsers
- [ ] Add component tests for dumb components
- [ ] Add integration tests for routes

---

## Build Verification

Always verify build after each refactoring step:

```bash
cd infra/dash
pnpm build
```

Expected: No TypeScript errors, no ESLint errors.

---

## Notes

- Refactoring should maintain functionality - don't change behavior
- Keep dumb components truly dumb - no API calls
- Use composition over configuration
- Avoid over-engineering: create abstractions only when they reduce complexity
- Document any new patterns in `docs/`

## Blossom Admin Workspace

- [ ] Add Blossom information architecture to docs and confirm route/endpoint naming
- [ ] Add `/blossom` route in `router.tsx`
- [ ] Add navigation entry in `components/layout/app-shell.tsx`
- [ ] Extend `types/admin.ts` with Blossom overview, object, review, user, quota, worker and audit contracts
- [ ] Extend types/contracts with Blossom policy, analytics, reports and BUD-10 identifiers
- [ ] Extend `services/admin.ts` with typed Blossom queries and mutations, preserving `x-request-id`
- [ ] Add TanStack Query hooks for Blossom overview, objects, review queue, users, workers and audit
- [ ] Add hooks for Blossom policy, reports and analytics
- [ ] Create `components/features/blossom/` smart/dumb split
- [ ] Implement KPI strip and alert rail
- [ ] Implement object browser with table/grid toggle, exact SHA-256 search and uploader identity filter
- [ ] Replace MIME select with editable combobox behavior
- [ ] Implement right-side inspection sheet with NIP-94 metadata and quick actions
- [ ] Add `Copiar Blossom ID` action using BUD-10 format
- [ ] Implement review queue with bulk approve/hard-delete/reprocess actions
- [ ] Implement policy/settings card for upload mode and default plans
- [ ] Add drill-down link from `/blossom` to `/blossom/plans`
- [ ] Implement `/blossom/plans` child route with stronger UI/UX for named plans and quotas
- [ ] Add plan summary strip, plan grid and detailed editor pane
- [ ] Add storage help tooltip beside MB/GB fields with explicit explanatory copy
- [ ] Implement whitelist/quota editor and uploader table
- [ ] Implement uploader detail sheet with destructive purge action
- [ ] Implement mirror submission panel and live workers board
- [ ] Make the header `Workers` button open a workers modal
- [ ] Implement Blossom analytics modal with charts and KPI summaries
- [ ] Implement BUD-09 reports tab and drill-down
- [ ] Implement immutable audit table
- [ ] Add i18n copy for all Blossom states, tabs, errors and destructive confirmations
- [ ] Verify `pnpm build` for `infra/dash`

## Persistence Normalization

- [ ] Document current `localStorage` usage and classify by relevance
- [ ] Keep existing `zustand` persisted stores where they already fit compact UI state
- [ ] Replace manual relay preset `localStorage` helpers with a dedicated persisted `zustand` store
- [ ] Add a small IndexedDB storage adapter for larger client-side operator history when relevant
- [ ] Persist Blossom mirror submission history in IndexedDB-backed client state if the UI exposes history/retry value
