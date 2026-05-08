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
