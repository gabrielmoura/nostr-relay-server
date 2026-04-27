# NIP-29 Coordination

## Objective

Implement NIP-29 group support as an optional, incremental and configurable module that preserves current relay behavior when disabled.

## Current Diagnosis

- The runtime already has suitable extension points in `cmd/server.go`, `internal/policies`, `infra/ingestion`, `infra/db`, `infra/handler/req`, `infra/handler/count`, and `infra/metrics`.
- Generic Nostr storage is already compatible with group content because the `event` table stores arbitrary kinds and indexes single-character tags, including `h`.
- The relay already supports global NIP-13 validation, batching, Redis cache/pubsub, and relay-generated signed events via the configured relay key.
- The repository schema does **not** currently declare NIP-29 tables, but the live PostgreSQL database already contains draft tables: `nip29_groups`, `nip29_roles`, `nip29_group_roles`, `nip29_group_members`.
- The current codebase does not yet wire those tables into runtime behavior.

## Reference Analysis

### `ref/relay29`

- Keeps group state in memory and reconstructs it from moderation events.
- Validates writes through a policy chain: `h` tag existence, recency, permissions, deleted-event protection, timeline references.
- Emits relay-generated metadata/admin/member/role events (`39000`-`39003`).
- Auto-processes join (`9021`) and leave (`9022`) requests.
- Treats timeline references (`previous`) as a rolling recent-id check over the last 50 group events.

### NIP-29

- Group content uses the `h` tag.
- Relay-generated state events use the `d` tag and must be relay-signed.
- Join/leave requests are optional but standardized.
- Moderation events (`9000`-`9009`) are relay/admin controlled.
- Timeline references and late-publication protection are optional but recommended.

### Runtime / Storage Findings

- PostgreSQL `event` table currently has ~4k rows and already stores kinds `9` and `39000` from external relays.
- Redis is available, has low memory pressure, and already serves cache/pubsub duties.
- Existing Redis key patterns suggest no current NIP-29-specific cache design.

## Decisions Taken

- Keep group message/event content in the existing `event` table.
- Add an optional groups module instead of modifying handler logic inline.
- Use PostgreSQL as source of truth for group state.
- Use Redis only for hot-path acceleration: membership, bans, group metadata, invite validation, timeline references.
- Prefer current package-level bootstrap style over introducing a new DI container now.

## Planned Incremental Architecture

1. Add `nip29` configuration block and runtime toggles.
2. Add repository/schema support for group state, bans and invites.
3. Introduce an optional groups manager package.
4. Hook group validation into policy hub and query flow.
5. Hook state mutation + relay-generated metadata emission into ingestion side effects.
6. Add Prometheus metrics.
7. Enable optional invite, PoW and timeline-reference protections.

## Implementation Status

- Done: config surface for `nip29.*` defaults and supported-NIP advertisement when enabled.
- Done: repository schema alignment for `nip29_groups`, `nip29_roles`, `nip29_group_roles`, `nip29_group_members`, `nip29_group_bans`, `nip29_group_invites`.
- Done: query helpers in `infra/db/nip29_query.go`.
- Done: optional runtime manager in `internal/groups`.
- Done: refactor of newly added Go files so each generated NIP-29 file stays below 300 lines, splitting manager/runtime concerns into focused files (`manager`, `validation`, `state`, `cache`, `apply*`, `helpers`) and DB access into focused query files.
- Done: root documentation updated in `README.md` and `nrserver.adoc` to describe the optional NIP-29 module, configuration, metrics and storage model.
- Done: NIP-11 configuration/docs expanded to cover optional relay fields, limitation block and enforced `max_subscriptions` behavior.
- Done: EVENT validation hook in `internal/policies`.
- Done: REQ/COUNT group-aware query filtering in `infra/handler/req` and `infra/handler/count`.
- Done: ingestion post-persist hook for group state mutation and relay-generated metadata emission.
- Done: Prometheus metrics for group lifecycle, rejections, processing latency and cache hits/misses.
- Done: optional invite, PoW and timeline-reference enforcement toggles.
- Done: fix metadata tag emission (`buildMetadataTags`) to always emit `["public"]`/`["open"]` when `Private`/`Closed` are false, matching NIP-29 spec and go-nostr reference.
- Done: fix `applyMetadataEdits` to recognize `["public"]`/`["open"]` antonym tags and stop unconditionally resetting `Restricted`/`Hidden` on every `edit-metadata`.
- Done: fix `applyCreateGroup` to not default group name to groupID; leave empty so the client-sent `kind:9002` sets the actual name without contamination from a premature `kind:39000`.
- Pending: dedicated tests for `internal/groups` and deeper validation of delete-event semantics inside group boundaries.
- Pending: re-emit `kind:39000` for groups whose persisted metadata event lacks `["public"]`/`["open"]` tags.

## Schema Direction

- Reuse live draft tables where practical: `nip29_groups`, `nip29_roles`, `nip29_group_roles`, `nip29_group_members`.
- Extend `nip29_groups` with missing policy columns where needed (`restricted`, `hidden`, PoW/timeline flags).
- Add `nip29_group_bans` for explicit ban semantics.
- Add `nip29_group_invites` for `kind:9009` support.

## Redis Direction

- `nip29:group:{relay}:{group_id}` for metadata/policy cache.
- `nip29:member:{relay}:{group_id}:{pubkey}` for membership lookup cache.
- `nip29:ban:{relay}:{group_id}:{pubkey}` for ban lookup cache.
- `nip29:invite:{relay}:{group_id}:{code}` for invite validation.
- `nip29:timeline:{relay}:{group_id}` for bounded recent event ids/prefixes.

## Risks

- Live DB already contains draft NIP-29 tables not represented in repository schema; migration order must avoid destructive drift.
- Relay-generated events require a configured relay private key whenever `nip29.enabled=true`.
- Private-group reads can leak if ID/reference queries are not filtered after DB lookup.
- Group state and generated metadata events can diverge if persistence and side effects are not coordinated carefully.

## Pending Work

- Add tests for validation and state transitions.
- Consider explicit admin tooling for `nip29_group_bans` lifecycle, since NIP-29 itself does not standardize a ban moderation kind.
- Consider stronger filtering for relay-generated metadata discovery mixed with external `39000`-`39003` events already present in the `event` table.

## Migrations Needed

- Repository schema must gain/align NIP-29 tables.
- Existing live `nip29_groups` table likely needs new columns for policy controls.
- New tables for bans and invites are required.

## Metrics Planned

- `nostr_nip29_groups_created_total`
- `nostr_nip29_groups_active`
- `nostr_nip29_events_received_total{kind}`
- `nostr_nip29_events_rejected_total{reason}`
- `nostr_nip29_invites_generated_total`
- `nostr_nip29_invites_consumed_total`
- `nostr_nip29_processing_seconds{operation}`
- `nostr_nip29_cache_total{cache,result}`

## Toggles Planned

- `nip29.enabled`
- `nip29.relay_scope`
- `nip29.cache_ttl_seconds`
- `nip29.membership_cache_ttl_seconds`
- `nip29.ban_cache_ttl_seconds`
- `nip29.timeline_cache_ttl_seconds`
- `nip29.group_creator_role`
- `nip29.default_roles[]`
- `nip29.create.*`
- `nip29.moderation.*`
- `nip29.admission.*`
- `nip29.invite.*`
- `nip29.pow.*`
- `nip29.timeline.*`
- `nip29.advanced.*`
- `nip29.permissions.*`

## Next Recommended Step

Add focused tests for group state transitions and validate the migration path against environments that already contain draft `nip29_*` tables.
