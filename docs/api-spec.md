# API Specification

## Overview

The Nostr Relay Server exposes two HTTP servers:

1. **External (Port)**: Public relay interface for Nostr clients
2. **Internal (Port+1)**: Admin and metrics endpoints

If `admin_token` is configured, all `/admin/*` endpoints require header `X-Admin-Token: <token>`.

### Admin GraphQL Transport

The internal admin API is exposed only through the authenticated GraphQL surface backed by `gqlgen` and served by Fiber.

- Proposed endpoint: `POST /admin/graphql`
- Proposed development endpoint: `GET /admin/graphql/schema`
- Proposed development endpoint: `GET /admin/graphql/playground`
- Proposed SDL source during docs phase: `docs/graphql-admin-schema.graphqls`
- Detailed migration spec: `docs/graphql-admin.md`

The previous `/admin/*` REST surface is no longer exposed by the internal server.

## Event Visualization Contracts

The admin dashboard event visualization work depends on the existing admin event payload already exposing enough raw protocol material to derive media and protocol-specific summaries in the frontend.

Frontend assumptions for `/panel/events/search` and `/panel/events/$eventId`:

- `event.kind`, `event.content`, `event.tags`, `event.id`, `event.pubkey`, `event.created_at` are always available
- `event.tags` may include `alt`, `imeta`, `e`, `a`, `p`, `k`, `relay`, `r`, `url`, `image`, `m`, `title`, `summary`, `d`
- `EventDetail.image_urls` is a backend-provided convenience list, but the frontend still treats `imeta` and textual URLs as canonical fallback sources

Interpretation rules documented from Nostr specifications used by the frontend:

- `kind:6` -> NIP-18 repost
- `kind:4550` -> NIP-72 community approval
- `kind:10050` -> NIP-51/NIP-17 DM relay list

No new backend endpoint is required for the first visual refinement. Optional future enrichment may add a backend helper for referenced-event snippets if the `@nostrify/*` read layer proves insufficient or too slow.

## WebSocket Protocol (NIP-01)

All Nostr communication happens over WebSocket using JSON messages.

### Message Types

#### Client → Relay

```json
["REQ", "<subscription_id>", <filter>]
["REQ", "<subscription_id>", <filter1>, <filter2>]
["EVENT", <event>]
["CLOSE", "<subscription_id>"]
["AUTH", <event>]
["COUNT", "<subscription_id>", <filter>]
```

#### Relay → Client

```json
["EVENT", "<subscription_id>", <event>]
["EOSE", "<subscription_id>"]
["OK", "<event_id>", <success>, <message>]
["NOTICE", "<message>"]
["CLOSED", "<subscription_id>", <reason>]
["AUTH", "<challenge>"]
```

### Filters

Filters define which events to subscribe to:

```json
{
  "ids": ["<event_id>"],
  "authors": ["<pubkey>", "<pubkey>"],
  "kinds": [0, 1, 2],
  "#e": ["<event_id>"],
  "#p": ["<pubkey>"],
  "since": 1690000000,
  "until": 1690100000,
  "limit": 100
}
```

### NIP-29 Validation Scope

- `EVENT`: when `nip29.enabled=true`, relay-side group validation only applies to explicit NIP-29 kinds (`9000`-`9022`, `39000`-`39003`). Events outside that scope are accepted or rejected only by the generic relay/security policies.
- `REQ` / `COUNT`: filters using `#h` are treated as group-scoped reads and must pass NIP-29 permission checks before matching events are delivered.
- `REQ` / `COUNT` without `#h` continue to use the normal relay pipeline. If matching results later contain private or hidden NIP-29 material, per-event delivery filtering still applies before the event is sent to the client.

### Marmot MIP-00 Event Scope

- When `marmot.mip00.enabled=true`, relay-side Marmot validation applies only to configured Marmot event kinds:
  - `kind:30443` KeyPackage events
  - `kind:10051` KeyPackage relay-list events
  - optional legacy `kind:443` only when explicitly enabled
- `REQ` and `COUNT` use the normal relay query path; no dedicated Marmot transport or admin endpoint is introduced.
- Phase 1 validation is tag- and shape-based only. The relay does not parse MLS payloads or verify cryptographic KeyPackage internals yet.
- Expected query patterns remain standard Nostr filters, especially by `authors`, `#d` and `#i` for `kind:30443`.

Marmot-related kinds outside this scope currently behave as generic Nostr events only:

- `444` Welcome
- `445` group events
- `447` / `448` / `449` push-token exchange kinds
- `10050` notification relay list

That means the relay can still persist and query them through the normal event pipeline, but no dedicated Marmot/MIP-05 validation or workflow contract exists for them in this phase.

## External Server Routes (Port)

### WebSocket Root `/`

**Upgrade:** WebSocket connection for Nostr protocol

**NIP-11:** Returns relay information when `Accept: application/nostr+json`

**NIP-86:** Accepts JSON-RPC over HTTP on the same `/` route when `Content-Type: application/nostr+json+rpc`

#### Request
```http
GET / HTTP/1.1
Upgrade: websocket
Connection: Upgrade
Accept: application/nostr+json
```

#### Response (NIP-11)
```json
{
  "name": "Nostr Relay Server",
  "description": "A Nostr Relay Server",
  "pub_key": "7ef721e77149c737...",
  "supported_nips": [1, 2, 4, 9, 11, 17, 25, 42, 45],
  "software": "https://github.com/gabrielmoura/nostr-relay-server",
  "version": "0.1.0",
  "limitation": {
    "max_message_length": 1048576,
    "max_subscriptions": 20,
    "max_filters": 100,
    "max_limit": 5000,
    "auth_required": false
  }
}
```

#### Request (NIP-86 JSON-RPC)

```http
POST / HTTP/1.1
Content-Type: application/nostr+json+rpc
Authorization: Nostr <base64_kind_27235_event>
```

```json
{
  "method": "banpubkey",
  "params": ["<hex-pubkey>", "spam"]
}
```

#### Authentication Rules for NIP-86

- `Authorization` is mandatory.
- Authorization event must be a valid NIP-98 `kind:27235` event.
- Required checks:
  - `kind == 27235`
  - valid signature
  - `created_at` inside a short freshness window
  - `method` tag equals the HTTP method
  - `u` tag equals the absolute request URL
  - `payload` tag equals the SHA-256 hex of the raw JSON-RPC body
- Only the configured relay administrator pubkey may execute methods.
- Failures return `401 Unauthorized`.

Operational note:

- this public API is optional and disabled by default
- the embedded dashboard should keep using the internal `/admin/*` API instead of signing NIP-98 requests in the browser

#### NIP-86 JSON-RPC Response

```json
{
  "result": true,
  "error": ""
}
```

#### Planned Supported NIP-86 Methods

- `supportedmethods`
- `banpubkey`
- `unbanpubkey`
- `listbannedpubkeys`
- `allowpubkey`
- `unallowpubkey`
- `listallowedpubkeys`
- `allowevent`
- `banevent`
- `listbannedevents`
- `changerelayname`
- `changerelaydescription`
- `blockip`
- `unblockip`
- `listblockedips`

#### NIP-86 Error Mapping

- `400` - malformed JSON-RPC payload, unsupported content type, invalid params
- `401` - missing/invalid NIP-98 auth or caller is not the configured admin pubkey
- `405` - HTTP method not allowed
- `500` - internal execution error

### Static Files

| Path | Description |
|------|-------------|
| `/nostr.png` | Relay icon |

### Well-Known Routes

#### `/.well-known/nostr/nip96.json`

Blossom server configuration (NIP-96).

**Response:**
```json
{
  "api_url": "http://localhost:9090/upload",
  "download_url": "http://localhost:9090/blob",
  "supported_nips": [1, 4, 5, 78, 94, 96, 98],
  "content_types": ["image/jpeg", "image/png", "video/mp4"],
  "tos_url": "http://localhost:9090/terms-of-service"
}
```

#### `/.well-known/nostr.json`

NIP-05 and media configuration.

**Query Parameters:**
- `?name=<username>` - Lookup NIP-05 user

**Response:**
```json
{
  "names": {
    "user": "b0635d6a9851d3aed0cd6c495b282167acf761729078d975fc341b22650b07b9"
  },
  "relays": {
    "b0635d6a9851d3aed0cd6c495b282167acf761729078d975fc341b22650b07b9": [
      "wss://relay.example.com",
      "wss://relay2.example.com"
    ]
  },
  "media": {
    "api_path": "http://localhost:9090/upload",
    "media_path": "http://localhost:9090/blob",
    "accepted_mimetypes": ["image/jpeg", "image/png"],
    "content_policy": {
      "allow_adult_content": false,
      "allow_violent_content": false
    }
  }
}
```

### Blossom Upload/Download (NIP-96)

#### `POST /upload`

Upload a file to Blossom storage.

**Headers:**
```
Authorization: Nostr <base64_event>
Content-Type: <mime_type>
```

**Body:** Binary file content

**Response (200):**
```json
{
  "hash": "sha256_hash_of_file",
  "url": "http://localhost:9090/blob/sha256_hash",
  "mime_type": "image/png",
  "created_at": 1690000000
}
```

**Errors:**
- `400` - Invalid file type or empty body
- `401` - Authentication failed or hash mismatch
- `500` - Server error

#### `GET /blob/:id`

Download a file by hash.

**Response:** Binary file with `Content-Type` header

#### `HEAD /blob/:id`

Check if file exists.

**Response Headers:**
- `200 OK` if exists
- `404 Not Found` if not

#### `GET /list/:id`

Get file metadata.

**Response:**
```json
{
  "hash": "sha256_hash",
  "link": "http://localhost:9090/blob/sha256_hash",
  "mime_type": "image/png",
  "created_at": 1690000000
}
```

### Redirect Routes

| Path | Redirects To |
|------|--------------|
| `/terms-of-service` | `{api_path}/terms-of-service` |

## Internal Server Routes (Port+1)

### `GET /metrics`

Prometheus metrics endpoint.

**Response:** Prometheus text format

### `GET /admin`

Admin interface placeholder.

**Response:** `"Admin Interface"`

### `GET /panel`

Serves the built React admin SPA from `infra/dash/dist` on the internal server.

The production binary embeds the generated `infra/dash/dist` assets using Go `embed`.

### `GET /panel/*`

SPA fallback route for client-side navigation. Static assets are served from `/panel/assets/*`.

Admin panel localization requirements:

- UI language can be detected from browser settings and persisted in local storage.
- Querystring override is supported for troubleshooting and direct linking (`?lang=en` or `?lang=pt-BR`).
- Current target locales are English and Portuguese (Brazil).

### `GET /admin/connections/active`

Returns active WebSocket connections for the current relay instance.

**Query Parameters:**
- `limit=<n>` - optional window size for virtual scrolling consumers
- `offset=<n>` - optional zero-based offset for incremental loading

**Response:**
```json
{
  "items": [
    {
      "ws_id": "ws_200.188.10.11_1774345260",
      "ip": "200.188.10.11",
      "authed": "<pubkey>",
      "subscription_count": 4,
      "connected_at": "2026-03-24T09:41:00Z",
      "last_seen_at": "2026-03-24T09:44:00Z",
      "user_agent": "WebSocket/Chrome"
    }
  ],
  "total": 1284,
  "limit": 100,
  "offset": 0,
  "has_more": true
}
```

### `GET /admin/connections/authed`

Returns authenticated WebSocket connections for the current relay instance.

Supports the same `limit` and `offset` parameters and response envelope used by `GET /admin/connections/active`.

### `POST /admin/connections/:wsid/disconnect`

Terminates a single active WebSocket connection by its administrative identifier.

**Body:**
```json
{
  "reason": "manual moderation"
}
```

**Response:**
```json
{
  "ws_id": "ws_200.188.10.11_1774345260",
  "disconnected": true
}
```

### `GET /admin/overview`

Returns the dashboard KPIs required by the admin SPA.

**Response:**
```json
{
  "active_connections": 1284,
  "authed_connections": 742,
  "logged_users": 388,
  "banned_users": 37,
  "indexed_events": 42300000,
  "events_per_minute": 2418,
  "relay_status": "operational"
}
```

### `GET /admin/users/logged`

Returns authenticated users grouped by pubkey, with profile metadata and connection counters.

**Query Parameters:**
- `limit=<n>` - optional window size for virtual scrolling
- `offset=<n>` - optional zero-based offset for incremental loading

**Response:**
```json
{
  "items": [
    {
      "pubkey": "<pubkey>",
      "npub": "npub1...",
      "display_name": "Relay Ops BR",
      "handle": "relayops_br",
      "picture": "https://...",
      "nip05": "relayops@nostr.br",
      "connection_count": 2,
      "last_seen_at": "2026-03-24T09:44:00Z",
      "connection_state": "stable"
    }
  ],
  "total": 388,
  "limit": 100,
  "offset": 0,
  "has_more": true
}
```

### `GET /admin/users/banned`

Returns banned users joined with their profile metadata.

**Query Parameters:**
- `q=<text>` - optional search on public key, name, display name or nip05
- `limit=<n>` - optional window size for virtual scrolling
- `offset=<n>` - optional zero-based offset for incremental loading

### `GET /admin/users/search`

Searches relay profiles for the admin panel.

**Query Parameters:**
- `q=<text>` - search over public key, `npub`, name, display name and nip05
- `limit=<n>` - optional window size for virtual scrolling
- `offset=<n>` - optional zero-based offset for incremental loading

### `GET /admin/users/:pubkey/profile`

Returns the best-known relay profile plus moderation metadata for a single user.

### `GET /admin/users/:pubkey/nip05`

Returns manual NIP-05 association for a specific user.

**Response (when found):**
```json
{
  "pubkey": "<hex_pubkey>",
  "exists": true,
  "name": "alice",
  "display_name": "Alice",
  "relay_hints": ["wss://relay.damus.io"],
  "created_at": "2026-04-17T12:00:00Z",
  "updated_at": "2026-04-17T12:15:00Z"
}
```

**Response (when missing):**
```json
{
  "pubkey": "<hex_pubkey>",
  "exists": false
}
```

### `GET /admin/nip05`

Lists manual NIP-05 associations used by `/.well-known/nostr.json`.

**Query Parameters:**
- `q=<text>` - optional search by name/pubkey/profile
- `limit=<n>`
- `offset=<n>`

### `POST /admin/nip05`

Creates or updates a manual NIP-05 association.

**Body:**
```json
{
  "name": "alice",
  "pubkey": "<hex_pubkey_or_npub>"
}
```

### `DELETE /admin/nip05/:name`

Deletes a manual NIP-05 association by `name`.

### `GET /admin/users/:pubkey/ban`

Returns whether a user is banned and the stored reason.

### `POST /admin/users/:pubkey/ban`

Bans a user by public key.

**Body:**
```json
{
  "reason": "spam",
  "related_ids": ["event-id-1"]
}
```

### `DELETE /admin/users/:pubkey/ban`

Removes a user ban by public key.

### `GET /admin/events/search`

Searches stored events by tags and/or full-text search.

Behavior notes:

- identifier-oriented fields should accept both canonical hex and applicable NIP-19 forms in the dashboard query builder
- full-text search must cover `content` and semantic tag text used by rich events, including community `description` tags on kind `34550`
- when listing kind `34550`, the admin response should keep enough tag data for the frontend to highlight `d`, `description` and `image`
- when Redis is enabled, the endpoint uses read-through response caching keyed by the normalized filter, `limit`, and `offset`
- the dedicated `cron` process proactively warms the default first page payload for `limit=50` and `offset=0`

**Query Parameters:**
- `q=<text>` - full-text search query
- `tag=p:value` - tag filter, repeatable
- `author=<pubkey>` - author filter, repeatable
- `kind=<kind>` - kind filter, repeatable
- `limit=<n>` - result limit
- `offset=<n>` - zero-based offset used by virtual scrolling/infinite loading

**Response:**
```json
{
  "items": [
    {
      "id": "6f9c...d12a",
      "pubkey": "<pubkey>",
      "created_at": 1774345260,
      "kind": 1,
      "tags": [["t", "relay"]],
      "content": "Atualizacao do cluster relay",
      "sig": "..."
    }
  ],
  "total": 1284,
  "limit": 100,
  "offset": 0,
  "has_more": true
}
```

### `POST /admin/events/import`

Imports one or many JSONL files with Nostr events using multipart form-data.

**Multipart fields:**
- `files` (repeatable) or `file`

**Response:**
```json
{
  "files": [
    {
      "filename": "events-2026-03-26.jsonl",
      "total": 240,
      "inserted": 230,
      "duplicates": 9,
      "invalid": 1,
      "error": ""
    }
  ]
}
```

### `GET /admin/stream/status`

Returns the current stream forwarding runtime state for admin observability.

**Response:**
```json
{
  "config": {
    "stream_up": true,
    "stream_down": false,
    "relays": ["wss://relay.damus.io", "wss://nos.lol"]
  },
  "dispatcher": {
    "started": true,
    "worker_count": 2,
    "event_queue_len": 0,
    "event_queue_cap": 1024,
    "request_queue_len": 1,
    "request_queue_cap": 256,
    "dropped_event_jobs": 0,
    "dropped_request_jobs": 0
  },
  "pool": {
    "initialized": true,
    "connected_relays": 2,
    "total_relays": 3,
    "relays": [
      {"url": "wss://relay.damus.io", "connected": true, "failure_count": 0},
      {"url": "wss://relay.nostr.band", "connected": false, "failure_count": 4, "last_error": "dial timeout"}
    ]
  },
  "counters": {
    "forwarded_events": 348,
    "forwarded_requests": 112,
    "forward_failures": 5
  }
}
```

### `GET /admin/events/search/aggregates`

Returns aggregation metrics for current event search filters.

**Query Parameters:** same as `GET /admin/events/search`

Behavior notes:

- aggregates are computed server-side from SQL-first grouped queries instead of loading all matched events into application memory
- when Redis is enabled, the endpoint uses read-through response caching keyed by the normalized filter
- the dedicated `cron` process proactively warms the default empty-filter aggregate payload

**Response:**
```json
{
  "total": 1284,
  "kinds": [{"kind": 1, "count": 830}],
  "top_authors": [{"pubkey": "<pubkey>", "count": 120}],
  "top_tags": [{"tag": "relay", "count": 410}]
}
```

### `GET /admin/events/search/timeline`

Returns timeline buckets for current event search filters.

**Query Parameters:** same as `GET /admin/events/search`, plus:
- `bucket=<hour|day>` - timeline granularity (default: `hour`)

Behavior notes:

- timeline buckets are computed server-side from SQL-first grouped queries instead of iterating over all matched events in Go
- when Redis is enabled, the endpoint uses read-through response caching keyed by the normalized filter plus `bucket`
- the dedicated `cron` process proactively warms the default empty-filter timeline payloads for both `bucket=day` and `bucket=hour`

**Response:**
```json
{
  "bucket": "hour",
  "points": [
    {"ts": 1774382400, "count": 43},
    {"ts": 1774386000, "count": 58}
  ]
}
```

### `GET /admin/events/:id`

Returns an enriched event detail payload for inspection and moderation.

**Response:**
```json
{
  "event": {"id": "...", "pubkey": "...", "kind": 1, "tags": [], "content": "..."},
  "identifiers": {
    "note": "note1...",
    "nevent": "nevent1...",
    "npub": "npub1...",
    "nprofile": "nprofile1..."
  },
  "author": {
    "pubkey": "...",
    "display_name": "Alice",
    "picture": "https://..."
  },
  "hashtags": ["relay", "nostr"],
  "image_urls": ["https://.../image.png"]
}
```

### `GET /admin/events/:id/reports`

Returns NIP-56 report events tied to an event id.

**Response:**
```json
{
  "items": [
    {
      "report_event_id": "...",
      "reporter_pubkey": "...",
      "reporter_npub": "npub1...",
      "reporter_display_name": "Mod 1",
      "reporter_picture": "https://...",
      "reported_event_id": "...",
      "reported_pubkey": "...",
      "report_type": "spam",
      "content": "spam campaign",
      "created_at": 1774386541
    }
  ],
  "total": 12
}
```

### `POST /admin/events/:id/fetch`

Fetches a missing event from external relays and persists it locally.

**Request Body:**
```json
{
  "relays": [
    "wss://relay.damus.io",
    "wss://relay.primal.net"
  ]
}
```

If `relays` is omitted, the server also tries configured stream relays plus a safe default relay set.

**Response:**
```json
{
  "event_id": "...",
  "source_relay": "wss://relay.damus.io",
  "persisted": true,
  "relays_tried": 6,
  "relay_results": [
    {"relay": "wss://relay.damus.io", "status": "found"},
    {"relay": "wss://relay.primal.net", "status": "not_found"},
    {"relay": "wss://bad-relay.example", "status": "connect_error", "error": "dial timeout"}
  ]
}
```

### `GET /admin/events/reported`

Lists reported target events (NIP-56 kind `1984`) with moderation-friendly metadata.

**Query Parameters:**
- `q=<text>` - search by target event id, target pubkey (hex or npub), or report content
- `type=<spam|nudity|malware|profanity|illegal|impersonation|other>` - optional report type filter
- `limit=<n>`
- `offset=<n>`

**Response (`items[]` excerpt):**
```json
{
  "target_event_id": "...",
  "target_pubkey": "...",
  "target_nevent": "nevent1...",
  "target_created_at": 1774380000,
  "target_created_at_iso": "2026-03-25T12:00:00Z",
  "target_author": {
    "pubkey": "...",
    "display_name": "Alice",
    "picture": "https://...",
    "nip05": "alice@example.com"
  },
  "report_count": 5,
  "last_reported": 1774386541,
  "last_reported_at": "2026-03-25T13:49:01Z",
  "report_types": ["spam", "malware"]
}

### `GET /admin/events/reported/summary`

Returns global moderation analytics for NIP-56 reports, computed from the full filtered dataset on the server and **not** from the paginated list slice.

**Query Parameters:** same filtering parameters accepted by `GET /admin/events/reported`, except pagination.

**Response:**
```json
{
  "total_events": 120,
  "total_reports": 364,
  "unique_target_authors": 42,
  "timeline": [
    {"bucket": "2026-05-01", "count": 18},
    {"bucket": "2026-05-02", "count": 27}
  ],
  "report_types": [
    {"name": "spam", "count": 140},
    {"name": "impersonation", "count": 58}
  ],
  "top_authors": [
    {"pubkey": "...", "display_name": "Alice", "count": 31}
  ],
  "top_targets": [
    {"target_event_id": "...", "count": 22}
  ]
}
```

Frontend rule:

- KPI and charts on `/panel/events/reported` must consume this summary endpoint when total-server analytics are required
- the virtualized list remains a separate paginated drill-down surface

### `GET /admin/labels`

Returns stored NIP-32 label events (`kind:1985`) for the admin dashboard.

**Query Parameters:**
- `namespace=<text>` - optional exact `L` namespace filter
- `label=<text>` - optional exact `l` value filter, repeatable for OR-style multi-label filtering
- `target_type=<event|pubkey|address|reference|topic>` - optional target type filter
- `target=<text>` - optional exact target value filter
- `author=<hex_pubkey>` - optional label author filter
- `q=<text>` - optional fuzzy search over target value and `content`
- `limit=<n>`
- `offset=<n>`

**Response:**
```json
{
  "items": [
    {
      "id": "ac206e...",
      "pubkey": "<author_pubkey>",
      "created_at": 1777975125,
      "kind": 1985,
      "content": "Conta usada para flood promocional.",
      "namespace": "ugc",
      "labels": ["spam", "scam"],
      "target": {
        "type": "pubkey",
        "value": "<hex-pubkey>",
        "relay_hint": "wss://relay.example"
      },
      "tags": [["L", "ugc"], ["l", "spam", "ugc"], ["p", "<hex-pubkey>"]]
    }
  ],
  "total": 12,
  "limit": 50,
  "offset": 0,
  "has_more": false
}
```

### `GET /admin/labels/summary`

Returns aggregated moderation-friendly summaries for NIP-32 labels.

**Query Parameters:** same filtering parameters accepted by `GET /admin/labels`, except pagination.

**Response:**
```json
{
  "total_events": 12,
  "total_targets": 7,
  "namespaces": [
    {"namespace": "ugc", "count": 8},
    {"namespace": "content-warning", "count": 2}
  ],
  "labels": [
    {"label": "spam", "count": 5},
    {"label": "nsfw", "count": 2}
  ],
  "target_types": [
    {"target_type": "pubkey", "count": 4},
    {"target_type": "event", "count": 3}
  ]
}
```

### `POST /admin/labels`

Creates, signs and stores a NIP-32 label event on behalf of the relay admin surface.

**Body:**
```json
{
  "namespace": "ugc",
  "labels": ["spam", "scam"],
  "comment": "Conta usada para flood promocional.",
  "target": {
    "type": "pubkey",
    "value": "<hex-pubkey>",
    "relay_hint": "wss://relay.example"
  }
}
```

**Validation rules:**
- `namespace` is required
- at least one `labels[]` value is required
- `target.type` must be one of `event`, `pubkey`, `address`, `reference`, `topic`
- `target.value` is required; dashboard-facing UX may accept NIP-19 (`note`, `nevent`, `npub`, `nprofile`, `naddr`) and normalize before submission
- `relay_information.priv_key` must be configured so the relay can sign the event

Profile-labeling note:

- labeling a profile/identity is supported through `target.type = "pubkey"`
- the admin UX should accept `hex`, `npub` and `nprofile` for this target type

**Response:**
```json
{
  "event": {
    "id": "...",
    "pubkey": "<relay_pubkey>",
    "created_at": 1778000000,
    "kind": 1985,
    "content": "Conta usada para flood promocional.",
    "tags": [
      ["L", "ugc"],
      ["l", "spam", "ugc"],
      ["l", "scam", "ugc"],
      ["p", "<hex-pubkey>", "wss://relay.example"]
    ],
    "sig": "..."
  },
  "stored": true
}
```

### `POST /admin/sync`

Starts a background Negentropy synchronization job with a remote relay.

**Body:**
```json
{
  "relay": "wss://relay.damus.io",
  "filter": {
    "kinds": [0, 1],
    "since": 1714000000
  },
  "dry_run": false
}
```

**Response:**
```json
{
  "status": "started",
  "relay": "wss://relay.damus.io"
}
```

### `POST /admin/download`

Bulk download events from multiple relays based on a filter.

**Body:**
```json
{
  "relays": ["wss://relay.damus.io", "wss://nos.lol"],
  "filter": {
    "kinds": [1],
    "limit": 1000
  }
}
```

**Response:**
```json
{
  "status": "started",
  "relays": ["wss://relay.damus.io", "wss://nos.lol"],
  "job_id": "..."
}
```

### `GET /admin/nip29/groups`

Lists NIP-29 groups managed by this relay.

**Response:**
```json
{
  "groups": [
    {
      "id": "group1",
      "name": "Nostr Devs",
      "about": "Group for nostr developers",
      "picture": "https://...",
      "is_public": true,
      "is_open": true
    }
  ]
}
```

### `GET /admin/wot/summary`

Returns the current Web of Trust (WoT) network status.

**Response:**
```json
{
  "nodes": 4500,
  "edges": 120000,
  "trusted_pubkeys": ["<pubkey1>", "<pubkey2>"],
  "last_recomputed_at": "2026-04-27T12:00:00Z"
}
```

### `POST /admin/wot/recompute`

Manually triggers a full WoT graph recomputation.

**Response:**
```json
{
  "status": "triggered"
}
```

### `POST /admin/wot/trusted-pubkeys`

Updates the set of trusted root pubkeys for the WoT computation.

**Body:**
```json
{
  "pubkeys": ["<pubkey1>", "<pubkey2>"]
}
```

**Response:**
```json
{
  "success": true
}
```
```

### Planned Internal Admin Endpoints For NIP-86 Dashboard

These endpoints are backend conveniences for the internal dashboard. They should reuse the same persistence/runtime behavior as the NIP-86 service layer instead of duplicating business rules.

#### `GET /admin/nip86/allowed-pubkeys`

Returns allowed pubkeys with reason and audit timestamps.

#### `POST /admin/nip86/allowed-pubkeys`

Creates or updates an allowed pubkey entry.

#### `DELETE /admin/nip86/allowed-pubkeys/:pubkey`

Removes an allowlisted pubkey.

#### `GET /admin/nip86/blocked-ips`

Returns blocked IPs with reason and audit timestamps.

#### `POST /admin/nip86/blocked-ips`

Blocks an IP and triggers disconnect of active matching websocket sessions.

#### `DELETE /admin/nip86/blocked-ips/:ip`

Removes an IP block entry.

#### `GET /admin/nip86/banned-events`

Returns banned event ids with audit metadata.

#### `POST /admin/nip86/banned-events`

Creates or updates a banned event entry.

#### `DELETE /admin/nip86/banned-events/:id`

Removes a banned event entry.

#### `GET /admin/nip86/relay-metadata`

Returns current runtime relay metadata overrides.

#### `POST /admin/nip86/relay-metadata`

Updates runtime relay name and description overrides.

### Download Job Endpoints - Planned refinement

The dashboard download UX needs real job visibility instead of a fire-and-forget toast.

#### `POST /admin/events/download`

Starts a backend-managed download job.

Response shape should include:

```json
{
  "status": "started",
  "job_id": "dl_...",
  "relays": ["wss://relay.damus.io"],
  "message": "download process started in background"
}
```

#### `GET /admin/events/download/jobs`

Returns recent download jobs ordered by creation time.

#### `GET /admin/events/download/jobs/:jobId`

Returns one download job with:

- lifecycle status
- relay list
- normalized filter payload
- result counters
- terminal error, if any

### Queue-backed operational job refinement

The queue refactor keeps the current admin entry points but makes them durable and observable.

#### `POST /admin/sync/negentropy`

Current response compatibility is preserved, but queue-backed execution should add `job_id`:

```json
{
  "status": "started",
  "relay": "wss://relay.damus.io",
  "job_id": "1042"
}
```

Operational requirements for queued sync jobs:

- the persisted payload/result must include the normalized filter used by the job so `/sync` can display it later
- the sync detail response must expose the exact filter used plus structured rejection diagnostics when remote relays answer `OK ... false ...`
- each remote relay must have a strict but configurable cap for concurrent negentropy jobs targeting that same relay
- canceling a sync job must move it to a true terminal `canceled` state instead of allowing implicit auto-resume

#### `POST /admin/jobs/:jobId/resume`

Explicitly resumes a previously canceled operational job when the job type supports resume semantics.

Initial scope:

- `sync.negentropy`

#### `DELETE /admin/jobs`

Deletes job history from the backend for the specified operational slice instead of only hiding cards in the UI.

**Query Parameters:**
- `job_name=<name>` - required for bounded cleanup from dashboard screens (`download.events`, `sync.negentropy`)
- `status=<status>` - optional, repeatable; if omitted the backend may default to terminal states only

This endpoint is intended for history cleanup, not for deleting active/running jobs.

#### `GET /admin/jobs/:jobId`

Generic operational job detail endpoint planned for the dashboard and operator tooling.

Response shape:

```json
{
  "id": "1042",
  "queue": "admin",
  "job_name": "sync.negentropy",
  "status": "running",
  "attempts": 1,
  "max_attempts": 5,
  "created_at": "2026-04-28T12:00:00Z",
  "started_at": "2026-04-28T12:00:01Z",
  "finished_at": null,
  "run_at": null,
  "last_error": "",
  "payload": {
    "remote": "wss://relay.damus.io",
    "direction": "up",
    "filter_json": "[{\"kinds\":[1]}]"
  },
  "result": {
    "remote": "wss://relay.damus.io",
    "direction": "up",
    "status": "failed",
    "filter": [{"kinds":[1]}],
    "error": "remote relay rejected 2 event(s)",
    "rejections": [
      {
        "event_id": "d9b70849564dc07fe78d35ac38f372d59b6e14e983ff8a3b22a581bd07db5ce1",
        "reason": "blocked: please use a dedicated relay for moderated communities",
        "raw": "[\"OK\",\"d9b70849564dc07fe78d35ac38f372d59b6e14e983ff8a3b22a581bd07db5ce1\",false,\"blocked: please use a dedicated relay for moderated communities\"]"
      }
    ]
  }
}
```

Interpretation rules for `/panel/sync`:

- the modal must prefer `result.filter` over `payload.filter_json` when both exist
- the modal must prefer `result.rejections` and `result.error` over the compact `last_error`
- `last_error` still powers the compact job card and list-level warning badge

#### `POST /admin/jobs/:jobId/retry`

Requeues a failed or dead operational job.

#### `POST /admin/jobs/:jobId/cancel`

Marks a queued or delayed operational job as canceled when the handler semantics allow it.

## Negentropy Protocol (NIP-47)

Negentropy is used for efficient set reconciliation between relays, minimizing bandwidth when comparing large event sets.

This relay supports Negentropy sessions over WebSocket and keeps backward compatibility with peers that still use legacy `NEG-HAVE` / `NEG-NEED` payload exchanges.

### Session Messages

```json
["NEG-OPEN", "<subscription_id>", <filter>, "<initial_message_hex>"]
["NEG-MSG", "<subscription_id>", "<message_hex>"]
["NEG-ERR", "<subscription_id>", "<reason>"]
["NEG-CLOSE", "<subscription_id>"]
```

Notes:

- `subscription_id` follows the normal Nostr subscription semantics.
- `<filter>` uses the same filter semantics as `REQ`.
- `NEG-OPEN` and `NEG-MSG` message payloads are hex-encoded protocol frames.

### Data Transfer During Reconciliation

Depending on the peer implementation, missing events can be transferred in two compatible ways:

1. **Legacy relay-to-relay extension**

```json
["NEG-HAVE", "<subscription_id>", [<event>, ...]]
["NEG-NEED", "<subscription_id>", ["<event_id>", ...]]
```

2. **Strfry-compatible path (recommended for broad interoperability)**

- Upload side sends `EVENT` for ids the remote relay needs.
- Download side requests ids via batched `REQ` filters and waits for `EOSE`.

Example request batch:

```json
["REQ", "<request_sub_id>", {"ids": ["<id1>", "<id2>"]}]
```

The relay may answer with:

- `EVENT` envelopes (download payload)
- `EOSE` (batch completed)
- `CLOSED` (request rejected, often due to relay-side limits)

### Operational Behavior

- Negentropy handling is gated by `enable_negentropy` in `conf.yaml`.
- `negentropy_auth=true` additionally requires websocket NIP-42 authentication for all Negentropy session messages and authorizes only `relay_information.pub_key`.
- The server keeps per-session state and exposes dedicated Negentropy V2 Prometheus metrics.
- The sync CLI uses adaptive REQ batching to handle relays with strict `ids` limits.
- When a remote relay answers with `AUTH`, the sync CLI signs a NIP-42 event with `relay_information.priv_key` and retries the Negentropy flow on the same websocket.

## Supported NIPs

| NIP | Name | Status |
|-----|------|--------|
| 01 | Basic Protocol | ✅ |
| 02 | Follow List | ✅ |
| 04 | Encrypted Direct Messages | ✅ |
| 09 | Event Deletion | ✅ |
| 11 | Relay Information | ✅ |
| 17 | Relay List Metadata | ✅ |
| 18 | Public Chat | ✅ |
| 25 | Reactions | ✅ |
| 40 | Expiration Timestamp | ✅ |
| 42 | Authentication | ✅ |
| 45 | Event Counts | ✅ |
| 50 | Search | ✅ |
| 62 | Request to Vanish | ✅ |
| 77 | Kind 30078 | ✅ |
| 96 | Blossom Storage | ✅ |
| 98 | HTTP Auth | ✅ |
| 86 | Relay Management API | planned |

## Error Responses

### WebSocket Notice
```json
["NOTICE", "error message here"]
```

### Event Rejection (OK Envelope)
```json
["OK", "<event_id>", false, "reason: event blocked"]
```

### Subscription Closed
```json
["CLOSED", "<subscription_id>", "auth-required: REQ filters are not accepted"]
```

## Rate Limits

Configured in `conf.yaml`:
```yaml
ws:
  rate_limit: 1    # requests per second
  burst: 5         # max burst size
```

## Planned Internal Blossom Admin API

All routes below live on the internal admin server and inherit optional `X-Admin-Token` protection plus `x-request-id` propagation.

New follow-up scope for this module:

- uploader policy must support `mandatory_review`, `enabled_users`, and `free`
- uploader quotas must support global defaults per mode plus per-pubkey overrides
- object URLs should prefer the stored extension when known: `/blob/<sha256>.png`
- the dashboard gains dedicated reports and analytics surfaces plus a workers modal shortcut
- Blossom must expose a lower-level plans/quotas configuration screen for richer plan modeling
- Blossom HTTP traffic must publish Prometheus metrics for total requests, latency and categorized errors, including missing/invalid auth

### `GET /admin/blossom/overview`

Returns KPI cards and system alerts.

```json
{
  "storage": {"used_bytes": 912345678, "free_bytes": 123456789, "used_percent": 88.1},
  "objects": {"total": 48321, "flagged": 91, "pending_review": 14},
  "traffic": {"monthly_ingress_bytes": 99887766, "monthly_egress_bytes": 3344556677},
  "users": {"active": 421, "whitelisted": 97},
  "workers": {"running": 6, "failed": 2},
  "policy": {
    "mode": "enabled_users",
    "default_storage_quota_bytes": 5368709120,
    "default_egress_quota_bytes": null,
    "enabled_user_default_storage_quota_bytes": 5368709120,
    "enabled_user_default_egress_quota_bytes": null,
    "updated_at": "2026-05-15T11:22:33Z"
  },
  "alerts": [
    {"level": "warning", "code": "disk-usage", "message": "Disco 90% cheio"}
  ]
}
```

Policy semantics:

- `mandatory_review`: any authenticated uploader may upload, but every new blob starts in `pending_review` and remains unavailable for public download until approved
- `enabled_users`: only pubkeys explicitly enabled in Blossom policy may upload; quota defaults come from the enabled-user default plan unless overridden per pubkey
- `free`: any authenticated uploader may upload; quota defaults come from the free-mode default plan unless left `null` for unlimited

### `GET /admin/blossom/policy`

Returns the effective server-side Blossom policy used during upload authorization and review gating.

The plans screen also needs richer payload detail than the current top-line mode:

- active default plan per mode
- list of named plans with storage/egress limits and visibility metadata
- optional explanatory copy to drive tooltips and warnings in the admin UI

### `PUT /admin/blossom/policy`

Updates the effective Blossom upload mode.

```json
{
  "mode": "free"
}
```

### `GET /admin/blossom/plans`

Returns the catalog of named Blossom upload plans used by the dedicated plans/quota screen.

Suggested response shape:

```json
{
  "items": [
    {
      "id": "free-5gb",
      "name": "Free 5 GB",
      "scope": "free",
      "storage_quota_bytes": 5368709120,
      "egress_quota_bytes": null,
      "description": "Plano padrão para uploads abertos",
      "is_default": true,
      "updated_at": "2026-05-25T18:10:00Z"
    }
  ]
}
```

### `PUT /admin/blossom/plans`

Creates or updates one named Blossom plan.

Expected fields:

- `id`
- `name`
- `scope=free|enabled_users`
- `storage_quota_bytes`
- `egress_quota_bytes`
- `description`
- `is_default`

### `DELETE /admin/blossom/plans/:id`

Soft-removes or hard-removes one named plan when it is not the active default for the current mode.

### `POST /admin/blossom/plans/:id/assign`

Assigns a named plan to a specific uploader. Associations are 1:1, so assigning a new plan will replace any existing plan assignment for that user.

```json
{
  "pubkey": "<hex-pubkey>"
}
```

### `DELETE /admin/blossom/plans/:id/assign/:pubkey`

Removes a plan assignment from an uploader.

### `GET /admin/blossom/objects`

Paginated object browser with exact SHA-256 search and MIME/extension filters.

Query params:

- `limit`, `offset`
- `sha256` exact match
- `mime_type`
- `extension`
- `view=grid|table`
- `review_state`
- `pubkey`
- `uploader_q` matches uploader identity by name, display name, nip05, `npub`, or hex pubkey

### `GET /admin/blossom/objects/:hash`

Returns one object plus enriched metadata, derived files, NIP-94 tags, EXIF/privacy status and moderation state.

Response additions used by the frontend:

- `direct_url` prefers extension-bearing URLs when `extension` is known
- `blossom_id` exposes a BUD-10 URI such as `blossom:<sha256>.png?xs=cdn.example.com&as=<pubkey>&sz=<bytes>`
- `report_count` exposes the number of open BUD-09 reports tied to the blob

### `POST /admin/blossom/objects/bulk-review`

Bulk moderation endpoint for approval or hard-delete.

```json
{
  "hashes": ["<sha256>", "<sha256>"],
  "action": "approve",
  "reason": "false positive"
}
```

Allowed actions:

- `approve`
- `hard_delete`
- `requeue_optimization`

Approval side effect for `mandatory_review` mode:

- approving an object clears the public-download block added at upload time

### `POST /admin/blossom/users/whitelist`

Create or update pubkey whitelist/quota settings.

When the server is in `enabled_users` mode, `enabled=true` grants upload permission for that pubkey.

Per-user quotas override the effective default plan for the current mode when non-null.

### `GET /admin/blossom/users`

Paginated uploader list with quota usage. Includes profile metadata `display_name`, `picture`, and `npub` when available.

Query params:

- `q` matches hex pubkey, `npub`, profile name, display name, `nip05`, or admin notes
- `sort_by` column to sort by (e.g., `storage_used_bytes`, `monthly_egress_bytes`, `object_count`, `last_upload_at`, `pubkey`)
- `sort_dir` direction of sort (`asc` or `desc`)

### `GET /admin/blossom/users/:pubkey`

Returns uploader profile, all linked files, quota usage and purge risk summary.

### `POST /admin/blossom/users/:pubkey/purge`

Enqueues full destructive purge for one uploader.

### `POST /admin/blossom/mirror`

Internal dashboard mirror command for BUD-04.

```json
{
  "source_url": "https://origin.example.com/blob/<sha256>",
  "expected_sha256": "<sha256>"
}
```

The server enqueues a real background mirror job and returns its job id immediately. Worker snapshots and audit records become the monitoring surface for operators.

### `GET /admin/blossom/workers`

Lists optimization, mirror, purge and cleanup jobs with queue status.

Query params:

- `status`
- `job_type`
- `target_hash`

The dashboard uses this endpoint in two places:

- workers tab for long-running operational monitoring
- header-level workers modal for quick inspection from any Blossom tab

### `GET /admin/blossom/reports`

Paginated BUD-09 report browser for blobs.

Query params:

- `limit`, `offset`
- `q` matches hash, reporter, target event id, target pubkey, or reason text
- `report_type`
- `status=open|dismissed|actioned`
- `object_hash`

### `POST /admin/blossom/reports/:id/resolve`

Marks one stored blob report as `dismissed` or `actioned`, optionally attaching an operator note.

### `GET /admin/blossom/analytics`

Returns pre-aggregated operational analytics for the dashboard modal.

Suggested response blocks:

- storage distribution by MIME and review state
- upload and egress time series
- top uploaders by storage and report count
- worker counts by status and job type
- report totals by type and resolution state

### `GET /admin/blossom/audit`

Read-only audit timeline backed by relational log plus optional `kind:24242` linkage.

## Blossom Prometheus Metrics

The external Blossom HTTP surface should emit route-level Prometheus metrics.

Required counters/histograms:

- `nostr_blossom_http_requests_total{route,method}`
- `nostr_blossom_http_request_duration_seconds{route,method}`
- `nostr_blossom_http_errors_total{route,method,category,status}`

Suggested error categories:

- `auth_missing`
- `auth_invalid`
- `auth_signature_invalid`
- `quota_exceeded`
- `policy_denied`
- `mime_rejected`
- `hash_mismatch`
- `not_found`
- `range_invalid`
- `internal`

Coverage requirements:

- unauthenticated `PUT /upload`, `PUT /mirror`, `PUT /media`, and `PUT /report` must increment `nostr_blossom_http_errors_total` with an auth-related category
- successful `GET /blob/:id` and `HEAD /blob/:id` should count independently from upload routes
- route labels should use normalized logical routes rather than raw hashes in the URL

## Planned Blossom/BUD Public Endpoints

### `PUT /mirror` (BUD-04)

- public Blossom route backed by the same queue worker used by the internal admin mirror action
- requires `Authorization: Nostr <base64url-kind-24242-event>`
- the authorization event must satisfy BUD-11 for the `upload` action:
  - `kind == 24242`
  - valid signature
  - `created_at` is in the past
  - `expiration` tag is present and still in the future
  - at least one `t=upload` tag
  - at least one `x=<expected sha256>` tag matching the requested blob hash
  - if present, `server` tags must include the current Blossom host
- request body:

```json
{
  "url": "https://origin.example.com/blob/<sha256>",
  "sha256": "<sha256>"
}
```

- request validation rules:
  - `url` must be absolute `http` or `https`
  - `sha256` must be lowercase 64-char hex and must match the auth `x` tag
  - the handler must reject empty or malformed payloads before enqueueing
- success response is asynchronous to avoid request blocking and abuse amplification:

```json
{
  "status": "queued",
  "job_id": "<queue-job-id>",
  "sha256": "<sha256>",
  "url": "https://origin.example.com/blob/<sha256>"
}
```

- the worker downloads the remote body in the background, computes the SHA-256 over the complete content stream, and only persists the file when the computed digest matches the requested hash exactly
- a hash mismatch, MIME detection failure, policy rejection or storage failure marks the job as failed and nothing is stored locally
- internal admin flow and public BUD-04 flow share the same underlying mirror job implementation

### `PUT /media` and `HEAD /media` (BUD-05)

- `PUT /media` accepts binary media in the request body and behaves as a trusted upload+optimize entrypoint distinct from `PUT /upload`
- clients SHOULD send `Content-Type` and `Content-Length`
- clients MAY send `X-SHA-256: <lowercase-hex>` so the server can validate auth and reject mismatches before or during persistence
- requires `Authorization: Nostr <base64url-kind-24242-event>` with BUD-11 `t=media`
- if `X-SHA-256` is present, at least one auth `x` tag must match it exactly
- server flow:
  - validate Blossom auth and upload policy
  - stream the original body to local storage while computing SHA-256
  - persist the canonical object row for the original hash
  - enqueue one background optimization job bound to that original hash
  - return the blob descriptor for the stored original object immediately, plus processing state metadata

Example `PUT /media` success response:

```json
{
  "status": "processing",
  "message": "media optimization queued",
  "url": "https://cdn.example.com/blob/<sha256>",
  "sha256": "<sha256>",
  "size": 184292,
  "type": "image/png",
  "uploaded": 1725909682,
  "job_id": "<queue-job-id>",
  "processing": {
    "optimized": false,
    "thumbnail": false,
    "blurhash": false,
    "streaming": false
  }
}
```

- worker responsibilities in the current rollout:
  - metadata extraction from the original object whenever possible, enabled by default
  - image thumbnail generation in WebP, enabled by default
  - image optimized WebP generation, enabled by default
  - video thumbnail generation, disabled by default
  - blurhash generation from the optimized image or source-image fallback, enabled by default
  - width/height extraction for images and videos, enabled by default
  - duration and bitrate extraction for audio/video via `ffprobe`, enabled by default
  - optional HLS/MPEG-DASH manifest generation behind configuration, disabled by default

Active configuration surface:

```yaml
store:
  media_processing:
    enabled: true
    extract_metadata: true
    generate_blurhash: true
    image:
      generate_thumbnail: true
      generate_webp: true
      max_width: 1600
      thumbnail_max_width: 320
    video:
      generate_thumbnail: false
      generate_poster_webp: true
      thumbnail_max_width: 320
    streaming:
      enable_hls: false
      enable_dash: false
```

Failure model:

- if original persistence fails, `PUT /media` fails the request
- if a later extraction/transformation step fails, the worker records the error, logs context, marks partial availability, and preserves any metadata/derivatives already produced

`HEAD /media`:

- requires the same Blossom `kind:24242` authorization with `t=media`
- uses `X-SHA-256` as the canonical lookup key for the original object
- returns no body
- returns `200 OK` when the object exists; readiness is exposed through headers:
  - `X-SHA-256`
  - `X-Media-Status: pending|processing|ready|failed`
  - `X-Optimized-Available: true|false`
  - `X-Thumbnail-Available: true|false`
  - `X-Blurhash-Available: true|false`
  - `X-Streaming-Available: true|false`
  - `X-Processing-Job-Id` when a known active job exists
- returns `404` when the canonical hash is unknown locally
- `X-Media-Status=ready` means the configured worker steps have finished, not that every optional feature is globally enabled

### BUD-08 metadata enrichment

Whenever `PUT /media` or a queued `PUT /mirror` job completes successfully, the server refreshes the stored NIP-94-compatible metadata set from the extracted media facts produced by the BUD-05 processing routine.

Generation rules:

- `nip94_tags` are regenerated as an ordered JSON array of NIP-94 tag tuples, not patched field-by-field
- the minimum generated tags are:
  - `["url", "<canonical blob url>"]`
  - `["m", "<lowercase mime>"]`
  - `["x", "<canonical sha256>"]`
  - `["ox", "<original sha256>"]` when a derivative differs from the source object
  - `["size", "<bytes>"]`
  - `["dim", "<width>x<height>"]` when dimensions are known
- optional generated tags include:
  - `["blurhash", "<value>"]`
  - `["thumb", "<thumbnail url>", "<thumbnail sha256>"]`
  - `["image", "<optimized preview url>", "<optimized sha256>"]`
  - `["fallback", "<stream manifest url>"]` when HLS/DASH is enabled and published as an alternate delivery path
  - repeated `["fallback", "<mirror url>"]` entries for every known mirror URL, including the origin URL captured by BUD-04
- the canonical blob descriptor returned by upload/mirror routes may expose the same material as a `nip94` field derived from `nip94_tags`

### `PUT /report` (BUD-09)

The public Blossom server should accept a signed `kind:1984` NIP-56 report event in the request body.

Server behavior:

- extract one or more `x` tags as blob hashes
- persist one review-report row per targeted blob hash
- optionally retain related `e` and `p` tags for relay moderation correlation
- return `2xx` on acceptance or a clear `4xx/5xx` error on failure

This endpoint feeds the internal `/admin/blossom/reports` review workspace.

### BUD-10 Blossom IDs

When the dashboard exposes `Copiar Blossom ID`, it should generate URIs in the format:

```text
blossom:<sha256>.<ext>?xs=<server-host>&as=<uploader-pubkey>&sz=<bytes>
```

Rules:

- use the stored extension when known; otherwise default to `.bin`
- include at least one `xs` hint derived from the canonical server URL
- include `as` when uploader pubkey is known
- include `sz` when object size is known
