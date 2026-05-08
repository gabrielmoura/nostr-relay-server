# Frontend Architecture

## Overview

The admin dashboard (`infra/dash/`) is a React 19 + TypeScript SPA built with TanStack Router and i18next. It provides operational controls for the Nostr relay server.

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
| **State Management** | TanStack Query | Server state management, caching, and mutations |
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

### Service Layer (Future)

All API calls should go through a service layer:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Component  │────►│  Service    │────►│   API       │
│             │     │  (typed)    │     │  (fetch)    │
└─────────────┘     └─────────────┘     └─────────────┘
```

Currently, API calls already flow through `services/admin.ts` for most dashboard paths. New NIP-86-facing admin features should continue to use this service-first pattern.

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

## Current known gap

The current sync queue UX exposes `cancel`, but the backend/runtime behavior can still let a canceled item resume automatically later. The intended fix is:

- preserve `canceled` as a stable terminal state
- add an explicit `resume` action for canceled jobs
- keep the frontend board aligned with real backend semantics instead of simulating cancellation locally

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
