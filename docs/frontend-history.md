# Frontend Change History

## 2026-05-06 - Admin moderation and jobs UX expansion

### Delivered context

- introduced `/labels` as the native NIP-32 management route in `infra/dash`
- added backend-signed label creation flow through internal admin endpoints
- added labels filters, presets, summary cards, timeline view and grouped-by-target view
- added contextual help modal for the labels form
- normalized NIP-19 target input for labels where applicable
- added local clear-history actions to `/download` and `/sync` jobs boards
- added KPI strips to `/events/reported` and `/users/search`
- expanded `/events/$eventId` with labels, reports, replies, responder identities and associated event references

### Current known behavior

- jobs clear-history is currently UI-local and hides entries from the board instead of deleting backend job state
- `/sync` currently has a bug where canceling a queued/running item may later allow it to resume automatically depending on backend queue behavior
- labels targeting a profile is supported through `target.type = pubkey`; the UI should accept `npub` and `nprofile` and normalize them to hex before submission

### Next documented follow-ups

- extend `NostrFilterBuilder` to accept both hex and NIP-19 forms in all relevant fields
- fix sync queue cancel semantics so canceled jobs remain terminal
- add an explicit resume action for canceled sync jobs
- keep changelog entries updated when moderation and operations UX changes again

## 2026-05-06 - Follow-up scope after operator review

### New requested scope

- `NostrFilterBuilder` must accept NIP-19 or hex consistently
- `/sync` must stop auto-resuming canceled jobs
- `/sync` must expose explicit resume for canceled jobs
- `/sync` needs a strict but configurable concurrency cap per remote relay
- `/sync` jobs must show the filters used
- `/sync` detail modal must show the exact normalized filter used by the queued job
- `/sync` detail modal must show structured relay rejection details when the remote returns `OK ... false ...`
- completed jobs should expose reenqueue from the board
- labels filtering must support multiple simultaneous labels in backend/API
- jobs history clearing must become real backend cleanup, not UI-only hiding
- `/events/search` must promote KPI cards above filters and render kind `34550` metadata (`d`, `description`, `image`)
- `/events/$eventId` must render richer kind `34550` metadata, infinite reply events, richer responder cards and moderator identities

### Explicit product clarification captured

- yes, labeling a profile is supported
- for `target.type = pubkey`, the labels UI should accept `hex`, `npub` and `nprofile`
