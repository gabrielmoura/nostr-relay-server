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

### 4. Documentation
- [x] Create `docs/frontend-architecture.md`
- [x] Create `docs/components-tree.md`
- [x] Create `docs/state-management.md`

---

## In Progress

### Documentation Update
- [ ] Review and finalize `docs/frontend-architecture.md`
- [ ] Review and finalize `docs/components-tree.md`
- [ ] Review and finalize `docs/state-management.md`

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

### State Management
- [ ] Evaluate TanStack Query for server state
- [ ] Add React 19 Actions for form submissions
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