# State Management

## Overview

The frontend uses React's built-in state mechanisms plus TanStack Router for URL-driven state. No external state management library (Redux, Zustand, etc.) is currently in use.

---

## State Categories

### 1. Local UI State

**Location**: Inside components via `useState`

**Examples**:
- Form input values
- Modal open/close
- Loading spinners
- Tab selection

```tsx
const [isOpen, setIsOpen] = useState(false);
const [tab, setTab] = useState('overview');
```

### 2. URL State (TanStack Router)

**Location**: URL path and query parameters

**Examples**:
- `/events/:id` - event ID from path
- `/events/search?kind=1&limit=50` - search filters

```tsx
const { id } = useRoute('/events/:id');
const searchParams = useSearch({ from: '/events/search' });
```

### 3. Server State

**Location**: Route components, fetched on mount/effect

**Examples**:
- API response data
- Relay status
- user lists

```tsx
const { data, isLoading, error } = useEvent(id);
```

### 4. Persisted Client State

The dashboard already mixes URL state, local component state and persisted browser state.

Survey of current persisted state in `infra/dash/src`:

- `useMediaPlayerStore` -> `zustand` + `localStorage`
- `useGeohashSearchStore` -> `zustand` + `localStorage`
- `useReportedEventsStore` -> `zustand` + `localStorage`
- `relay-presets.ts` -> manual `localStorage`

Target normalization:

- **Zustand + localStorage** for compact UI preferences and small recent-value lists
- **Zustand + IndexedDB-backed storage adapter** for larger operator artifacts where write volume or payload size justifies it
- **Raw localStorage helpers should be phased out** in favor of explicit stores so persistence shape is typed and centralized

Planned relevant uses:

- relay preset lists -> dedicated persisted store instead of manual helper
- Blossom workspace preferences -> persisted store when they are not already URL-driven
- Blossom mirror submission history -> IndexedDB-backed store because it is append-oriented and may grow over time

---

## Feature State Flows

### Event Search (`event-search-page.tsx`)

```
┌──────────────────────────────────────────────────────────────┐
│                    Event Search Flow                         │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  URL Query Params                                            │
│       │                                                      │
│       ▼                                                      │
│  EventSearchForm (Smart)                                     │
│       │ onSearch: (filters) => void                         │
│       │                                                      │
│       ▼                                                      │
│  fetch /api/admin/events (GET)                              │
│       │                                                      │
│       ▼                                                      │
│  Parse via event-search.ts                                   │
│       │                                                      │
│       ▼                                                      │
│  EventSearchTimeline (Smart)                                │
│       │ events: Event[]                                       │
│       │                                                      │
│       ├──► EventSearchItem (Dumb)                            │
│       ├──► EventSearchAggregates (Dumb)                      │
│       └──► EventImportModal (Smart)                          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**States**:
| State | Trigger |
|-------|---------|
| `initial` | No search performed yet |
| `loading` | API call in flight |
| `success` | Data received, render list |
| `empty` | No results for query |
| `error` | API failure |

### Event Detail (`event-detail-page.tsx`)

```
┌──────────────────────────────────────────────────────────────┐
│                    Event Detail Flow                          │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  URL Path :id                                                │
│       │                                                      │
│       ▼                                                      │
│  fetch /api/admin/event/:id (GET)                            │
│       │                                                      │
│       ▼                                                      │
│  Parse via event-parser.ts                                   │
│       │                                                      │
│       ▼                                                      │
│  EventDetailPageContent (Smart)                              │
│       │                                                      │
│       ├──► EventMetadata (Dumb)                              │
│       ├──► EventMedia (Dumb)                                 │
│       │       ├──► EventImageGrid                            │
│       │       └──► EventVideoPlayer                         │
│       ├──► NostrReferences (Dumb)                           │
│       │       ├──► EventRepostCard                           │
│       │       └──► ReactionTargetEvent                      │
│       └──► ListRefSyncCard (Smart)                          │
│               └──► RelaySearchModal                          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**States**:
| State | Trigger |
|-------|---------|
| `loading` | Fetching event |
| `error` | Event not found / API error |
| `success` | Event loaded |
| `repost` | Displaying reposted event |

### Rich Event Visualization State Flow

```
┌──────────────────────────────────────────────────────────────┐
│              Search + Detail Media Interpretation            │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Admin event payload                                          │
│       │                                                      │
│       ├── tags / content / image_urls                        │
│       ├── alt tag                                            │
│       └── kind                                               │
│              │                                               │
│              ▼                                               │
│  event-parser.ts                                             │
│       │                                                      │
│       ├── parseImetaResources()                              │
│       ├── collectMediaForEvent()                             │
│       ├── parseCommunityApproval()                           │
│       ├── parseDMRelays()                                    │
│       └── parseEmbeddedRepost()                              │
│              │                                               │
│              ▼                                               │
│  UI decision layer                                           │
│       │                                                      │
│       ├── no textual content -> append alt                   │
│       ├── one image -> render image                          │
│       ├── many images -> carousel                            │
│       ├── one video -> click-to-load preview                 │
│       ├── mixed media -> carousel                            │
│       ├── kind 4550 -> community approval card               │
│       ├── kind 6 -> repost card                              │
│       ├── kind 10050 -> DM relay list card                   │
│       └── kind 1111 + community a-tag -> community context + preview │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

Additional local UI state:

| State | Location | Purpose |
|-------|----------|---------|
| `videoLoaded` | `EventSearchItem`, `EventVideoPlayer`, carousel item | Defers expensive video rendering until operator intent |
| `selectedSlide` | `MediaCarousel` | Keeps carousel indicators and panel context aligned |
| `detailPanelSizes` | resizable panel state if persisted later | Optional operator preference for left/right split |

Server-state enrichment plan:

- primary event data remains sourced from the internal admin API
- optional referenced-event enrichment may be fetched through `@nostrify/react` queries when protocol cards need extra context not indexed by the admin payload
- optional community-address enrichment may be fetched through `@nostrify/react` queries when search rows need community name/image from `kind:34550`
- failures in this enrichment path must degrade to existing id-based cards, never block the main detail page

Error recovery rules:

- media parse failure falls back to link list, not blank UI
- referenced-event fetch failure falls back to event id and tag metadata
- invalid embedded repost JSON falls back to plain content rendering
- community-name resolution failure for `kind:1111` falls back to the raw `a` tag identifier, not an empty community field
- community-image resolution failure still preserves the community badge and textual label
- user-name resolution failure in search rows falls back to pubkey shortening, not blank author labels

### Event Search Analytics Modal Flow

### Event Search Analytics Modal Flow

```
┌──────────────────────────────────────────────────────────────┐
│              Event Search Analytics Modal Flow               │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Active route search filters                                 │
│       │                                                      │
│       ├── useEventSearchAggregates(filters)                  │
│       └── useEventSearchTimeline(filters, bucket)            │
│              │                                               │
│              ▼                                               │
│  header button toggles analytics modal                       │
│       │ optionally passes initial analytical tab             │
│              │                                               │
│              ▼                                               │
│  modal renders KPI strip + analytical charts for full filtered dataset │
│       │                                                      │
│       ├── click kind/tag refines modal dataset               │
│       ├── click author may filter and/or navigate            │
│       └── trends tab summarizes month/year tag leaders       │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

States:

| State | Meaning |
|-------|---------|
| `closed` | modal hidden |
| `open-loading` | modal open while charts are loading |
| `open-success` | modal open with server-backed aggregates/timeline data |
| `open-error` | modal open with retry surface |
| `open-tabbed` | modal open on a specific requested analytical tab |

### Ban User Flow

```
┌──────────────────────────────────────────────────────────────┐
│                      Ban User Flow                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  User clicks "Ban"                                           │
│       │                                                      │
│       ▼                                                      │
│  BanUserDialog opens                                        │
│       │                                                      │
│       ▼                                                      │
│  User selects reason, clicks confirm                         │
│       │                                                      │
│       ▼                                                      │
│  POST /api/admin/ban                                         │
│       │                                                      │
│       ▼                                                      │
│  On success: close dialog, refresh list                     │
│  On error: show error message                                │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**States**:
| State | Trigger |
|-------|---------|
| `idle` | Dialog open, no action |
| `submitting` | POST in flight |
| `success` | Ban confirmed |
| `error` | API error |

### Sync/Download Mutations

Mutations that trigger background jobs on the server.

- **Sync**: Reconciles events with a remote relay.
- **Download**: Fetches events based on filters from multiple relays.

**Flow**:
1. Component calls `mutateAsync`.
2. UI shows loading overlay/spinner.
3. API returns 200 (job started).
4. UI shows success toast.
5. `onSuccess` callback invalidates relevant lists (e.g., event search).

### Download Queue Flow (Refinement)

```
┌──────────────────────────────────────────────────────────────┐
│                  Download Queue Flow                         │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  RelayListModal                                              │
│       │ persists selected relays in localStorage             │
│       ▼                                                      │
│  DownloadPage (Smart)                                        │
│       │ start download mutation                              │
│       ▼                                                      │
│  POST /admin/events/download                                 │
│       │ returns job_id                                       │
│       ▼                                                      │
│  useDownloadJobsQuery (polling)                              │
│       │                                                      │
│       ├──► DownloadJobQueue                                  │
│       │       └──► DownloadJobCard                           │
│       │               ├──► Ver filtros                       │
│       │               └──► Ver detalhes                      │
│       │                                                      │
│       └──► page-level activity indicator                     │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

States:

| State | Meaning |
|-------|---------|
| `queued` | Request accepted, waiting for execution or first poll |
| `running` | Backend job still processing relay pages |
| `completed` | Backend finished with summary counters |
| `failed` | Backend returned terminal error message |

### Reported Events Analytics Flow

```
┌──────────────────────────────────────────────────────────────┐
│               Reported Events Analytics Flow                 │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Filter state: query + reportType                            │
│       │                                                      │
│       ▼                                                      │
│  useInfiniteReportedEvents(query, type)                      │
│       │                                                      │
│       ├── paginated list rows                                 │
│       │                                                      │
│       └── useReportedEventsSummary(query, type)               │
│               │                                               │
│               ▼                                               │
│          full filtered server aggregates                      │
│       │                                                      │
│       ▼                                                      │
│  dumb Recharts components                                    │
│       │                                                      │
│       ├── KPI strip                                          │
│       ├── trend chart                                        │
│       ├── type chart                                         │
│       ├── top-authors chart                                  │
│       └── top-targets chart                                  │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

States:

| State | Meaning |
|-------|---------|
| `loading` | first moderation slice still loading |
| `success` | list + analytics derived from server-backed summary |
| `empty` | no reported events for the current filter |
| `error` | `/admin/events/reported` failed |

Recovery rules:

- if chart aggregation fails locally, the route should still show the reported-event list
- if a single chart has no usable data, render an empty analytical card instead of failing the page

### Global UI State Flow

```
┌──────────────────────────────────────────────────────────────┐
│                 Global UI State with Zustand                 │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Route / feature action                                      │
│       │                                                      │
│       ▼                                                      │
│  zustand store action                                         │
│       │ uses immer for immutable updates                     │
│       ▼                                                      │
│  persisted slice sync to localStorage                        │
│       │                                                      │
│       └── restored on app boot / feature mount              │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

Rules:

- store only global UI/session intent
- never persist raw reported-event pages
- chart selections may update both store state and route/query state when appropriate
- when a route is declared URL-driven, route search params are the canonical source for restorable filter state

Persistence rules:

- relay presets live in `localStorage`
- backend download job state lives in memory and is fetched by polling
- the frontend queue must preserve local cards until refresh or explicit dismissal, even after completion

### Generic Operational Jobs Flow (Planned)

```
┌──────────────────────────────────────────────────────────────┐
│                Generic Operational Jobs Flow                 │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Feature Route (download / sync)                             │
│       │                                                      │
│       ├── mutation starts backend job                        │
│       │      └── returns job_id                              │
│       │                                                      │
│       └── useJobsQuery({ job_name })                         │
│               │ polling / invalidation                       │
│               ▼                                              │
│           JobsBoard (Smart)                                  │
│               │                                              │
│               ├── JobQueueSummary (Dumb)                     │
│               ├── JobQueueFilters (Dumb)                     │
│               ├── JobCard[] (Dumb)                           │
│               └── JobDetailsDialog (Dumb)                    │
│                                                              │
│  Retry / Cancel actions                                      │
│       │                                                      │
│       └── useRetryJobMutation / useCancelJobMutation         │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

States:

| State | Meaning |
|-------|---------|
| `queued` | accepted and waiting for worker |
| `delayed` | delayed or retry-scheduled |
| `running` | actively executing in a worker |
| `succeeded` | terminal success |
| `failed` | terminal failure without dead-letter promotion yet |
| `dead` | dead-letter state |
| `canceled` | canceled before completion |

Mutation rules:

- start-sync and start-download invalidate the filtered jobs query immediately
- retry invalidates the relevant jobs list and selected job detail
- cancel invalidates the relevant jobs list and selected job detail
- resume invalidates the relevant jobs list and selected job detail
- the board should poll only while there are active states (`queued`, `delayed`, `running`)

Recovery rules:

- query failure -> inline panel with retry action
- mutation failure -> toast + persistent inline error when action is job-specific
- selected job dialog must remain open if retry/cancel fails, preserving operator context
- cancel success for sync must stop future execution until a separate resume mutation is invoked
- sync detail rendering must prefer `job.result.error` and `job.result.rejections` over `job.last_error`
- sync filter rendering must prefer `job.result.filter`, then fallback payload sources without mutating stored job data

### Labels Management Flow (Planned)

```
┌──────────────────────────────────────────────────────────────┐
│                   Labels Management Flow                     │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  URL Search Params                                           │
│       │ namespace / label / targetType / q                   │
│       ▼                                                      │
│  LabelsPage (Smart Route)                                    │
│       │                                                      │
│       ├── useLabelsSummaryQuery(filters)                     │
│       ├── useLabelsQuery(filters)                            │
│       │                                                      │
│       ▼                                                      │
│  LabelsWorkspace (Smart)                                     │
│       │                                                      │
│       ├──► LabelsStatsStrip (Dumb)                           │
│       ├──► LabelsFilterBar (Dumb)                            │
│       ├──► LabelsTimeline (Dumb)                             │
│       └──► LabelsTargetsTable (Dumb)                         │
│                                                              │
│  CreateLabelDialog (Smart)                                   │
│       │                                                      │
│       ├── createLabel mutation                               │
│       └── optional banUser mutation                          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

States:

| State | Meaning |
|-------|---------|
| `loading` | summary/list queries are in flight |
| `success` | list and/or grouped targets available |
| `empty` | no labels at all or no labels for the current filter |
| `error` | list/summary fetch failed |
| `submitting` | create-label mutation in progress |
| `ban-chaining` | label created, optional ban mutation still running |
| `mutation-error` | create or ban action failed; form state preserved |

Mutation rules:

- `createLabel` invalidates `labels`, `labels-summary`, and related target-specific pages when applicable
- optional `banUser` invalidates `banned-users`, `ban-status`, `user`, `users-search`, and `relay-overview`
- if ban chaining fails after label creation succeeds, the UI must report partial success instead of rolling back the created label visually
- NIP-19 typed input is normalized in the client before `createLabel` submission so the API still receives canonical values

Recovery rules:

- form input stays intact on mutation failure
- route-level fetch failure shows inline retry panel
- mutation failures surface `ApiError.requestId` when available

Related state notes:

- jobs history clearing on `/download` and `/sync` is view-local unless a future backend deletion endpoint is introduced
- jobs history clearing should migrate to backend deletion once `DELETE /admin/jobs` is implemented
- event detail associations may partially load independently (`labels`, `reports`, `reply events`, `reply authors`)
- sync canceled jobs should transition to a true terminal state; any future return to execution must only happen through an explicit resume mutation
- `NostrFilterBuilder` should normalize NIP-19 and hex representations into canonical query values before dispatching search state
- the sync details modal is a derived read-only projection of persisted queue payload/result; it must never invent missing diagnostics client-side

Event-search specific note:

- KPI cards should be rendered before filters so the screen opens with immediate situational context

### WoT Management Flow

```
┌──────────────────────────────────────────────────────────────┐
│                      WoT State Flow                          │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  useWoTSummary (Query)                                       │
│       │                                                      │
│       ▼                                                      │
│  Display nodes/edges counts                                  │
│                                                              │
│  useAddTrustedPubkey (Mutation)                              │
│       │                                                      │
│       ▼                                                      │
│  onSuccess: invalidate ["wot-summary"]                       │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

---

## Mutations and Actions

### Pattern (TanStack Query)

Adopted as the primary mechanism for server state.

```tsx
// use-admin-data.ts
export const useSyncMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: startSync,
    onSuccess: () => {
      // Invalidate relevant queries if needed
      queryClient.invalidateQueries({ queryKey: ["events-search"] });
    },
  });
};
```

### When to Use What

| Pattern | Use Case |
|---------|----------|
| TanStack Query | **Primary**. All server data fetching and mutations. |
| React 19 Action | Simple form-driven mutations without cross-component cache needs. |
| Direct fetch | Legacy only. Avoid in new features. |

---

## Optimistic Updates

Currently not implemented. Future consideration for:

- Ban/unban actions (immediate UI update)
- Event deletion
- User profile updates

```tsx
// Future example with useOptimistic
const [optimisticStatus, setOptimistic] = useOptimistic(
  banStatus,
  (_, newStatus) => newStatus
);
```

---

## Error Handling Strategy

### Error States

```tsx
const [error, setError] = useState<Error | null>(null);

// In render
{error ? (
  <ErrorFallback error={error} onRetry={refetch} />
) : (
  <Content />
)}
```

### Error Boundary Usage

| Level | Catches | Shows |
|-------|---------|-------|
| App | Uncaught React errors | "Something went wrong" + retry |
| Route | Feature fetch failures | Feature-specific error UI |
| Component | Isolated render failures | Component fallback |

### Recovery

1. **Retry**: Re-fetch data, re-submit form
2. **Reset**: Clear form state, return to initial
3. **Navigate**: Redirect to list, dashboard

---

## Loading States

### Skeleton Loading

For content-heavy pages:

```tsx
{isLoading ? (
  <div className="space-y-4">
    <Skeleton className="h-4 w-full" />
    <Skeleton className="h-4 w-3/4" />
  </div>
) : (
  <Content />
)}
```

### Inline Loading

For actions:

```tsx
<Button disabled={isLoading}>
  {isLoading ? <Spinner /> : 'Submit'}
</Button>
```

---

## Persistence

### URL Persistence

Search filters, pagination, selected tabs should be in URL:

```tsx
// Good: persists on refresh
<Link to="/events/search" search={{ kind: 1, limit: 50 }} />

// Avoid: lost on navigation
const [limit, setLimit] = useState(50);
```

### LocalStorage

Currently not used. Future consideration for:

- User preferences (theme, language)
- Recently viewed events

---

## Testing State

Unit test state transitions:

```tsx
// Example: test error → retry → success flow
render(<EventDetailPage id="123" />);
expect(screen.getByText('Loading...')).toBeInTheDocument();

await waitFor(() => {
  expect(screen.getByText('Error')).toBeInTheDocument();
});

fireEvent.click(screen.getByText('Retry'));

await waitFor(() => {
  expect(screen.getByText('Event Content')).toBeInTheDocument();
});
```

---

## Performance Considerations

### Avoid

- Storing large datasets in state
- Unnecessary re-renders (use proper deps arrays)
- Deep object mutations

### Prefer

- URL for shared state
- Normalized data shapes
- Memoization for expensive calculations

---

## Future Improvements

1. **TanStack Query**: Add for server state caching
2. **React 19 Actions**: Migrate form submissions
3. **Optimistic UI**: Add for mutations
4. **Error Tracking**: Integrate Sentry for production errors
5. **x-request-id**: Propagate from API responses to error UI

## Planned Blossom Workspace State Flow

```
┌──────────────────────────────────────────────────────────────┐
│                    Blossom Workspace Flow                    │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  route /blossom + URL search params                          │
│       │                                                      │
│       ├── tab, view, sha256, mime_type, extension            │
│       ├── review_state, pubkey, uploader_q                   │
│       ├── report_type, report_status, worker_status          │
│       └── limit, offset                                      │
│       ▼                                                      │
│  BlossomWorkspace (Smart)                                    │
│       │                                                      │
│       ├── useBlossomOverview()                               │
│       ├── useBlossomPolicy()                                 │
│       ├── useBlossomObjects(filters)                         │
│       ├── useBlossomWorkers(filters)                         │
│       ├── no heavy reports/audit/review table by default     │
│       ├── useBlossomAnalytics() when modal opens             │
│       └── conditionally: useBlossomAudit/useBlossomUsers     │
│              │                                               │
│              ▼                                               │
│  view composition                                            │
│       ├── KPI strip + alerts                                 │
│       ├── policy card + editable filters                     │
│       ├── object table or thumbnail grid                     │
	│       ├── user quota table                                   │
	│       ├── mirror/worker monitor                              │
	│       ├── links to child review/reports/audit screens        │
	│       ├── workers modal                                      │
	│       └── analytics modal                                    │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

States:

| State | Meaning |
|-------|---------|
| `loading` | primary overview/object query in flight |
| `success-table` | object browser loaded in table mode |
| `success-grid` | object browser loaded in grid mode |
| `empty` | no objects/reports/users match active filters |
| `error` | one or more primary admin Blossom queries failed |
| `drawer-open` | object or user inspection sheet open |
| `workers-modal-open` | quick worker inspection overlay open |
| `analytics-modal-open` | server-backed analytics overlay open |

### Blossom mutations and recovery

| Mutation | Strategy | Recovery |
|----------|----------|----------|
| bulk approve/delete | TanStack Query mutation, no optimistic delete | invalidate overview, objects, review queue and audit |
| policy update | mutation from dedicated subroute with exclusive radio selection | invalidate overview, policy, users and object queries |
| plan create/update/delete | mutation driven by modal draft state | invalidate policy, plans, overview and audit |
| plan assignment | mutation from search modal | invalidate users, overview and audit |
| purge user | mutation + confirmation dialog | keep sheet open on error and preserve typed reason |
| quota/whitelist save | mutation with local form state preservation | inline field errors and retry |
| mirror request | mutation creates background job | invalidate workers and audit, show pending state immediately |
| report resolution | mutation from reports tab/sheet | invalidate reports, overview and audit |
| force optimization | mutation enqueues job | invalidate object detail and workers |

Error recovery rules:

- thumbnail or blurhash failure falls back to MIME badge + fixed placeholder, not broken image UI
- object detail fetch failure does not clear current list selection; the sheet shows retry
- audit and workers errors remain local panels when the library view is otherwise usable
- workers modal failure must preserve the last successful snapshot if available and show manual refresh
- analytics modal failure must degrade to retry UI instead of blocking the main workspace
- destructive mutation failures must surface `requestId` when available so operators can correlate backend logs
- custom MIME filter input remains controlled by URL state; invalid or uncommon MIME strings are treated as exact filters, not client-side validation errors

### Blossom Plans Subscreen Flow

```text
route /blossom/plans
	-> useBlossomPlans()
	-> local modal state (create/edit, delete confirm, assign)
	-> mutate save/delete/assign
	-> invalidate plans + users + policy + overview
```

UI state rules:

- unsaved editor fields stay local to each modal, not global store state
- user search inside the assignment modal is debounced and powered by the existing `/users/search` endpoint
- the plans screen does not edit policy state anymore; policy lives in `/blossom/policy`

### Blossom Review / Reports / Audit Child Flows

```text
route /blossom/review
  -> gated by useBlossomPolicy()
  -> if review disabled, render unavailable state
  -> else useBlossomObjects(review filters) + review mutations

route /blossom/reports
  -> useBlossomReports(filters)
  -> object drill-down still uses BlossomObjectSheet

route /blossom/audit
  -> useBlossomAudit(filters)
```

Visibility rule:

- `review` child navigation is rendered only when the effective policy mode requires manual review
