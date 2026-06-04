# Data Schema

## Overview

This document describes both PostgreSQL and Redis data structures:
- **PostgreSQL**: Persistent storage for events, profiles, and file metadata
- **Redis**: Cache, pub/sub, and subscription state management

Admin GraphQL migration note:

- the planned GraphQL admin API reuses the current PostgreSQL and Redis structures
- this phase does not introduce new tables, Redis keys, or persistence formats
- resolver design should map transport fields to the existing admin query/runtime models already used by REST

---

## PostgreSQL Tables

## Tables

### 1. `event` - Nostr Events

Primary storage for all Nostr events.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | TEXT | NOT NULL, UNIQUE | Event ID (SHA256 hash) |
| `pubkey` | TEXT | NOT NULL | Author public key |
| `created_at` | INTEGER | NOT NULL | Unix timestamp |
| `kind` | INTEGER | NOT NULL | Event kind number |
| `tags` | JSONB | NOT NULL | Event tags array |
| `content` | TEXT | NOT NULL | Event content |
| `sig` | TEXT | NOT NULL | Event signature |
| `tagvalues` | TEXT[] | GENERATED | Extracted tag values |
| `content_search` | TSVECTOR | GENERATED | Full-text search index |
| `deleted_by` | VARCHAR(64) | NULL | Deletion event ID (fake deletion) |

**Indexes:**
- `ididx` - B-tree on `id` (text_pattern_ops)
- `pubkeyprefix` - B-tree on `pubkey` (text_pattern_ops)
- `timeidx` - B-tree on `created_at` DESC
- `kindidx` - B-tree on `kind`
- `kindtimeidx` - B-tree on `(kind, created_at DESC)`
- `idx_event_pubkey_created_at` - B-tree on `(pubkey, created_at DESC)`
- `idx_event_author_kind` - B-tree on `(pubkey, kind, created_at DESC)`
- `idx_event_deletions` - partial B-tree on `(created_at DESC, id)` when `deleted_by IS NOT NULL`
- `idx_event_tags_gin` - GIN on `tags` with `jsonb_path_ops`
- `content_search_idx` - GIN on `content_search`
- `idx_event_created_at_id` - B-tree on `(created_at, id)`

**Operational note:**
- do not use B-tree covering indexes with `INCLUDE (content, tags, sig)` on `event`; these payload columns can exceed PostgreSQL's per-index-row limit and break ingestion

**Generated Columns:**
- `tagvalues`: Extracted via `tags_to_tagvalues()` function (retained as helper/compatibility column, not as the primary exact tag-filter path)
- `content_search`: Portuguese full-text search vector

### 2. `profiles` - User Profiles

Metadata extracted from Kind 0 events.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Auto-increment ID |
| `public_key` | VARCHAR(64) | NOT NULL, UNIQUE | Public key (64 chars) |
| `name` | TEXT | NOT NULL | Display name |
| `about` | TEXT | NULL | Bio/description |
| `picture` | TEXT | NULL | Profile picture URL |
| `bot` | BOOLEAN | DEFAULT FALSE | Is bot account |
| `banner` | TEXT | NULL | Banner image URL |
| `website` | TEXT | NULL | Website URL |
| `display_name` | TEXT | NULL | Alternative display name |
| `lud16` | TEXT | NULL | Lightning address |
| `pronouns` | TEXT | NULL | Pronouns |
| `nip05` | TEXT | NULL | NIP-05 identifier |
| `enable_store_files` | BOOLEAN | DEFAULT FALSE | File storage enabled |
| `enable_nip05` | BOOLEAN | DEFAULT FALSE | NIP-05 verified |

**Indexes:**
- `idx_profiles_name` - B-tree on `name`
- `idx_profiles_nip05` - B-tree on `nip05`
- `idx_profiles_display_name` - B-tree on `display_name`

### 3. `banned_users` - Ban Management

Stores banned users and ban reasons.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Auto-increment ID |
| `user_id` | BIGINT | NOT NULL, FK(profiles.id) | Reference to profile |
| `reason` | TEXT | NOT NULL | Ban reason |
| `related_ids` | VARCHAR(60)[] | NULL | Related event IDs |

**Indexes:**
- `idx_banned_users_user_id` - B-tree on `user_id`
- `idx_banned_users_id` - B-tree on `id`

### 4. `objects` - Blossom File Metadata

Stores metadata for uploaded files (NIP-96).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `hash` | VARCHAR(64) | PRIMARY KEY | SHA256 file hash |
| `created_at` | TIMESTAMPTZ | NOT NULL | Upload timestamp |
| `mime_type` | VARCHAR(255) | NULL | File MIME type |
| `size` | BIGINT | NULL | File size in bytes |
| `blocked` | BOOLEAN | NULL | Content blocked flag |
| `expires_at` | TIMESTAMPTZ | NULL | Expiration timestamp |
| `blocked_by_reason` | TEXT | NULL | Block reason |
| `public_key` | VARCHAR(64) | NOT NULL | Uploader public key |
| `tags` | JSONB | NULL | Additional tags |

**Indexes:**
- `idx_objects_mime_type` - B-tree on `mime_type`
- `idx_objects_blocked` - B-tree on `blocked`

### 5. `nip29_groups` - Group State

Authoritative state for NIP-29 groups when the optional groups module is enabled.

Recommended shape:

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `relay` | TEXT | PK component | Relay scope (`canonical_url` or normalized relay identity) |
| `group_id` | TEXT | PK component | NIP-29 group id (`h`/`d` tag value) |
| `name` | VARCHAR | NOT NULL | Display name |
| `picture` | TEXT | NULL | Group picture |
| `about` | TEXT | NULL | Group description |
| `private` | BOOLEAN | NOT NULL DEFAULT FALSE | Read restricted to members |
| `closed` | BOOLEAN | NOT NULL DEFAULT FALSE | Join requests ignored unless invite policy allows |
| `restricted` | BOOLEAN | NOT NULL DEFAULT FALSE | Write restricted to members |
| `hidden` | BOOLEAN | NOT NULL DEFAULT FALSE | Metadata hidden from non-members |
| `require_moderation_timeline_ref` | BOOLEAN | NOT NULL DEFAULT FALSE | Enforce `previous` references on moderation events |
| `min_pow` | INTEGER | NOT NULL DEFAULT 0 | Minimum PoW difficulty override |
| `last_metadata_update` | TIMESTAMPTZ | NOT NULL | Last 39000 refresh timestamp |
| `last_admins_update` | TIMESTAMPTZ | NOT NULL | Last 39001 refresh timestamp |
| `last_members_update` | TIMESTAMPTZ | NOT NULL | Last 39002 refresh timestamp |
| `last_roles_update` | TIMESTAMPTZ | NOT NULL | Last 39003 refresh timestamp |

Suggested indexes:

- `idx_groups_name` on `(name)`
- `idx_groups_last_metadata_update` on `(last_metadata_update)`
- `idx_groups_last_members_update` on `(last_members_update)`

### 6. `nip29_roles` / `nip29_group_roles` - Supported Roles

These tables separate role catalog from group-to-role assignment.

- `nip29_roles` stores stable role definitions (`role_id`, `name`, `description`)
- `nip29_group_roles` stores which roles are valid for a group

This avoids repeating descriptions on every member row and keeps relay policy changes local.

### 7. `nip29_group_members` - Membership and Role Assignment

Membership rows should support fast checks by `(relay, group_id, user_id)` and allow a member to have multiple roles.

Operational recommendation:

- use one row per `(relay, group_id, user_id, role_id)` for role assignment
- avoid using the `banned` flag as the primary ban mechanism; keep bans in a separate table for clean semantics and simpler indexes
- add an index on `(relay, group_id, user_id)` for hot membership lookups

### 8. `nip29_group_bans` - Explicit Ban State

Recommended addition for production clarity.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `relay` | TEXT | PK component | Relay scope |
| `group_id` | TEXT | PK component | Group id |
| `user_id` | VARCHAR(64) | PK component | Banned pubkey |
| `reason` | TEXT | NULL | Optional audit reason |
| `created_by` | VARCHAR(64) | NULL | Moderator/admin pubkey |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Ban timestamp |

Suggested indexes:

- PK on `(relay, group_id, user_id)`
- optional index on `(relay, user_id)` for admin tooling

### 9. `nip29_group_invites` - Invite Codes (Optional)

Recommended for `kind:9009` support.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `relay` | TEXT | PK component | Relay scope |
| `group_id` | TEXT | PK component | Group id |
| `code` | TEXT | PK component | Invite code |
| `created_by` | VARCHAR(64) | NOT NULL | Creator pubkey |
| `max_uses` | INTEGER | NOT NULL DEFAULT 1 | Allowed uses |
| `uses` | INTEGER | NOT NULL DEFAULT 0 | Consumed uses |
| `expires_at` | TIMESTAMPTZ | NULL | Optional expiration |
| `revoked_at` | TIMESTAMPTZ | NULL | Revocation marker |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Creation timestamp |
| `last_used_at` | TIMESTAMPTZ | NULL | Last redemption timestamp |

Suggested indexes:

- PK on `(relay, group_id, code)`
- `idx_group_invites_expires_at` on `(expires_at)` for cleanup
- `idx_group_invites_active_lookup` on `(relay, group_id, code, revoked_at, expires_at)` for validation

### 10. `nip86_allowed_pubkeys` - Explicit Allowlist

Required for `allowpubkey`, `unallowpubkey`, and `listallowedpubkeys`.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `pubkey` | VARCHAR(64) | PRIMARY KEY | Allowed pubkey |
| `reason` | TEXT | NULL | Optional audit reason |
| `created_by` | VARCHAR(64) | NOT NULL | Admin pubkey that applied the change |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Last update timestamp |

Suggested indexes:

- PK on `(pubkey)`
- optional `idx_nip86_allowed_pubkeys_updated_at` on `(updated_at DESC)`

### 11. `nip86_banned_events` - Event Ban State

Required for `banevent`, `allowevent`, `listbannedevents`, and moderation audit.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `event_id` | VARCHAR(64) | PRIMARY KEY | Banned event id |
| `reason` | TEXT | NULL | Optional audit reason |
| `created_by` | VARCHAR(64) | NOT NULL | Admin pubkey that applied the change |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Ban timestamp |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Last update timestamp |

Suggested indexes:

- PK on `(event_id)`
- optional `idx_nip86_banned_events_updated_at` on `(updated_at DESC)`

### 12. `nip86_blocked_ips` - Network Blocklist

Required for `blockip`, `unblockip`, and `listblockedips`.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `ip` | INET | PRIMARY KEY | Exact blocked IP |
| `reason` | TEXT | NULL | Optional audit reason |
| `created_by` | VARCHAR(64) | NOT NULL | Admin pubkey that applied the change |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Block timestamp |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Last update timestamp |

Suggested indexes:

- PK on `(ip)`
- optional `idx_nip86_blocked_ips_updated_at` on `(updated_at DESC)`

### 13. `nip86_relay_metadata` - Runtime NIP-11 Overrides

Required so `changerelayname` and `changerelaydescription` survive process restarts without editing `conf.yaml`.

Single-row table keyed by relay identity.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `relay_url` | TEXT | PRIMARY KEY | Logical relay id, preferably `relay_information.url` |
| `name` | TEXT | NULL | Runtime override for NIP-11 name |
| `description` | TEXT | NULL | Runtime override for NIP-11 description |
| `updated_by` | VARCHAR(64) | NOT NULL | Admin pubkey that made the latest change |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Last change timestamp |

## Functions

### `tags_to_tagvalues(JSONB) -> TEXT[]`

Extracts single-character tag values for indexing.

```sql
SELECT array_agg(t->>1) 
FROM (SELECT jsonb_array_elements($1) AS t) s 
WHERE length(t->>0) = 1;
```

## Database Relationships

```
┌─────────────┐       ┌──────────────┐
│   event     │       │   profiles   │
├─────────────┤       ├──────────────┤
│ pubkey ─────┼───────┤ public_key    │
└─────────────┘       └──────────────┘
       │
       │ (via banned_users)
       ▼
┌─────────────┐       ┌──────────────┐
│banned_users │       │   objects    │
├─────────────┤       ├──────────────┤
│ user_id ────┼───────┤              │
└─────────────┘       └──────────────┘
```

## Configuration

Database connection via `conf.yaml`:

```yaml
db:
  max_conns: 50      # Maximum connections in pool
  min_conns: 1       # Minimum connections
  postgres_uri: "postgres://user:pass@host:5432/dbname"
  max_conn_lifetime_minutes: 30
  max_conn_idle_minutes: 5
  health_check_period_seconds: 30
```

## Queries

### Event Queries

The system builds dynamic SQL queries based on Nostr filters:

- Filter by `authors` (pubkeys)
- Filter by `ids`
- Filter by `kinds`
- Filter by `tags` (e, p, d, etc.)
- Filter by `since`/`until` timestamps
- Full-text search on `content`

### NIP-32 Labels Query Model

`kind:1985` label events remain in the shared `event` table.

No new relational table is planned for labels in the first rollout.

Administrative label queries will extract semantic fields from `tags JSONB` using `jsonb_array_elements(tags)`:

- namespace from the first `L` tag
- one or many label values from `l` tags
- target value from the first matching `e`, `p`, `a`, `r`, or `t` tag
- optional relay hint from the third position of `e` or `p`

This keeps the relay Nostr-native and avoids duplicating label state in a second schema.

Recommended label-query access patterns:

- `WHERE kind = 1985`
- exact namespace filter through `L`
- repeated label filters through `l` with OR semantics for dashboard multi-select UX
- exact target filter through `e` / `p` / `a` / `r` / `t`
- aggregation by namespace, label, and target type for dashboard counters

Profile-labeling support:

- labels targeting identities use the existing `p` tag model
- dashboard input may arrive as `npub` or `nprofile`, but storage/query remain canonical hex pubkeys in the event tags

### Count Queries

Same filters as event queries but returns `COUNT(*)`.

---

## Redis Data Structures

### Cache Keys

| Key Pattern | Type | TTL | Purpose | Example |
|-------------|------|-----|---------|---------|
| `ban:{pubkey}` | STRING | 1h | Banned user reason | `"spam"` |
| `allow:{pubkey}` | STRING | 1h | Allowed pubkey audit marker | `"ops allowlist"` |
| `banevent:{id}` | STRING | 10m | Banned event audit marker | `"malware"` |
| `blockip:{ip}` | STRING | 5m | Blocked IP audit marker | `"abuse"` |
| `profile:{pubkey}` | HASH | 5m | Profile metadata | `{name, about, picture}` |
| `query:{hash}` | STRING | 30s | Query results | `[event1, event2]` |
| `event:{id}` | STRING | 10m | Event cache | `{json}` |
| `dedup:{id}` | STRING | 1h | Event deduplication | `"1"` |
| `nip05:doc` | STRING | 24h | Cached NIP-05 document (`names` + optional `relays`) | `{\"names\":{...}}` |
| `sub:filter:{hash}` | STRING | 5m | Filter hash | `"hash"` |
| `ws:last_seen:{ws_id}` | STRING | 2m | WebSocket heartbeat timestamp | `1710000000` |
| `query:meta:{hash}` | HASH | 30s | Query cache metadata | `{hits,last_access}` |
| `nip29:group:{relay}:{group_id}` | HASH | configurable | Cached group metadata/policy |
| `nip29:member:{relay}:{group_id}:{pubkey}` | STRING | configurable | Membership hit cache |
| `nip29:ban:{relay}:{group_id}:{pubkey}` | STRING | configurable | Ban hit cache |
| `nip29:invite:{relay}:{group_id}:{code}` | HASH | until expiry | Invite validation and redemption state |
| `nip29:timeline:{relay}:{group_id}` | LIST | short TTL | Rolling recent event ids/prefixes for `previous` checks |

### Redis Guidance for NIP-29

- Use Redis as a **hot-path accelerator**, not as the source of truth.
- PostgreSQL remains authoritative for group metadata, memberships, bans and invite audit.
- Membership and ban lookups are good cache targets because they are checked repeatedly on EVENT/REQ hot paths.
- Invite redemption should use an atomic Redis path only when it meaningfully reduces DB contention; otherwise prefer PostgreSQL row locking.
- Timeline references are a strong Redis fit because they are short-lived, bounded and frequently rewritten.

### Redis Guidance for NIP-86

- Use exact-key lookups only (`allow:{pubkey}`, `blockip:{ip}`, `banevent:{id}`); do not add wildcard invalidation or large set scans to the request path.
- Invalidate targeted keys on every admin mutation instead of global cache flushes.
- Reuse existing pub/sub only when a cross-instance disconnect fan-out is needed for `blockip`; PostgreSQL remains the source of truth.
- Avoid storing large JSON-RPC responses in Redis; NIP-86 results are small and mutation-heavy.

### Pub/Sub Channels

| Channel | Purpose | Message Format |
|---------|---------|----------------|
| `events` | New event broadcast | `{"id":"...","pubkey":"...","kind":1,...}` |
| `sub:{subscription_id}` | Subscription match | `{"event":{...},"ws_id":"..."}` |
| `ws:connect` | WebSocket connect | `{"ws_id":"...","pubkey":"..."}` |
| `ws:disconnect` | WebSocket disconnect | `{"ws_id":"..."}` |
| `sub:create` | New subscription | `{"ws_id":"...","sub_id":"...","filter":{}}` |
| `sub:close` | Close subscription | `{"ws_id":"...","sub_id":"..."}` |
| `sub:cleanup` | Orphan subscription cleanup | `{"ws_id":"..."}` |

### Subscription Storage

Key: `subs:{ws_id}`  
Type: HASH

| Field | Value |
|-------|-------|
| `{sub_id}` | `{"filter":{...},"created_at":1234567890}` |
| `info` | `{"ip":"...","authed":"..."}` |

### Orphan Cleanup Strategy

- every websocket updates `ws:last_seen:{ws_id}` with short TTL
- `subs:{ws_id}` is removed on normal disconnect
- a periodic cleanup job removes `subs:{ws_id}` when `ws:last_seen:{ws_id}` no longer exists
- `sub:cleanup` pub/sub messages notify other instances to evict stale mirrors

### Planned Queue Keys

The operational queue refactor uses compact Redis structures. PostgreSQL is not introduced as the primary job store in the first iteration.

| Key Pattern | Type | TTL | Purpose |
|---|---|---|---|
| `rq:{queue}:seq` | STRING | none | Sequential numeric job id allocator |
| `rq:{queue}:stream:{priority}` | STREAM | bounded by `MAXLEN ~` | Main delivery channel for Consumer Groups |
| `rq:{queue}:delayed` | ZSET | none | Delayed and retry-ready jobs by `run_at_ms` |
| `rq:{queue}:dead` | ZSET | policy-driven | Dead-letter queue ordered by terminal timestamp |
| `rq:{queue}:jobs` | ZSET | none | Recent job ordering index by creation time |
| `rq:{queue}:body:{job_id}` | STRING | configurable | Serialized job payload/envelope |
| `rq:{queue}:result:{job_id}` | STRING | configurable | Optional structured terminal result payload |
| `rq:{queue}:state` | STRING (BITFIELD) | none | 3-bit compact lifecycle state per job id |
| `rq:{queue}:attempts` | STRING (BITFIELD) | none | 8-bit attempt counter per job id |
| `rq:{queue}:meta:{job_id}` | HASH | configurable | Small metadata only (`j`, `q`, `p`, `a`, `ma`, `ca`, `la`, `sa`, `fa`, `ra`, `e`) |
| `rq:{queue}:metrics:{yyyyMMddHHmm}` | HASH | short TTL | Per-minute aggregate counters |
| `rq:{queue}:workers:{yyyyMMddHHmm}` | HLL | short TTL | Approximate unique workers per window |
| `rq:{queue}:unique:{fingerprint}` | STRING | option-driven | Idempotency / uniqueness guard |

### Planned Queue Metadata Fields

| Field | Meaning |
|---|---|
| `j` | job name |
| `q` | queue name |
| `p` | priority |
| `a` | current attempts |
| `ma` | max attempts |
| `ca` | created at unix ms |
| `la` | last attempt unix ms |
| `sa` | started at unix ms |
| `fa` | finished at unix ms |
| `ra` | delayed/retry release unix ms |
| `e` | summarized last error |

### Planned Queue Status Encoding

Stored in `rq:{queue}:state` with `BITFIELD` using 3 bits per sequential id.

| Value | Meaning |
|---:|---|
| 0 | unknown |
| 1 | queued |
| 2 | running |
| 3 | succeeded |
| 4 | failed |
| 5 | delayed |
| 6 | dead |
| 7 | canceled |

### Planned Queue Operational Notes

- Streams are the source of delivery truth.
- ZSET is used for delayed and retry scheduling only.
- payload stays in STRING to avoid large Hash memory overhead.
- per-job Hash data must remain compact; no large JSON blobs in Hash fields.
- all queue keys for the same logical queue share one Redis hash tag (`{queue}`) to keep Lua multi-key operations cluster-safe.

### Sync Job Payload and Result Shape

For `sync.negentropy`, the Redis `body` and `result` blobs are the audit surface consumed by `/panel/sync`.

Recommended payload subset:

```json
{
  "remote": "wss://relay.example.com",
  "direction": "up",
  "public_key": "",
  "filter_json": "[{\"kinds\":[1],\"authors\":[\"..."]}]",
  "timeout_seconds": 30
}
```

Recommended result subset:

```json
{
  "remote": "wss://relay.example.com",
  "direction": "up",
  "status": "failed",
  "filter": [{"kinds":[1],"authors":["..."]}],
  "error": "remote relay rejected 2 event(s)",
  "rejections": [
    {
      "event_id": "d9b708...",
      "reason": "blocked: please use a dedicated relay for moderated communities",
      "raw": "[\"OK\",\"d9b708...\",false,\"blocked: please use a dedicated relay for moderated communities\"]"
    }
  ]
}
```

Notes:

- `filter_json` keeps the original serialized request for requeue compatibility.
- `result.filter` stores the normalized parsed filter used at runtime so the UI does not need to reverse-engineer it.
- `result.rejections` is bounded, human-readable diagnostic data; `meta.e` keeps only the compact `last_error` summary.

### Configuration

Redis connection via `conf.yaml`:

```yaml
redis:
  enabled: true
  addr: "127.0.0.1:6379"
  password: ""
  db: 0
  pool_size: 10
  subscription_cleanup_interval_seconds: 60
  subscription_stale_after_seconds: 120
  cache:
    ban_ttl: 1h
    profile_ttl: 5m
    query_ttl: 30s
    event_ttl: 10m
    dedup_ttl: 1h
    query_meta_ttl: 30s
    nip05_doc_ttl: 24h
```

### Prepared Statements

For optimized query execution:

```sql
-- Event by ID (covering index)
PREPARE event_by_id AS 
  SELECT id, pubkey, created_at, kind, tags, content, sig 
  FROM event WHERE id = $1;

-- Events by author with limit
PREPARE events_by_author_kind AS 
  SELECT id, pubkey, created_at, kind, tags, content, sig 
  FROM event 
  WHERE pubkey = $1 AND kind = $2 
  ORDER BY created_at DESC LIMIT $3;

-- Recent events by kind
PREPARE recent_events_by_kind AS 
  SELECT id, pubkey, created_at, kind, tags, content, sig 
  FROM event 
  WHERE kind = $1 AND created_at > $2 
  ORDER BY created_at DESC LIMIT $3;
```

### Query Optimization Indexes

Additional indexes for performance:

```sql
-- Partial index for recent events (use a fixed timestamp in migrations)
CREATE INDEX idx_event_recent ON event (created_at DESC, id)
WHERE created_at > 1710000000;

-- Partial index for popular kinds
CREATE INDEX idx_event_kinds_popular ON event (kind, created_at DESC) 
WHERE kind IN (0, 1, 3, 7);

-- Safe author-time index for hot author queries
CREATE INDEX idx_event_pubkey_created_at ON event (pubkey, created_at DESC);

-- Composite index for author + kind filters
CREATE INDEX idx_event_author_kind ON event (pubkey, kind, created_at DESC);

-- Partial index for deletion events
CREATE INDEX idx_event_deletions ON event (created_at DESC) 
WHERE deleted_by IS NOT NULL;
```

Avoid these index shapes on `event`:

```sql
CREATE INDEX ... INCLUDE (content, tags, sig);
CREATE INDEX ... ON event ((tags::text));
CREATE INDEX ... ON event ((content));
```

They increase write amplification and can fail for large events.

Prefer exact tag filters such as:

```sql
tags @> '[["p","<pubkey>"]]'::jsonb
tags @> '[["e","<event_id>"]]'::jsonb
tags @> '[["t","nostr"]]'::jsonb
```

## Planned Blossom Admin Extensions

### 14. `blossom_objects_admin` - Enriched Blossom Object Metadata

Extends the existing `objects` row without changing the public hash identity model.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `hash` | VARCHAR(64) | PK, FK(`objects.hash`) | Canonical object hash |
| `extension` | VARCHAR(32) | NULL | Derived extension used by filters/UI |
| `width` | INTEGER | NULL | Media width in pixels |
| `height` | INTEGER | NULL | Media height in pixels |
| `duration_ms` | BIGINT | NULL | Audio/video duration |
| `bitrate_kbps` | INTEGER | NULL | Encoded bitrate |
| `blurhash` | TEXT | NULL | Progressive placeholder |
| `thumbnail_hash` | VARCHAR(64) | NULL | Thumbnail object hash when stored as a derivative |
| `optimized_hash` | VARCHAR(64) | NULL | Primary optimized derivative hash |
| `hls_manifest_hash` | VARCHAR(64) | NULL | Optional HLS/DASH manifest hash |
| `processing_status` | VARCHAR(24) | NOT NULL DEFAULT 'pending' | `pending`, `processing`, `ready`, `failed` |
| `processing_error` | TEXT | NULL | Last optimization failure summary |
| `exif_status` | VARCHAR(24) | NOT NULL DEFAULT 'pending' | `pending`, `clean`, `stripped`, `rejected` |
| `gps_detected` | BOOLEAN | NOT NULL DEFAULT FALSE | Sensitive GPS metadata detected |
| `last_downloaded_at` | TIMESTAMPTZ | NULL | Idle cleanup reference |
| `download_count` | BIGINT | NOT NULL DEFAULT 0 | Access volume |
| `ingress_bytes` | BIGINT | NOT NULL DEFAULT 0 | Original upload traffic |
| `egress_bytes` | BIGINT | NOT NULL DEFAULT 0 | Download traffic attributable to this object |
| `review_state` | VARCHAR(24) | NOT NULL DEFAULT 'ready' | `ready`, `flagged`, `pending_review`, `approved`, `deleted` |
| `flag_reason` | TEXT | NULL | Operator/AI moderation note |
| `nip94_tags` | JSONB | NOT NULL DEFAULT '[]'::jsonb | Ordered NIP-94 tag tuples generated from extracted media facts |
| `mirrors` | JSONB | NOT NULL DEFAULT '[]'::jsonb | Deduplicated mirror/source URLs used to emit NIP-94 `fallback` tags |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Latest enrichment timestamp |

Rules for `nip94_tags`:

- persisted shape is a JSON array of arrays, e.g. `[["url","..."],["m","image/png"],["x","..."]]`
- this field is fully regenerated by the BUD-05 media extraction pipeline after upload or mirror completion
- the source fields for generation are the canonical object row (`objects`) plus extracted admin metadata (`width`, `height`, `duration_ms`, `bitrate_kbps`, `blurhash`, `thumbnail_hash`, `optimized_hash`, `mirrors`)
- `mirrors` must preserve the original BUD-04 source URL when a mirror job succeeds
- `processing_status` reflects the last known BUD-05 worker state for `HEAD /media` probes and dashboard inspection

### 15. `blossom_review_reports` - Review Queue Inputs

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Review signal id |
| `object_hash` | VARCHAR(64) | NOT NULL, FK(`objects.hash`) | Target object |
| `source` | VARCHAR(24) | NOT NULL | `nip56`, `ai`, `system`, `operator` |
| `reporter_pubkey` | VARCHAR(64) | NULL | Nostr reporter when available |
| `report_type` | VARCHAR(64) | NULL | NIP-56 or AI category |
| `reason` | TEXT | NULL | Human-readable explanation |
| `payload` | JSONB | NOT NULL DEFAULT '{}'::jsonb | Raw report evidence |
| `status` | VARCHAR(24) | NOT NULL DEFAULT 'open' | `open`, `dismissed`, `actioned` |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Signal creation |
| `resolved_at` | TIMESTAMPTZ | NULL | Resolution timestamp |
| `resolved_by` | VARCHAR(64) | NULL | Operator pubkey/admin identity |

### 16. `blossom_pubkey_quotas` - Upload Authorization and Limits

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `pubkey` | VARCHAR(64) | PRIMARY KEY | Authorized uploader |
| `enabled` | BOOLEAN | NOT NULL DEFAULT TRUE | Hard allow/deny switch |
| `storage_quota_bytes` | BIGINT | NULL | Max persisted disk usage |
| `egress_quota_bytes` | BIGINT | NULL | Max monthly egress |
| `notes` | TEXT | NULL | Operational notes |
| `created_by` | VARCHAR(64) | NOT NULL | Admin identity |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Last quota change |

Effective quota resolution rules:

- in `free` mode, any uploader may upload and `storage_quota_bytes` / `egress_quota_bytes` override the global free-mode default plan when present
- in `enabled_users` mode, only rows with `enabled = TRUE` may upload, and non-null quota columns override the enabled-user default plan
- in `mandatory_review` mode, upload permission follows the same allowlist/free semantics selected by the server policy, but the object stays blocked until an explicit approval

### 17. `blossom_pubkey_usage` - Aggregated Usage by Uploader

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `pubkey` | VARCHAR(64) | PRIMARY KEY | Uploader pubkey |
| `object_count` | BIGINT | NOT NULL DEFAULT 0 | Number of live objects |
| `storage_used_bytes` | BIGINT | NOT NULL DEFAULT 0 | Current disk footprint |
| `monthly_ingress_bytes` | BIGINT | NOT NULL DEFAULT 0 | Current month ingress |
| `monthly_egress_bytes` | BIGINT | NOT NULL DEFAULT 0 | Current month egress |
| `last_upload_at` | TIMESTAMPTZ | NULL | Most recent upload |
| `last_reset_month` | DATE | NOT NULL | Quota reset anchor |

### 18. `blossom_server_policy` - Effective Upload Mode

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | SMALLINT | PRIMARY KEY, fixed value `1` | Singleton row |
| `mode` | VARCHAR(24) | NOT NULL | `mandatory_review`, `enabled_users`, `free` |
| `updated_by` | VARCHAR(64) | NOT NULL | Admin identity |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Last policy change |

### 19. `blossom_plans` - Named Quota Plan Catalog

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | VARCHAR(64) | PRIMARY KEY | Stable plan id used by admin UI |
| `name` | VARCHAR(120) | NOT NULL | Operator-facing plan label |
| `scope` | VARCHAR(24) | NOT NULL | `free` or `enabled_users` |
| `storage_quota_bytes` | BIGINT | NULL | Disk allowance; `NULL` means unlimited |
| `egress_quota_bytes` | BIGINT | NULL | Monthly egress allowance |
| `description` | TEXT | NULL | Explanatory copy shown in the plans screen |
| `is_default` | BOOLEAN | NOT NULL DEFAULT FALSE | Whether this plan is the default for its scope |
| `updated_by` | VARCHAR(64) | NOT NULL | Admin identity |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Last plan edit |

Rules:

- at most one `is_default = TRUE` plan should exist per scope
- per-pubkey quota overrides in `blossom_pubkey_quotas` or `blossom_plan_assignments` take precedence over plan defaults
- `blossom_server_policy` keeps the active upload mode, while `blossom_plans` provides reusable presets

### 20. `blossom_plan_assignments` - 1:1 Plan Assignment per User

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `pubkey` | VARCHAR(64) | PRIMARY KEY | Authorized uploader |
| `plan_id` | VARCHAR(64) | NOT NULL, FK(`blossom_plans.id`) | Assigned plan |
| `assigned_by` | VARCHAR(64) | NOT NULL | Admin identity |
| `assigned_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Assignment timestamp |

Rules:
- A user can only have one active plan at a time. This table handles 1:1 associations.
- The plan assignment determines the effective storage and egress quotas for the user.

### 21. Mirror and media job state

Mirror and derivative jobs do not introduce a dedicated PostgreSQL table in this phase.

- request payload, execution state, retry counters and final result live in the existing Redis-backed generic jobs runtime
- the canonical mirror payload is `{source_url, expected_sha256, requested_by}`
- workers expose `queued`, `running`, `succeeded`, `failed` and `canceled` states through the existing jobs monitor surface
- relational persistence only starts after success, when `objects` and `blossom_objects_admin` are updated and audit rows are appended

### 22. `blossom_audit_log` - Immutable Administrative Trail

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Audit id |
| `actor_pubkey` | VARCHAR(64) | NOT NULL | Acting admin/operator |
| `action` | VARCHAR(64) | NOT NULL | `mirror.create`, `object.delete`, `user.purge`, etc. |
| `target_type` | VARCHAR(32) | NOT NULL | `object`, `pubkey`, `quota`, `policy`, `job` |
| `target_id` | TEXT | NOT NULL | Hash, pubkey or job id |
| `request_id` | TEXT | NULL | Propagated `x-request-id` |
| `payload` | JSONB | NOT NULL DEFAULT '{}'::jsonb | Redacted action context |
| `nostr_event_id` | VARCHAR(64) | NULL | Matching `kind:24242` event when emitted |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Immutable write time |
