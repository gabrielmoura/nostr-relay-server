# Data Schema

## Overview

This document describes both PostgreSQL and Redis data structures:
- **PostgreSQL**: Persistent storage for events, profiles, and file metadata
- **Redis**: Cache, pub/sub, and subscription state management

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
- `arbitrarytagvalues` - GIN on `tagvalues`
- `content_search_idx` - GIN on `content_search`
- `idx_event_created_at_id` - B-tree on `(created_at, id)`

**Generated Columns:**
- `tagvalues`: Extracted via `tags_to_tagvalues()` function (single-character tags)
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

### Count Queries

Same filters as event queries but returns `COUNT(*)`.

---

## Redis Data Structures

### Cache Keys

| Key Pattern | Type | TTL | Purpose | Example |
|-------------|------|-----|---------|---------|
| `ban:{pubkey}` | STRING | 1h | Banned user reason | `"spam"` |
| `profile:{pubkey}` | HASH | 5m | Profile metadata | `{name, about, picture}` |
| `query:{hash}` | STRING | 30s | Query results | `[event1, event2]` |
| `event:{id}` | STRING | 10m | Event cache | `{json}` |
| `dedup:{id}` | STRING | 1h | Event deduplication | `"1"` |
| `sub:filter:{hash}` | STRING | 5m | Filter hash | `"hash"` |
| `ws:last_seen:{ws_id}` | STRING | 2m | WebSocket heartbeat timestamp | `1710000000` |
| `query:meta:{hash}` | HASH | 30s | Query cache metadata | `{hits,last_access}` |

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

-- Covering index for author queries
CREATE INDEX idx_event_author_covering ON event (pubkey, created_at DESC) 
INCLUDE (id, kind, tags, content, sig);

-- Composite index for author + kind filters
CREATE INDEX idx_event_author_kind ON event (pubkey, kind, created_at DESC);

-- Partial index for deletion events
CREATE INDEX idx_event_deletions ON event (created_at DESC) 
WHERE deleted_by IS NOT NULL;
```
