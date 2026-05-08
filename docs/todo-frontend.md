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
