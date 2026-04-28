# Frontend Decisions

## ADR-F004: Generic Operational Jobs Board Reuses Existing Routes

**Status:** Proposed  
**Date:** 2026-04-28

### Context

The backend now exposes one shared queue model for download, sync and cron work. The dashboard currently treats download jobs with feature-specific cards and treats sync only as a toast-triggered background action. We need a frontend model that matches the durable backend job system without fragmenting the operator experience.

### Decision

Use one reusable jobs module and embed it into the existing operational routes first.

1. `/download` shows generic jobs filtered to `download.events`
2. `/sync` shows generic jobs filtered to `sync.negentropy`
3. a dedicated global `/jobs` route is deferred until there is clear operator demand

### Reasons

1. **Low-risk migration:** preserves familiar entry points for operators
2. **Reuse:** one board, one dialog, one mutation model for retry/cancel
3. **Truthful UX:** the UI reflects the backend queue lifecycle instead of per-feature simulations
4. **Docs-first scalability:** future cron/dead-letter views can reuse the same contracts

### Consequences

- ✅ download and sync converge on one jobs presentation model
- ✅ service layer and hooks stay compact and typed
- ✅ route-level orchestration stays aligned with current Smart/Dumb rules
- ⚠️ `/download` and `/sync` pages become denser and require careful visual hierarchy
- ⚠️ cron jobs will still be less visible until a broader `/jobs` route is added later
