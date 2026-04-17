# Frontend Architecture

## Overview

The admin dashboard (`infra/dash/`) is a React 19 + TypeScript SPA built with TanStack Router and i18next. It provides operational controls for the Nostr relay server.

## Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Framework** | React 19 | UI library with new hooks (useActionState, useOptimistic) |
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

Currently, API calls are embedded in route components. Future refactoring should extract them to `services/`.

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

## State Management

### Local State

- React `useState` for form inputs and UI state
- TanStack Router for URL state (query params, path params)

### Server State

Currently: direct fetch calls in route components.
Future: TanStack Query for caching, invalidation, and optimistic updates.

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