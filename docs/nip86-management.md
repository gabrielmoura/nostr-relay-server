# NIP-86 Management Plan

## Current Diagnosis

- External root `/` currently serves only two behaviors: NIP-11 on `Accept: application/nostr+json` and WebSocket upgrade for the relay protocol.
- Internal admin actions already exist on `/admin/*`, protected by optional `X-Admin-Token`.
- The codebase already has reusable parts for this feature:
  - `infra/handler/listener` tracks live websocket connections and can disconnect them by admin id.
  - `infra/cache` and `infra/redis` already provide exact-key TTL cache helpers.
  - `infra/handler/store/blossom/util.go` already decodes `Authorization: Nostr <base64-event>` and verifies event signatures.
  - `infra/db/profile_query.sql.go` and `infra/db/admin_query.go` already implement ban-related persistence patterns with `pgx`.
- There is no local `ref/nips` directory in the repository at the moment, so the protocol baseline comes from MCP `nostr` plus existing project docs.
- There is no current `admin_pubkey` configuration; only `admin_token` exists today.
- The implementation should stay dormant unless `nip86.enabled=true`.

## End-User Guidance

- If you only need the embedded dashboard, keep `nip86.enabled=false`.
- Enable NIP-86 only when you need remote relay-management automation from a trusted Nostr-native client or script.
- Treat `admin_pubkey` as a privileged operator identity, not as a convenience setting.
- Keep `relay_information.url` stable and externally correct before enabling NIP-86, otherwise NIP-98 `u` tag validation will fail.

## Schema Change Plan

### Reuse Without Change

- `profiles`
- `banned_users`
- `event`

### New Tables

1. `nip86_allowed_pubkeys`
2. `nip86_banned_events`
3. `nip86_blocked_ips`
4. `nip86_relay_metadata`

### Why These Tables

- `banned_users` already covers `banpubkey` / `unbanpubkey`.
- NIP-86 also needs explicit allowlist state, event-level moderation state, network block state, and runtime NIP-11 metadata overrides; none of these exist in PostgreSQL today.
- Storing relay metadata overrides in PostgreSQL avoids mutating `conf.yaml` at runtime and keeps behavior stable across restarts.

## Transport Plan

Root route decision order on external `/`:

1. If `Content-Type` is `application/nostr+json+rpc`, handle NIP-86 HTTP JSON-RPC.
2. Else if `Accept` contains `application/nostr+json`, return NIP-11.
3. Else if request is a WebSocket upgrade, continue with current relay behavior.
4. Else return upgrade-required / method-not-allowed response as appropriate.

## Configuration Plan

- `nip86.enabled`: feature flag for the whole management API
- `admin_pubkey`: required when NIP-86 is enabled
- `nip86.auth_window_seconds`: NIP-98 freshness window
- `nip86.cache_ttl_seconds`: TTL for blocked-IP / banned-event hot-path cache entries

## Authentication Plan

All NIP-86 requests require NIP-98 auth with stricter validation than the current Blossom helper:

- `kind == 27235`
- valid signature
- short freshness window
- `method` tag equals request method
- `u` tag equals exact absolute request URL
- `payload` tag equals SHA-256 hex of the raw JSON-RPC body
- `pubkey` equals configured `admin_pubkey`

## Service and Repository Shape

Keep abstractions small and local to the consumer.

### Planned Service Split

- `infra/handler/http/nip86.go` - HTTP decode/encode only
- `internal/nip86/service.go` - dispatcher + business orchestration
- `internal/nip86/auth.go` - NIP-98 verification helper
- `internal/nip86/types.go` - request/response/method constants
- `infra/db/nip86_pubkey_query.go` - ban/allow pubkey persistence
- `infra/db/nip86_event_query.go` - event moderation persistence
- `infra/db/nip86_ip_query.go` - IP block persistence
- `infra/db/nip86_metadata_query.go` - relay metadata overrides

### Repository Boundaries

- interfaces should be defined near the NIP-86 service, not in the DB package
- constructors should return concrete structs
- no new DI container; manual wiring in `cmd/server.go` / existing bootstrap style is enough

## Method Mapping

| NIP-86 method | Planned backing path |
|---|---|
| `supportedmethods` | static service list |
| `banpubkey` | `banned_users` + cache invalidation |
| `unbanpubkey` | `banned_users` delete + cache invalidation |
| `listbannedpubkeys` | query latest records from `banned_users` |
| `allowpubkey` | `nip86_allowed_pubkeys` upsert + cache invalidation |
| `unallowpubkey` | `nip86_allowed_pubkeys` delete + cache invalidation |
| `listallowedpubkeys` | `nip86_allowed_pubkeys` select |
| `allowevent` | remove from `nip86_banned_events` |
| `banevent` | `nip86_banned_events` upsert |
| `listbannedevents` | `nip86_banned_events` select |
| `changerelayname` | `nip86_relay_metadata` upsert |
| `changerelaydescription` | `nip86_relay_metadata` upsert |
| `blockip` | `nip86_blocked_ips` upsert + disconnect live matches |
| `unblockip` | `nip86_blocked_ips` delete |
| `listblockedips` | `nip86_blocked_ips` select |

## Runtime Side Effects

- `blockip` must disconnect current websocket sessions from the exact IP immediately after the transaction succeeds.
- Cache invalidation should stay targeted:
  - `ban:{pubkey}`
  - `allow:{pubkey}`
  - `banevent:{id}`
  - `blockip:{ip}`
- No broad Redis scans on request path.

## Logging Plan

Every successful admin mutation should emit structured JSON logs including:

- `action`
- `target`
- `admin`
- `reason`
- `ip`
- `request_id` when available

Example:

```json
{
  "action": "blockip",
  "target": "203.0.113.10",
  "admin": "<hex-pubkey>",
  "reason": "abuse",
  "ip": "198.51.100.7"
}
```

## Optimization Notes From MCP Checks

- PostgreSQL already has the core moderation/event tables but not the new NIP-86 tables.
- Redis is small (`~1.44 MB`) and currently uses `noeviction`; targeted TTL keys are safe, but this is not a place for large response caching.
- Prefer exact-key lookups and invalidation over set/list scans.
- Existing Redis keyspace already serves unrelated workloads, so new NIP-86 keys should stay well namespaced and low cardinality.

## Current Limitations

- The external NIP-86 surface is deliberately narrow and does not replace the internal admin dashboard.
- Immediate `blockip` disconnect is guaranteed only for connections visible to the local process.
- Multi-node deployments may require extra Redis/pubsub orchestration if immediate cross-node disconnect is required.
- Runtime relay metadata overrides persist in PostgreSQL but do not mutate `conf.yaml`.
