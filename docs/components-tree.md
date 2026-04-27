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

---

## Feature Components

### Event Detail (`components/features/event-detail/`)

| Component | Type | Props | Events |
|-----------|------|-------|--------|
| `EventMetadata` | Dumb | `event` | - |
| `EventMedia` | Dumb | `event` | - |
| `EventImageGrid` | Dumb | `images` | - |
| `EventVideoPlayer` | Dumb | `src`, `poster` | - |
| `EventRepostCard` | Dumb | `repostedEvent` | - |
| `ReactionTargetEvent` | Dumb | `targetEvent` | - |
| `EventListItems` | Dumb | `events`, `onEventClick` | `onEventClick` |
| `NostrReferences` | Dumb | `event` | - |
| `ListRefSyncCard` | Smart | `listId`, `onSync`, `onClose` | `onSync`, `onClose`, `onRelaySelect` |
| `RelaySearchModal` | Smart | `open`, `onClose` | `onSelect` |
| `EventDetailErrorState` | Dumb | `error`, `onRetry` | `onRetry` |

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

**Parser**: `lib/event-search.ts` (transforms API response to display data)

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
| `/nip86` | `nip86-page.tsx` | planned | NIP-86 command center |
| `/nip86/allowed` | `nip86-allowed-page.tsx` | planned | Allowlist management |
| `/nip86/blocked-ips` | `nip86-blocked-ips-page.tsx` | planned | IP blocking + disconnect visibility |
| `/nip86/banned-events` | `nip86-banned-events-page.tsx` | planned | Event moderation state |

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
