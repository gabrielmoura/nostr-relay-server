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

---

## ADR-F005: NIP-32 Labels Use a Dedicated Admin Route and Backend-Signed Mutation Flow

**Status:** Proposed  
**Date:** 2026-05-06

### Context

The labels experience described in the reference project depends on browser-side Nostr queries and a separate publishing worker. The admin dashboard in this repository already standardizes on typed internal API calls plus TanStack Query.

We need a labels screen that fits the existing dashboard architecture and can create NIP-32 events without exposing relay signing to the browser.

### Decision

Add a dedicated `/labels` route in `infra/dash` and consume backend-signed admin endpoints.

1. `LabelsPage` stays the smart route entrypoint
2. `LabelsWorkspace` is the only feature-level smart container
3. all rendering blocks remain dumb and reusable
4. label creation uses TanStack Query mutation, not browser-side Nostr publish
5. optional ban chaining reuses the existing ban mutation after label creation succeeds

### Reasons

1. **Consistency:** matches the rest of the dashboard service layer.
2. **Traceability:** preserves `x-request-id` and server validation errors.
3. **Security:** keeps relay private-key usage on the backend.
4. **Reusability:** the same dialog can later appear in reported events and user detail flows.

### Consequences

- ✅ labels gain a native admin experience instead of a bolt-on external UI
- ✅ Smart/Dumb separation stays explicit
- ✅ server state remains cacheable and invalidatable through TanStack Query
- ⚠️ label creation cannot be optimistic because the signed event id is only known after backend success

---

## ADR-F006: Treat Sync Cancel as Terminal and Resume as Explicit Operator Action

**Status:** Proposed  
**Date:** 2026-05-06

### Context

The generic jobs board now exposes cancel actions on `/sync`, but the observed runtime behavior allows some canceled items to resume automatically later. This breaks operator trust because the UI communicates a terminal action while the backend queue may still continue work.

### Decision

1. sync cancel must be treated as terminal state in both backend and frontend semantics
2. a separate explicit `resume` action must be introduced for canceled sync jobs
3. `NostrFilterBuilder` inputs should accept either hex or NIP-19 when the domain concept allows both, reducing operator friction across search and moderation flows

### Reasons

1. **Trust:** operators need cancellation to mean stop
2. **Auditability:** resume becomes an intentional new action instead of hidden queue behavior
3. **Consistency:** NIP-19-aware inputs align operational tools with how Nostr operators actually copy identifiers

### Consequences

- ✅ sync job lifecycle becomes easier to reason about
- ✅ frontend actions map more honestly to backend state
- ✅ filter and target fields become friendlier to real Nostr workflows
- ⚠️ backend queue semantics may need changes beyond the dashboard layer

---

## ADR-F007: Expand Operator Inputs and Rich Event Context Instead of Requiring Canonical Raw Data

**Status:** Proposed  
**Date:** 2026-05-06

### Context

Operators naturally work with copied Nostr identifiers like `npub`, `note`, `nevent`, `nprofile` and `naddr`, not only raw hex. They also need richer context for community events (`kind:34550`) and queued sync work without drilling into raw JSON every time.

### Decision

1. `NostrFilterBuilder` should accept NIP-19 or hex wherever the concept allows both
2. labels target input keeps normalizing profile/event identifiers before submission
3. `/events/search` and `/events/$eventId` should render `kind:34550`-specific metadata (`d`, `description`, `image`, moderators)
4. sync job cards/details should expose the filters used and allow reenqueue/resume paths explicitly

### Reasons

1. **Operator ergonomics:** matches real Nostr workflows
2. **Moderation speed:** reduces hops to inspect relevant metadata
3. **Safety:** normalization still preserves canonical backend contracts

### Consequences

- ✅ search, sync and labels inputs become easier to use
- ✅ community events become more interpretable in moderation flows
- ✅ job actions become more auditable and explicit
- ⚠️ frontend adapters and backend contracts must stay tightly aligned on normalization rules

---

## ADR-F008: Sync Job Modal Uses Curated Panels for Filter and Relay Rejections

**Status:** Proposed  
**Date:** 2026-05-08

### Context

The generic jobs dialog already exposes raw payload/result JSON, but operators reviewing sync failures need two answers immediately: which filter was executed and which relay rejections happened. Requiring manual JSON inspection slows triage and hides important diagnostics in noisy blobs.

### Decision

1. keep the generic dialog shell in `JobsBoard`
2. add curated sync-only panels inside the modal for executed filter and rejection diagnostics
3. keep raw payload/result JSON panels as secondary, still-visible debug surfaces
4. prefer structured backend data from `job.result` over reconstructing meaning from `last_error`

### Reasons

1. **Speed:** the operator sees the important sync context first.
2. **Safety:** the UI reflects persisted backend truth instead of re-deriving diagnostics heuristically.
3. **Reuse:** the generic jobs route structure remains intact while only the modal body gains sync-specific branches.

### Consequences

- ✅ `/panel/sync` gets a clearer operational drill-down without a new route
- ✅ raw JSON remains available for deep debugging
- ⚠️ sync-specific modal content must stay isolated so `/download` and other jobs do not inherit irrelevant UI noise
