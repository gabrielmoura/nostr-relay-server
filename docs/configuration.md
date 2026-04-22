# Configuration Reference

This document describes the `conf.yaml` structure supported by the relay.

- Loader: `config.LoadConfig()`
- Source: `conf.yaml` (searched in `.`, `../..`, `/etc/nrs`)
- Parser: Viper (`yaml`)

## Quick Notes

- `db.postgres_uri` is required.
- If `app_env` is empty, runtime fallback is `production`.
- Cron expressions use **6 fields** (`sec min hour day month weekday`) because scheduler uses `cron.WithSeconds()`.
- `stream.stream_up` / `stream.stream_down` are the active keys.

Operational CLI helpers:

- `nrserver conf print` prints default template.
- `nrserver conf effective` prints loaded effective runtime configuration.
- `nrserver conf validate` validates required fields and enabled cron schedules.
- `nrserver conf write --file <path>` writes template config file.

## Full YAML Skeleton

```yaml
port: 4869
app_env: production
admin_token: ""

ws:
  rate_limit: 1
  burst: 5
  auth: false
  auth_mode: none

anon:
  i2p: ""
  enable_i2p: false

relay_information:
  url: http://localhost:9090
  name: Nostr Relay Server
  description: A Nostr Relay Server
  pub_key: ""
  priv_key: ""
  contact: ""
  supported_nips: [11, 1, 2, 4, 25]
  software: https://github.com/gabrielmoura/nostr-relay-server
  version: 0.1.0
  canonical_url: ws://localhost:9090
  icon: http://localhost:9090/nostr.png

relay:
  query_limit: 100
  query_ids_limit: 500
  query_authors_limit: 500
  query_kinds_limit: 10
  query_tags_limit: 100
  keep_recent_events: true
  max_size_event_in_bytes: 100000
  filter_limit: 99999999
  reporting_limit: 5
  enable_anonymous_req: true
  max_tag_value_length: 100
  protected_kinds: [30078, 4]
  minimum_pow_limit: 0
  fake_deletion: false
  vanish_event: false
  enable_empty_filter: false

db:
  max_conns: 10
  min_conns: 1
  max_conn_lifetime_minutes: 30
  max_conn_idle_minutes: 5
  health_check_period_seconds: 30
  postgres_uri: postgres://user:password@localhost:5432/dbname

redis:
  enabled: false
  addr: 127.0.0.1:6379
  password: ""
  db: 0
  pool_size: 10
  subscription_cleanup_interval_seconds: 60
  subscription_stale_after_seconds: 120
  cache:
    ban_ttl: 3600
    profile_ttl: 300
    query_ttl: 30
    query_meta_ttl: 30
    event_ttl: 600
    dedup_ttl: 3600

ingestion:
  batch_size: 1000
  batch_timeout_ms: 100
  workers: 4
  queue_size: 10000

cron:
  enabled: true
  db_optimization:
    enabled: false
    schedule: "0 30 3 * * *"
    analyze: true
    vacuum_analyze: false
    reindex_event: false
  reported_events_fetch:
    enabled: false
    schedule: "0 */30 * * * *"
    relays: []
    lookback_hours: 24
    limit_per_relay: 200
  delete_old_events:
    enabled: false
    schedule: "0 0 4 * * *"
    older_than_days: 365
    batch_size: 2000
  nip40:
    enabled: false
    schedule: "0 */15 * * * *"
    batch_size: 2000

stream:
  relays: []
  stream_up: true
  stream_down: false

enable_negentropy: false

store:
  enabled: false
  api_path: http://localhost:9090/upload
  media_path: http://localhost:9090/blob
  accepted_mimetypes: []
  allow_adult_content: false
  allow_violent_content: false
  names: []

nip29:
  enabled: false
  relay_scope: ""
  cache_ttl_seconds: 60
  membership_cache_ttl_seconds: 30
  ban_cache_ttl_seconds: 30
  timeline_cache_ttl_seconds: 300
  group_creator_role: admin
  default_roles:
    - name: admin
      description: Full group administration
    - name: moderator
      description: Moderation without ownership
  create:
    enabled: true
    max_groups_per_pubkey: 10
  moderation:
    allow_private_groups: true
    require_recent_moderation: true
    recent_window_seconds: 60
  admission:
    default_closed: false
    default_private: false
    default_restricted: false
    default_hidden: false
    require_membership_for_write: true
  invite:
    enabled: false
    default_max_uses: 1
    default_ttl_seconds: 86400
    allow_multi_use: false
  pow:
    enabled: false
    default_min_difficulty: 0
    moderation_min_difficulty: 0
  timeline:
    enabled: false
    required_on_moderation: false
    min_references: 0
    recent_window: 50
  advanced:
    emit_member_list_events: true
    emit_role_events: true
    cache_membership_lookup: true
    cache_group_metadata: true
```

## Key-by-Key Reference

### Root

| Key | Type | Default | Description |
|---|---|---:|---|
| `port` | int | `9090` | External server port. Internal server uses `port+1`. |
| `app_env` | string | `production` (runtime fallback) | Environment mode. |
| `admin_token` | string | `""` | If set, `/admin` requires `X-Admin-Token`. |
| `enable_negentropy` | bool | `false` | Enables Negentropy flow (`NEG-OPEN`, `NEG-MSG`, `NEG-CLOSE`) and related sync handlers. |

### Negentropy and Sync Operational Notes

When `enable_negentropy=true`:

- The WebSocket router accepts Negentropy messages and opens reconciliation sessions.
- The relay can interoperate with peers using legacy `NEG-HAVE` / `NEG-NEED` as well as Strfry-style data transfer (`EVENT` + `REQ`).
- The sync CLI (`nrserver sync`) performs reconciliation and event transfer with batched REQ ids to reduce rejection risk on strict relays.

Recommended production posture:

- Keep `relay.query_ids_limit` consistent with your expected sync profile.
- Monitor Negentropy-specific metrics on `/metrics`:
  - `nostr_negentropy_v2_requests_total{operation,result}`
  - `nostr_negentropy_v2_cache_total{backend,result}`
  - `nostr_negentropy_v2_sessions_active`
  - `nostr_negentropy_v2_protocol_errors_total`
  - `nostr_negentropy_v2_events_imported_total`
- If Redis is enabled, Negentropy V2 cache paths can leverage Redis TTL-backed storage; otherwise memory cache is used.

### `ws`

| Key | Type | Default | Description |
|---|---|---:|---|
| `ws.rate_limit` | number | `1` | WS request rate limiter (token/sec). |
| `ws.burst` | int | `5` | Rate limiter burst. |
| `ws.auth` | bool | `false` | Legacy compatibility flag. If `true` and `ws.auth_mode` is empty, mode behaves as `strict`. |
| `ws.auth_mode` | string | `none` | Authentication mode: `strict`, `flexible`, `optional`, `none`. |

Authentication modes:

- `strict`: authentication required for everything (`REQ`/`EVENT`).
- `flexible`: authentication required for `EVENT`; `REQ` remains open.
- `optional`: authentication is accepted as identity, but not required.
- `none`: authentication disabled (default).

### `anon`

| Key | Type | Default | Description |
|---|---|---:|---|
| `anon.i2p` | string | `""` | I2P endpoint metadata. |
| `anon.enable_i2p` | bool | `false` | Enables I2P mode. |

### `relay_information`

| Key | Type | Default | Description |
|---|---|---:|---|
| `relay_information.url` | string | `http://localhost:<port>` | Public relay info URL. |
| `relay_information.name` | string | `Nostr Relay Server` | Display name. |
| `relay_information.description` | string | `A Nostr Relay Server` | Description for NIP-11. |
| `relay_information.pub_key` | string | `""` | Relay pubkey. |
| `relay_information.priv_key` | string | `""` | Relay private key (keep secret). |
| `relay_information.contact` | string | `""` | Contact metadata. |
| `relay_information.supported_nips` | int[] | `[11,1,2,4,25]` | Advertised supported NIPs. |
| `relay_information.software` | string | repo URL | Software URL. |
| `relay_information.version` | string | `0.1.0` | Version string. |
| `relay_information.canonical_url` | string | `ws://localhost:<port>/relay` | Canonical websocket URL. |
| `relay_information.icon` | string | `http://localhost:<port>/nostr.png` | Relay icon URL. |

### `relay`

| Key | Type | Default | Description |
|---|---|---:|---|
| `relay.query_limit` | int | `100` | Max events returned per query. |
| `relay.query_ids_limit` | int | `500` | Max IDs in filter. |
| `relay.query_authors_limit` | int | `500` | Max authors in filter. |
| `relay.query_kinds_limit` | int | `10` | Max kinds in filter. |
| `relay.query_tags_limit` | int | `100` | Max tag filters. |
| `relay.max_tag_value_length` | int | `100` | Max tag value length accepted. |
| `relay.keep_recent_events` | bool | `true` | Keep recency strategy enabled. |
| `relay.max_size_event_in_bytes` | int | `100000` | Max event payload size. |
| `relay.filter_limit` | int | `99999999` | Internal filter guard limit. |
| `relay.reporting_limit` | int64 | `5` | Reports threshold for moderation actions. |
| `relay.enable_anonymous_req` | bool | `true` | Allow unauthenticated REQ. |
| `relay.protected_kinds` | int[] | `[30078,4]` | Kinds requiring stricter checks. |
| `relay.minimum_pow_limit` | int | `0` | Minimum POW required. |
| `relay.fake_deletion` | bool | `false` | Soft-delete instead of hard delete. |
| `relay.vanish_event` | bool | `false` | Enable vanish-event behavior. |
| `relay.enable_empty_filter` | bool | zero value (`false`) | Allow empty filters in REQ. |

### `db`

| Key | Type | Default | Description |
|---|---|---:|---|
| `db.max_conns` | int32 | `10` | pgx pool max connections. |
| `db.min_conns` | int32 | `1` | pgx pool min connections. |
| `db.max_conn_lifetime_minutes` | int32 | `30` | Connection max lifetime. |
| `db.max_conn_idle_minutes` | int32 | `5` | Idle connection eviction. |
| `db.health_check_period_seconds` | int32 | `30` | pgx health check period. |
| `db.postgres_uri` | string | **required** | PostgreSQL DSN. |

### `redis`

| Key | Type | Default | Description |
|---|---|---:|---|
| `redis.enabled` | bool | `false` | Enable Redis integration. |
| `redis.addr` | string | `127.0.0.1:6379` | Redis address. |
| `redis.password` | string | `""` | Redis password. |
| `redis.db` | int | `0` | Redis DB index. |
| `redis.pool_size` | int | `10` | Redis pool size. |
| `redis.subscription_cleanup_interval_seconds` | int | `60` | Cleanup interval. |
| `redis.subscription_stale_after_seconds` | int | `120` | Stale subscription threshold. |

`redis.cache`:

| Key | Type | Default |
|---|---|---:|
| `redis.cache.ban_ttl` | int | `3600` |
| `redis.cache.profile_ttl` | int | `300` |
| `redis.cache.query_ttl` | int | `30` |
| `redis.cache.query_meta_ttl` | int | `30` |
| `redis.cache.event_ttl` | int | `600` |
| `redis.cache.dedup_ttl` | int | `3600` |

### `ingestion`

| Key | Type | Default | Description |
|---|---|---:|---|
| `ingestion.batch_size` | int | `1000` | Batch insert size. |
| `ingestion.batch_timeout_ms` | int | `100` | Max wait to flush batch. |
| `ingestion.workers` | int | `4` | Ingestion worker count. |
| `ingestion.queue_size` | int | `10000` | Ingestion queue size. |

### `cron`

Global switch:

| Key | Type | Default | Description |
|---|---|---:|---|
| `cron.enabled` | bool | `true` | Enables cron scheduler command jobs. |

`cron.db_optimization`:

| Key | Type | Default | Description |
|---|---|---:|---|
| `enabled` | bool | `false` | Run DB optimization routine. |
| `schedule` | string | `0 30 3 * * *` | Cron expression (with seconds). |
| `analyze` | bool | `true` | Run `ANALYZE`. |
| `vacuum_analyze` | bool | `false` | Run `VACUUM (ANALYZE)`. |
| `reindex_event` | bool | `false` | Run `REINDEX TABLE CONCURRENTLY event`. |

`cron.reported_events_fetch`:

| Key | Type | Default | Description |
|---|---|---:|---|
| `enabled` | bool | `false` | Enable automatic fetch of report events (kind 1984). |
| `schedule` | string | `0 */30 * * * *` | Cron expression. |
| `relays` | string[] | `[]` | Explicit relay list (required when enabled). |
| `lookback_hours` | int | `24` | Lookback window for report search. |
| `limit_per_relay` | int | `200` | Max fetched events per relay run. |

`cron.delete_old_events`:

| Key | Type | Default | Description |
|---|---|---:|---|
| `enabled` | bool | `false` | Enable retention cleanup. |
| `schedule` | string | `0 0 4 * * *` | Cron expression. |
| `older_than_days` | int | `365` | Delete events older than N days. |
| `batch_size` | int | `2000` | Batch deletion chunk size. |

`cron.nip40`:

| Key | Type | Default | Description |
|---|---|---:|---|
| `enabled` | bool | `false` | Enable NIP-40 expiration cleanup (`expiration` tag). |
| `schedule` | string | `0 */15 * * * *` | Cron expression for expiration cleanup. |
| `batch_size` | int | `2000` | Batch deletion chunk size per run. |

### `stream`

| Key | Type | Default | Description |
|---|---|---:|---|
| `stream.relays` | string[] | `[]` | Relay pool targets for forwarding. |
| `stream.stream_up` | bool | `true` | Enable upstream forward of selected kinds. |
| `stream.stream_down` | bool | `false` | Enable downstream REQ backfill via pool. |

### `store`

| Key | Type | Default | Description |
|---|---|---:|---|
| `store.enabled` | bool | `false` | Enable Blossom storage mode. |
| `store.api_path` | string | `http://localhost:<port>/upload` | Upload endpoint URL. |
| `store.media_path` | string | `http://localhost:<port>/blob` | Download base URL. |
| `store.accepted_mimetypes` | string[] | prefilled | Allowed MIME list for uploads. |
| `store.allow_adult_content` | bool | zero value (`false`) | Content policy toggle. |
| `store.allow_violent_content` | bool | zero value (`false`) | Content policy toggle. |
| `store.names` | string[] | zero value (`[]`) | Custom names/tags. |

### `nip29`

`nip29` is fully optional. When `nip29.enabled=false`, the relay must behave exactly as today.

| Key | Type | Default | Description |
|---|---|---:|---|
| `nip29.enabled` | bool | `false` | Enables the optional NIP-29 groups module. |
| `nip29.relay_scope` | string | `""` | Explicit logical relay scope for group state; defaults to canonical relay identity when empty. |
| `nip29.cache_ttl_seconds` | int | `60` | Default metadata/state cache TTL. |
| `nip29.membership_cache_ttl_seconds` | int | `30` | Membership lookup cache TTL. |
| `nip29.ban_cache_ttl_seconds` | int | `30` | Group ban lookup cache TTL. |
| `nip29.timeline_cache_ttl_seconds` | int | `300` | TTL for recent timeline references in Redis. |
| `nip29.group_creator_role` | string | `admin` | Role granted to group creator. |

`nip29.create`:

| Key | Type | Default | Description |
|---|---|---:|---|
| `enabled` | bool | `true` | Allows relay-side group creation. |
| `max_groups_per_pubkey` | int | `10` | Hard limit for groups created per pubkey. |

`nip29.moderation`:

| Key | Type | Default | Description |
|---|---|---:|---|
| `allow_private_groups` | bool | `true` | Allows `private` groups to be created/edited. |
| `require_recent_moderation` | bool | `true` | Reject stale moderation actions. |
| `recent_window_seconds` | int | `60` | Recency window for moderation actions. |

`nip29.admission`:

| Key | Type | Default | Description |
|---|---|---:|---|
| `default_closed` | bool | `false` | Default join policy for new groups. |
| `default_private` | bool | `false` | Default read policy for new groups. |
| `default_restricted` | bool | `false` | Default write policy for new groups. |
| `default_hidden` | bool | `false` | Default metadata visibility policy. |
| `require_membership_for_write` | bool | `true` | Default write restriction policy. |

`nip29.invite`:

| Key | Type | Default | Description |
|---|---|---:|---|
| `enabled` | bool | `false` | Enables invite-code flow for `kind:9009` / `9021 code`. |
| `default_max_uses` | int | `1` | Default invite use count. |
| `default_ttl_seconds` | int | `86400` | Default invite expiration. |
| `allow_multi_use` | bool | `false` | Allows multi-use invites when explicitly requested. |

`nip29.pow`:

| Key | Type | Default | Description |
|---|---|---:|---|
| `enabled` | bool | `false` | Enables NIP-13 PoW checks for groups. |
| `default_min_difficulty` | int | `0` | Default minimum PoW difficulty for group writes. |
| `moderation_min_difficulty` | int | `0` | Optional stricter PoW for moderation actions. |

`nip29.timeline`:

| Key | Type | Default | Description |
|---|---|---:|---|
| `enabled` | bool | `false` | Enables timeline reference enforcement. |
| `required_on_moderation` | bool | `false` | Requires `previous` on moderation actions. |
| `min_references` | int | `0` | Minimum accepted `previous` references. |
| `recent_window` | int | `50` | Number of recent group events tracked for validation. |

`nip29.advanced`:

| Key | Type | Default | Description |
|---|---|---:|---|
| `emit_member_list_events` | bool | `true` | Emit `39002` events after membership changes. |
| `emit_role_events` | bool | `true` | Emit `39003` events after role changes. |
| `cache_membership_lookup` | bool | `true` | Use Redis for membership lookup when available. |
| `cache_group_metadata` | bool | `true` | Use Redis for group metadata caching when available. |

## Production Example (Strict Retention + Report Fetch)

```yaml
cron:
  enabled: true
  db_optimization:
    enabled: true
    schedule: "0 15 3 * * *"
    analyze: true
    vacuum_analyze: false
    reindex_event: false
  reported_events_fetch:
    enabled: true
    schedule: "0 */20 * * * *"
    relays:
      - "wss://relay.damus.io"
      - "wss://nos.lol"
      - "wss://relay.primal.net"
    lookback_hours: 24
    limit_per_relay: 300
  delete_old_events:
    enabled: true
    schedule: "0 0 4 * * *"
    older_than_days: 365
    batch_size: 2000
  nip40:
    enabled: true
    schedule: "0 */15 * * * *"
    batch_size: 2000
```

## Validation Checklist

- `db.postgres_uri` is set and reachable.
- If `cron.reported_events_fetch.enabled=true`, `relays` is non-empty.
- If `cron.nip40.enabled=true`, review the schedule to avoid oversized cleanup windows.
- `stream.relays` contains only valid `ws://`/`wss://` URLs.
- `admin_token` is set in production internal networks.
- Retention (`older_than_days`) matches compliance requirements.
