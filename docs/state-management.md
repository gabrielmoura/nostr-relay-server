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

---

## Mutations and Actions

### Current Pattern (Direct Fetch)

```tsx
const handleSubmit = async (data: FormData) => {
  setIsSubmitting(true);
  try {
    await fetch('/api/admin/action', {
      method: 'POST',
      body: JSON.stringify(data)
    });
    onSuccess();
  } catch (err) {
    setError(err);
  } finally {
    setIsSubmitting(false);
  }
};
```

### Future Pattern (React 19 Actions)

```tsx
const [state, formAction, isPending] = useActionState(
  async (prev, formData) => {
    const data = Object.fromEntries(formData);
    await post('/api/admin/action', data);
    return { status: 'success' };
  },
  { status: 'idle' }
);
```

### When to Use What

| Pattern | Use Case |
|---------|----------|
| Direct fetch | Simple one-off actions, no form |
| React 19 Action | Form submissions, pending state needed |
| TanStack Query | When caching, invalidation needed |

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