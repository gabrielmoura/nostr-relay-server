# Components Tree

## Hierarchy Overview

```
App (Smart)
├── ErrorBoundary (global fallback)
├── AppShell (layout)
│   ├── Header
│   ├── Sidebar (navigation)
│   └── Outlet
│       └── Route Components (Smart)
└── Routes (file-based via TanStack Router)
```

---

## UI Components (Dumb)

Located in `infra/dash/src/components/ui/`

| Component | Props | Purpose |
|------------|-------|---------|
| `Button` | `variant`, `size`, `disabled`, `onClick`, `children` | Action buttons |
| `Card` | `className`, `children` | Content container |
| `Dialog` | `open`, `onOpenChange`, `children` | Modal overlay |
| `Input` | `type`, `value`, `onChange`, `placeholder` | Text input |
| `Select` | `value`, `onValueChange`, `options` | Dropdown selection |
| `Table` | `columns`, `data` | Data tables |
| `Tabs` | `defaultValue`, `children` | Tabbed content |
| `Badge` | `variant`, `children` | Status labels |
| `Avatar` | `src`, `alt`, `fallback` | User image |
| `Skeleton` | `className` | Loading placeholder |
| `Toast` | `title`, `description`, `variant` | Notifications |

---

## Shared Components (Dumb/Smart mix)

Located in `infra/dash/src/components/shared/`

| Component | Type | Props | Events |
|-----------|------|-------|--------|
| `PageHeader` | Dumb | `title`, `description`, `actions`, `breadcrumbs` | - |
| `MetricCard` | Dumb | `title`, `value`, `change`, `icon` | - |
| `UserAvatarChip` | Dumb | `pubkey`, `name`, `showVerified` | - |
| `StatePanels` | Smart | `data` | - |
| `VirtualizedList` | Smart | `items`, `renderItem`, `height` | `onEndReached` |
| `RelayListModal` | Smart | `open`, `onOpenChange`, `value`, `onChange`, `storageKey` | `onChange`, `onSubmit` |

---

## Feature Components

### Event Detail (`components/features/event-detail/`)

| Component | Type | Props | Events |
|-----------|------|-------|--------|
| `EventMetadata` | Dumb | `event` | - |
| `EventMedia` | Dumb | `event` | - |
| `EventImageGrid` | Dumb | `images` | - |
| `EventVideoPlayer` | Dumb | `src`, `poster` | - |
| `MediaCarousel` | Dumb | `media`, `poster`, `altTexts?`, `lazyVideo` | `onSlideChange?` |
| `EventRepostCard` | Dumb | `repostedEvent` | - |
| `ReactionTargetEvent` | Dumb | `targetEvent` | - |
| `EventListItems` | Dumb | `events`, `onEventClick` | `onEventClick` |
| `CommunityApprovalCard` | Dumb | `communityRef`, `approvedEventId`, `approvedKind`, `postAuthor`, `approvedEvent?` | - |
| `DMRelayListCard` | Dumb | `relays` | - |
| `NostrReferences` | Dumb | `event` | - |
| `ListRefSyncCard` | Smart | `listId`, `onSync`, `onClose` | `onSync`, `onClose`, `onRelaySelect` |
| `RelaySearchModal` | Smart | `open`, `onClose` | `onSelect` |
| `EventDetailErrorState` | Dumb | `error`, `onRetry` | `onRetry` |

Planned additions for rich-event work:

- `EventKindSummaryCard` (Dumb): shared summary surface for empty-text protocol events.
- `EventMediaPreview` (Dumb): chooses single image, single video gate, or carousel.
- `EventMediaStatsCard` (Dumb): counts images, videos, MIME types and alt labels for operators.
- `EventReferencedContext` (Smart or near-smart): fetches minimal referenced event context through `@nostrify/react` when admin payload does not already include it.

Relay selection for event recovery should delegate persistence and editing to the shared `RelayListModal` storage workflow.

---

### Download (`components/features/download/`) - Planned refinement

| Component | Type | Purpose |
|-----------|------|---------|
| `DownloadJobQueue` | Smart | Polls and renders backend-backed download jobs |
| `DownloadJobCard` | Dumb | Visual card for one queued/running/completed/failed download job |
| `DownloadJobDetailsDialog` | Dumb | Shows per-job relay results, summary and raw filters |
| `DownloadFiltersDialog` | Dumb | Compact viewer for the serialized filter used by a job |

### Generic Jobs (`components/features/jobs/`) - Planned

| Component | Type | Props | Events |
|-----------|------|-------|--------|
| `JobsBoard` | Smart | `jobs`, `isLoading`, `error`, `filterPreset`, `onRetry`, `onCancel`, `onRefresh` | `onRetry`, `onCancel`, `onRefresh`, `onSelectJob` |
| `JobQueueFilters` | Dumb | `queueOptions`, `statusOptions`, `value`, `onChange` | `onChange` |
| `JobQueueSummary` | Dumb | `totals`, `activeCount`, `deadCount`, `delayedCount` | - |
| `JobCard` | Dumb | `job`, `onViewDetails`, `onRetry`, `onCancel` | `onViewDetails`, `onRetry`, `onCancel` |
| `JobDetailsDialog` | Dumb | `job`, `open`, `onOpenChange` | `onOpenChange` |
| `SyncJobFilterPanel` | Dumb | `job` | - |
| `SyncJobDiagnosticsPanel` | Dumb | `job` | - |
| `JobResultPanel` | Dumb | `job` | - |
| `JobEmptyState` | Dumb | `title`, `description` | - |

Rules:

- `JobsBoard` is the only smart component in this module.
- all rendering-only blocks stay dumb and reusable across `/download` and `/sync`.
- no component in `components/features/jobs/` talks to `fetch` directly.
- sync-specific detail enrichment is additive: only the modal content branches on `job.job_name === "sync.negentropy"`.

**Parser**: `lib/event-parser.ts` (extracts imeta, media, references from event tags)

---

### Event Search (`components/features/event-search/`)

| Component | Type | Props | Events |
|-----------|------|-------|--------|
| `EventSearchForm` | Smart | `initialValues`, `onSearch` | `onSearch` |
| `EventSearchItem` | Dumb | `event`, `relays` | - |
| `EventSearchAggregates` | Dumb | `counts`, `kinds` | - |
| `EventSearchTimeline` | Smart | `events`, `onLoadMore` | `onLoadMore` |
| `EventImportModal` | Smart | `open`, `onClose` | `onImport` |
| `EventSearchAnalyticsModal` | Dumb | `open`, `onOpenChange`, `initialTab`, `aggregates`, `timeline`, `isLoading`, `isError`, `onRetry` | `onOpenChange`, `onRetry` |
| `EventSearchAnalyticsKpiStrip` | Dumb | `metrics` | - |
| `EventSearchTopAuthorsChart` | Dumb | `items` | `onBarSelect?` |
| `EventSearchTopTagsChart` | Dumb | `items` | `onSliceSelect?` |

For the current refinement, the event-search analytics modal also needs:

- a relay-overview interpretation of KPIs, not only list-local counters
- click-driven kind/tag filtering inside the modal
- author rows/bars that can both filter and navigate to the user detail page
- a trends tab for month/year-oriented tag summaries when backed by aggregates

Planned additions for list rendering:

- `EventSearchCommunityContext` (Dumb): compact context strip with community thumbnail, semantic badge and resolved community label.
- `EventSearchMediaInline` (Dumb): compact single-image, single-video-gate or compact carousel preview.
- `EventSearchKindBadgeRow` (Dumb): semantic badges for `kind:6`, `kind:4550`, `kind:10050`, `kind:20`, `kind:21`, `kind:31234` and list kinds.
- `EventSearchProtocolCard` (Dumb): protocol-specific miniature card used when text content is empty or secondary.

Additional event-search refinements now required:

- `EventSearchCommunityPostPreview` (Dumb): textual preview + associated tags for `kind:1111` community posts.
- `EventKindTooltip` (Dumb): tooltip wrapper for `K:*` badges using NIP-derived descriptions.
- `EventReferenceCopyBadge` (Dumb): click-to-copy badge for event references using NIP-19 when applicable.

For the current refinement, `EventSearchCommunityContext` must include:

- resolved community thumbnail when available
- visible semantic badge such as `Post da comunidade` or `Aprovacao da comunidade`
- community name or identifier fallback

**Parser**: `lib/event-search.ts` (transforms API response to display data)

---

### Reported Events (`components/features/reported-events/`) - Planned refinement

| Component | Type | Props | Events |
|-----------|------|-------|--------|
| `ReportedEventsKpiStrip` | Dumb | `metrics` | - |
| `ReportedEventsTrendChart` | Dumb | `points`, `isEmpty` | `onPointSelect?` |
| `ReportedEventsTypeChart` | Dumb | `items`, `isEmpty` | `onSliceSelect?` |
| `ReportedEventsTopAuthorsChart` | Dumb | `items`, `isEmpty` | `onBarSelect?` |
| `ReportedEventsTopTargetsChart` | Dumb | `items`, `isEmpty` | `onBarSelect?` |
| `ReportedEventsFilters` | Dumb | `query`, `reportType`, `onQueryChange`, `onTypeChange` | `onQueryChange`, `onTypeChange` |
| `ReportedEventsWorkspace` | Smart | `initialQuery?`, `initialType?` | `onSelectEvent`, `onRetry` |

Rules:

- the route or `ReportedEventsWorkspace` is the only smart orchestrator
- all chart blocks remain dumb and receive pre-aggregated props from the server-backed summary query
- `recharts` usage stays isolated to the analytical components
- the event list and the reports modal remain drill-down surfaces under the analytical summary layer

### Global State (`stores/`) - Planned

| Store | Type | Purpose |
|-------|------|---------|
| `reported-events-store` | Smart infra | Global filter state, chart selections and persisted analytics preferences |
| `media-player-store` | Smart infra | Session-level video/player preferences and safe persisted UI flags |
| `geohash-search-store` | Smart infra | Geohash input normalization and optional persisted recent values |
| `relay-presets-store` | Smart infra | Shared relay preset persistence replacing raw `localStorage` helpers |
| `blossom-operator-store` | Smart infra | Persisted Blossom UI preferences and IndexedDB-backed mirror history |

Rules:

- stores hold global UI/session state only
- server datasets remain in TanStack Query
- stores use `zustand` with `immer`
- small stores persist to `localStorage`
- larger append-oriented operator history may persist through an IndexedDB storage adapter

---

### Ban User (`components/features/ban-user-dialog/`)

| Component | Type | Props | Events |
|-----------|------|-------|--------|
| `BanUserDialog` | Smart | `open`, `onClose`, `user` | `onConfirm`, `onCancel` |
| `UnbanUserAlert` | Smart | `user`, `onConfirm` | `onConfirm` |

---

### NIP-05 (`components/features/nip05-*/`)

| Component | Type | Purpose |
|-----------|------|---------|
| `Nip05AssociateDialog` | Smart | Associate NIP-05 to pubkey |
| `Nip05EditDialog` | Smart | Edit existing NIP-05 |
| `Nip05CreateDialog` | Smart | Create new NIP-05 entry |

---

### NIP-86 (`components/features/nip86/`) - Planned

| Component | Type | Purpose |
|-----------|------|---------|
| `AllowedPubkeysPanel` | Smart | Search, list and mutate allowlisted pubkeys |
| `BlockedIPsPanel` | Smart | Search, list, block/unblock IPs and surface disconnect impact |
| `BannedEventsPanel` | Smart | Search, list and unban moderated event ids |
| `RelayMetadataForm` | Smart | Edit runtime relay name/description overrides |
| `Nip86ActionToolbar` | Dumb | Compact filter/action strip reused across NIP-86 lists |

---

### Labels (`components/features/labels/`) - Planned

| Component | Type | Props | Events |
|-----------|------|-------|--------|
| `LabelsWorkspace` | Smart | `initialFilters?` | `onCreateLabel`, `onBanPubkey`, `onFilterChange` |
| `LabelsAnalyticsModal` | Dumb | `open`, `onOpenChange`, `summary` | `onOpenChange` |
| `LabelsHelpDialog` | Dumb | `open`, `onOpenChange` | `onOpenChange` |
| `LabelsStatsStrip` | Dumb | `summary` | - |
| `LabelsFilterBar` | Dumb | `filters`, `namespaces`, `labels`, `onChange` | `onChange`, `onReset` |
| `LabelsTimeline` | Dumb | `items`, `isLoading`, `onBanPubkey` | `onBanPubkey`, `onSelectItem` |
| `LabelsTargetsTable` | Dumb | `items`, `onBanPubkey` | `onBanPubkey`, `onSelectTarget` |
| `CreateLabelDialog` | Smart | `defaultTarget?`, `open`, `onOpenChange`, `onCreated` | `onCreated`, `onOpenChange` |
| `LabelFormFields` | Dumb | `value`, `errors`, `onChange` | `onChange`, `onSubmit` |
| `LabelCategoryPicker` | Dumb | `selected`, `onToggle`, `onAddCustom` | `onToggle`, `onAddCustom` |
| `LabelTargetBadge` | Dumb | `target` | - |
| `LabelEmptyState` | Dumb | `title`, `description` | - |

---

## Route Components (Smart)

Located in `infra/dash/src/routes/`

| Route | Component | Lines | Purpose |
|-------|-----------|-------|---------|
| `/` | `overview-page.tsx` | ~178 | Dashboard metrics |
| `/events/search` | `event-search-page.tsx` | 158 | Event search |
| `/events/:id` | `event-detail-page.tsx` | 190 | Event detail |
| `/users/search` | `user-search-page.tsx` | ~133 | User search |
| `/users/:pubkey` | `user-detail-page.tsx` | ~143 | User profile |
| `/connections` | `active-connections-page.tsx` | - | Active WS connections |
| `/logged-connections` | `logged-connections-page.tsx` | - | Connection history |
| `/logged-users` | `logged-users-page.tsx` | - | Authenticated users |
| `/banned` | `banned-users-page.tsx` | - | Banned user list |
| `/reported` | `reported-events-page.tsx` | ~154 | NIP-56 reports |
| `/stream` | `stream-status-page.tsx` | ~127 | Stream/forward status |
| `/nip05` | `nip05-page.tsx` | - | NIP-05 management |
| `/nip86` | `nip86-page.tsx` | - | NIP-86 command center |
| `/labels` | `labels-page.tsx` | - | NIP-32 labels management |
| `/sync` | `sync-page.tsx` | ~100 | Negentropy synchronization |
| `/download` | `download-page.tsx` | ~110 | Bulk event download |
| `/groups` | `groups-page.tsx` | ~120 | NIP-29 group management |
| `/wot` | `wot-page.tsx` | ~110 | WoT & Trusted Pubkeys |

Planned route refinements:

- `download-page.tsx` becomes a smart orchestrator for `DownloadForm` + generic `JobsBoard`
- `sync-page.tsx` becomes a smart orchestrator for `SyncForm` + generic `JobsBoard`
- `event-detail-page.tsx` should aggregate labels, reports, replies, responder identities and associated events around the primary event

Additional refinements in scope:

- `NostrFilterBuilder` should normalize hex/NIP-19 input before mutating route state
- `JobsBoard` should gain explicit `resume`, real backend `clear history`, filter preview, and terminal `reenqueue` affordances where supported
- `event-detail-page.tsx` should show richer responder cards and moderator identities for community events (`kind:34550`)
- `event-search-page.tsx` and `event-detail-page.tsx` should share one media interpretation model so image/video/alt behavior stays consistent across search and detail views

---

## Layout Components

| Component | Type | Purpose |
|-----------|------|---------|
| `AppShell` | Smart | Main layout wrapper, routes to sidebar |

---

## Component Communication

### Props Flow

```
Route (Smart)
    │
    ├─► Feature Components (Smart)
    │       │
    │       └─► Dumb Components (render)
    │
    └─► Shared/Dumb Components (render)
```

### Event Flow

```
User Action (click/submit)
    │
    ▼
Dumb Component (callback)
    │
    ▼
Feature Component (handler)
    │
    ▼
Route Component (API call / state update)
```

---

## Error Boundary Placement

| Boundary | Location | Catches |
|----------|----------|---------|
| Global | `App.tsx` | Fatal React errors |
| Per-Route | Route components | Feature-level failures |
| Per-Component | Risky components | Isolated render failures |

---

## Unreviewed Large Components (>200 lines)

These components may need future refactoring:

| Component | Location | Lines | Notes |
|-----------|----------|-------|-------|
| `ban-user-dialog.tsx` | `components/features/` | 229 | Consider split |
| `state-panels.tsx` | `components/shared/` | - | Consider split |
| `app-shell.tsx` | `components/layout/` | - | Consider split |

---

## Import Dependencies

```
routes/
  └─► components/features/
        └─► components/shared/
              └─► components/ui/
```

Routes should not import directly from `ui/` unless for very specific use cases. Prefer feature components as the interface.

---

## Planned Blossom Feature Module

| Component | Type | Props | Events |
|-----------|------|-------|--------|
| `BlossomWorkspace` | Smart | `initialTab?`, `initialFilters?` | `onFilterChange`, `onSelectObject`, `onSelectUser` |
| `BlossomPlansPage` | Smart | `initialPlanId?` | `onCreatePlan`, `onEditPlan`, `onDeletePlan`, `onAssignPlan` |
| `BlossomPolicyPage` | Smart | - | `onPolicyModeChange` |
| `BlossomReviewPage` | Smart | `initialFilters?` | `onApprove`, `onDelete`, `onSelectObject` |
| `BlossomReportsPage` | Smart | `initialFilters?` | `onResolve`, `onSelectObject` |
| `BlossomAuditPage` | Smart | `initialFilters?` | `onFilterChange` |
| `BlossomKpiStrip` | Dumb | `summary` | - |
| `BlossomAlertRail` | Dumb | `alerts` | `onSelectAlert?` |
| `BlossomFiltersBar` | Dumb | `filters`, `mimeOptions`, `extensionOptions`, `onChange` | `onChange`, `onReset` |
| `BlossomMimeCombobox` | Dumb | `value`, `options`, `onChange` | `onChange` |
| `BlossomUserFilter` | Dumb | `value`, `onChange` | `onChange` |
| `BlossomViewToggle` | Dumb | `view`, `onChange` | `onChange` |
| `BlossomObjectsTable` | Dumb | `items`, `selectedHashes`, `onToggleSelection`, `onSelectObject` | `onToggleSelection`, `onSelectObject` |
| `BlossomObjectsGrid` | Dumb | `items`, `selectedHashes`, `onToggleSelection`, `onSelectObject` | `onToggleSelection`, `onSelectObject` |
| `BlossomBulkActionsBar` | Smart | `selectedHashes`, `onCompleted` | `onApprove`, `onDelete`, `onRequeue` |
| `BlossomObjectSheet` | Smart | `hash`, `open`, `onOpenChange` | `onDelete`, `onReprocess`, `onCopyUrl`, `onCopyBlossomId` |
| `BlossomAnalyticsDialog` | Smart | `open`, `onOpenChange` | `onRetry`, `onSelectSegment` |
| `BlossomReviewQueue` | Smart | `filters?` | `onApprove`, `onDelete`, `onSelectObject` |
| `PolicySummaryCard` | Dumb | `mode` | `onOpenPolicyRoute` |
| `BlossomPlanModal` | Dumb | `open`, `plan?`, `saving` | `onSave`, `onOpenChange` |
| `BlossomDeletePlanModal` | Dumb | `open`, `planName`, `deleting` | `onConfirm`, `onOpenChange` |
| `BlossomAssignPlanModal` | Smart | `open`, `planId`, `planName` | `onAssign`, `onSearch`, `onOpenChange` |
| `BlossomStorageHelpTooltip` | Dumb | `unit`, `scope` | - |
| `BlossomWhitelistEditor` | Smart | `record?`, `onSaved` | `onSaved` |
| `BlossomUsersTable` | Smart | `items`, `policyMode`, `sortBy`, `sortDir` | `onSortChange`, `onWhitelistToggle`, `onPurge` |
| `BlossomUserSheet` | Smart | `pubkey`, `open`, `onOpenChange` | `onPurge`, `onQuotaSave` |
| `BlossomMirrorPanel` | Smart | `onSubmitted` | `onSubmitted` |
| `BlossomWorkersBoard` | Dumb | `jobs`, `onRefresh` | `onRefresh`, `onSelectJob` |
| `BlossomWorkersDialog` | Smart | `open`, `onOpenChange`, `filters?` | `onRefresh`, `onSelectJob` |
| `BlossomReportsTable` | Dumb | `items`, `onSelectReport`, `onResolve` | `onSelectReport`, `onResolve` |
| `BlossomReportSheet` | Smart | `reportId`, `open`, `onOpenChange` | `onResolve`, `onSelectObject` |
| `BlossomAuditTable` | Dumb | `items`, `filters`, `onChange` | `onChange` |

Rules:

- `BlossomWorkspace` is the main smart orchestrator for route-level queries and tab state.
- `BlossomPlansPage` is a focused child route dedicated to named plans, modal CRUD and user-plan association.
- `BlossomPolicyPage` owns the exclusive upload-mode selector and removes policy editing from the plans screen.
- `BlossomReviewPage`, `BlossomReportsPage` and `BlossomAuditPage` are child routes used to reduce density on the main Blossom hub.
- drawers/sheets own their drill-down query or mutation orchestration when it meaningfully isolates complexity.
- thumbnails, badges, metadata rows, KPI cards and audit rows remain dumb.
- modal overlays (`BlossomWorkersDialog`, `BlossomAnalyticsDialog`) are smart because they own short-lived query orchestration.
- the users table is smart because sorting, infinite pagination and conditional columns depend on server state.
- destructive controls (`hard-delete`, `purge`) must always flow through confirmation UI.
