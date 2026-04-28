# Nostr Relay Server ⚡

![GitHub issues](https://img.shields.io/github/issues/gabrielmoura/nostr-relay-server?style=for-the-badge)
![GitHub forks](https://img.shields.io/github/forks/gabrielmoura/nostr-relay-server?style=for-the-badge)
![GitHub stars](https://img.shields.io/github/stars/gabrielmoura/nostr-relay-server?style=for-the-badge)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/gabrielmoura/nostr-relay-server)

High-performance Nostr relay written in Go, with PostgreSQL storage, optional Redis integration, embedded admin panel, Blossom-compatible media storage, Negentropy sync, optional NIP-29 groups, optional NIP-86 relay-management API, Prometheus metrics, and configurable cron jobs.

---

## Overview

Nostr Relay Server is a production-oriented relay implementation focused on performance, observability, and operational simplicity.

It provides:

- WebSocket relay core for the Nostr protocol
- PostgreSQL as the primary storage backend
- Optional Redis cache/pubsub integration
- Embedded admin API and admin panel at `/panel` for moderation, synchronization, and NIP-29/WoT management
- Optional NIP-86 JSON-RPC management on the public root endpoint
- Blossom-compatible media storage
- Negentropy-based relay synchronization and bulk event downloader
- Optional NIP-29 relay-based groups with relay-generated state events
- Prometheus metrics
- Import/export and operational CLI tools
- Cron jobs for maintenance and background routines

---

## Features

- **High performance relay core** for Nostr workloads
- **Admin API + Admin Panel** exposed on the internal server
- **Optional NIP-86 relay management API** using JSON-RPC + NIP-98 auth
- **PostgreSQL storage** with support for operational tooling
- **Optional Redis support** for cache and pub/sub scenarios
- **Blossom media storage support** for blobs and uploads
- **Negentropy sync** with compatible relays
- **Optional NIP-29 groups** with configurable admission, moderation, invite codes, PoW and timeline-reference enforcement
- **Prometheus metrics** for monitoring and observability
- **Configurable cron jobs** for cleanup and maintenance
- **JSONL import/export** for migration and backup workflows
- **Flexible configuration** through `conf.yaml`

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Default Endpoints](#default-endpoints)
3. [Requirements](#requirements)
4. [CLI Commands](#cli-commands)
5. [Admin API and Panel](#admin-api-and-panel)
6. [NIP-86 Relay Management](#nip-86-relay-management)
7. [Configuration](#configuration)
8. [Limitations and Operational Caveats](#limitations-and-operational-caveats)
9. [Monitoring with Prometheus and Grafana](#monitoring-with-prometheus-and-grafana)
10. [NIP-29 Groups](#nip-29-groups)
11. [Supported Protocols](#supported-protocols)
12. [Documentation](#documentation)
13. [Frontend Development](#frontend-development)
14. [Build](#build)
15. [FAQ](#faq)
16. [Contributing](#contributing)
17. [License](#license)

---

## Quick Start

### Requirements

- Go `1.24+`
- PostgreSQL
- Redis *(optional)*

### 1. Generate the configuration file

```bash
go run ./cmd/nrserver conf write
````

This creates the default runtime configuration file:

```text
conf.yaml
```

### 2. Edit `conf.yaml`

At minimum, review and adjust:

* `db.postgres_uri`
* relay ports
* relay information
* `relay_information.limitation.max_subscriptions` if you want to cap active subscriptions per connection
* storage settings
* `admin_token` if you want to protect the admin API
* `nip86.enabled` and `admin_pubkey` if you want to expose NIP-86 relay management

### 3. Run database migrations

```bash
go run ./cmd/nrserver seed
```

### 4. Start the relay

```bash
go run ./cmd/nrserver server
```

Optional bootstrap events:

```bash
go run ./cmd/nrserver server --bootstrap
```

You can also use the built binary instead of `go run`:

```bash
nrserver server
```

---

## Default Endpoints

By default, the project starts two HTTP servers:

### External relay server

Used by Nostr clients.

* **Relay / NIP-11:** `http://localhost:9090`
* **NIP-86 JSON-RPC:** `POST /` with `Content-Type: application/nostr+json+rpc` *(optional, disabled by default)*

### Internal server

Used for administration, metrics, and the embedded panel.

* **Admin API:** `http://localhost:9091/admin`
* **Admin Panel:** `http://localhost:9091/panel`
* **Metrics:** `http://localhost:9091/metrics`

> The internal server usually runs on `port + 1`.

---

## CLI Commands

Run commands with:

```bash
go run ./cmd/nrserver <command>
```

Or, if you already built the binary:

```bash
nrserver <command>
```

### `server`

Starts the external and internal servers.

```bash
nrserver server [flags]
```

| Flag                | Description                               | Default |
| :------------------ | :---------------------------------------- | :------ |
| `-b`, `--bootstrap` | Creates initial events and bootstrap data | `false` |
| `-c`, `--config`    | Enables loading configuration from file   | `true`  |
| `-h`, `--help`      | Shows command help                        | -       |

---

### `seed`

Prepares database schema and optional bootstrap relay events.

```bash
nrserver seed [flags]
```

| Flag | Description | Default |
| :--- | :--- | :--- |
| `--bootstrap` | Creates relay bootstrap events (kind `0`, `411`, `10002`, `10063`) after migration | `false` |
| `--bootstrap-idempotent` | Skips bootstrap insertion when marker events already exist (requires `--bootstrap`) | `false` |
| `--skip-migrate` | Skips migration step (requires `--bootstrap`) | `false` |
| `--dry-run` | Prints planned actions without writing to DB | `false` |
| `--timeout` | Migration timeout duration | `30s` |

Examples:

```bash
nrserver seed
nrserver seed --bootstrap
nrserver seed --bootstrap --bootstrap-idempotent
nrserver seed --bootstrap --skip-migrate
nrserver seed --dry-run
```

Operational notes:

- Requires valid config and DB connectivity (except `--dry-run`).
- `--bootstrap` generates a new keypair and inserts bootstrap events.
- `--bootstrap-idempotent` checks marker tag `nrserver-bootstrap:<canonical_url>` before insertion.
- Re-running `--bootstrap` creates additional events; use with operational intent.

---

### `conf`

Inspects, validates and generates configuration files.

```bash
nrserver conf <subcommand>
```

Available subcommands:

- `print` (`show` alias): print default configuration template
- `effective`: load and print effective runtime configuration
- `validate`: validate config and enabled cron schedules
- `write`: write default config template to file

#### `conf print`

Prints the full configuration with defaults.

```bash
nrserver conf print
nrserver conf print --format json
```

#### `conf effective`

Prints configuration as loaded by runtime (defaults + file values).

```bash
nrserver conf effective
nrserver conf effective --file ./conf.yaml --format json
```

#### `conf validate`

Validates required fields and semantic checks.

```bash
nrserver conf validate
nrserver conf validate --file ./conf.yaml
```

#### `conf write`

Writes the default `conf.yaml` file.

```bash
nrserver conf write
nrserver conf write --file /etc/nrs/conf.yaml --force
```

---

### `import`

Imports events from a `.jsonl` file.

```bash
nrserver import [flags]
```

| Flag                  | Description                | Default        |
| :-------------------- | :------------------------- | :------------- |
| `-f`, `--file`        | JSONL file to import       | `events.jsonl` |
| `-b`, `--batch-size`  | Number of events per batch (`0` = line mode) | `100`          |
| `-w`, `--num-workers` | Parallel import workers    | `2`            |
| `--stats-interval`    | Import stats log interval (`0` disables) | `5s` |
| `--fail-on-error`     | Return non-zero when row-level errors occur | `false` |

Examples:

```bash
nrserver import --file events.jsonl
nrserver import --file events.jsonl --batch-size 500 --num-workers 4
nrserver import --batch-size 0 --num-workers 8 --fail-on-error
```

---

### `export`

Exports events to a `.jsonl` file.

```bash
nrserver export [flags]
```

| Flag                     | Description                | Default                  |
| :----------------------- | :------------------------- | :----------------------- |
| `-f`, `--file`           | Destination file           | `export-TIMESTAMP.jsonl` |
| `--format`               | Export format: `jsonl` or `tsv` | `jsonl` |
| `--filter`               | Optional Nostr filter JSON object | - |
| `--filter-file`          | Optional path to JSON file with filter object | - |
| `--limit`                | Maximum number of exported events (`0` = unlimited) | `0` |
| `--segment-size`         | Events per output segment file (`0` = disabled) | `0` |
| `--no-header`            | Do not write TSV header line | `false` |
| `--overwrite`            | Allow overwriting existing output files | `false` |
| `-b`, `--batch-size`     | Number of events per batch | `100`                    |
| `-w`, `--writer-workers` | Parallel encoder workers   | `2`                      |

Examples:

```bash
nrserver export
nrserver export --filter '{"authors":["<hex-pubkey>"]}'
nrserver export --filter-file ./filter.json
nrserver export --limit 1000 --segment-size 500
nrserver export --format tsv --file events.tsv
nrserver export --format tsv --segment-size 500 --no-header --overwrite
```

Segmentation behavior:

- With `--segment-size 500`, files are rotated every 500 events.
- Output naming uses indexed suffixes, e.g.:
  - `export-1776391752-001.jsonl`
  - `export-1776391752-002.jsonl`
  - `export-1776391752-001.tsv`

---

### `sync`

Synchronizes events with a remote relay using Negentropy.

```bash
nrserver sync <url> [flags]
```

| Flag             | Description                                                                    | Default |
| :--------------- | :----------------------------------------------------------------------------- | :------ |
| `-r`, `--remote` | Remote relay URL (`ws://` or `wss://`). Optional when using positional `<url>` | -       |
| `-p`, `--pk`     | Public key (hex or npub) used as author constraint                            | -       |
| `-d`, `--dir`    | Sync direction: `both`, `down`, `up`, `none`                                   | `both`  |
| `--direction`    | Deprecated alias for `--dir`                                                    | `both`  |
| `--filter`       | Nostr filter JSON (single object or array of filters)                           | `{}`    |
| `--timeout`      | Abort sync after N seconds without activity (`0` disables timeout)              | `0`     |

> This requires the remote relay to support Negentropy.

#### Sync behavior in practice

The sync command performs four stages:

1. Load local events and build local Negentropy vector
2. Open WebSocket session and start reconciliation (`NEG-OPEN`/`NEG-MSG`)
3. Upload events the remote side is missing (`EVENT`)
4. Download missing remote events using batched `REQ` filters + `EOSE`

Direction semantics:

- `both`: upload and download
- `up`: upload only
- `down`: download only
- `none`: reconcile sets only (no event transfer)

Filter semantics:

- A single filter object runs one sync pass.
- A filter array runs one pass per filter segment to preserve compatibility with relays that require object filters in `NEG-OPEN`.

Compatibility notes:

- Works with relays that support standard Negentropy session messages.
- Includes compatibility logic for Strfry-style sync transfer (`EVENT` + `REQ`), not only legacy `NEG-HAVE`/`NEG-NEED` data exchange.
- Handles relay-side request limits by batching IDs, reducing `CLOSED` responses for oversized filters.

Operational recommendations:

- Enable Negentropy on your local relay: `enable_negentropy: true`.
- Prefer `wss://` remotes in production.
- Start with a smaller synchronization scope (e.g., `--pk`) for first-time tests.
- Watch runtime logs for `NEG-ERR`, `NOTICE`, and `CLOSED` responses from the remote relay.

---

### `download`

Downloads events from relays into the local database.

```bash
nrserver download [flags]
```

| Flag               | Description                                                                                                      | Default |
| :----------------- | :--------------------------------------------------------------------------------------------------------------- | :------ |
| `-r`, `--relay-url` | Relay URL list to connect to                                                                                    | `wss://relay.damus.io` |
| `-p`, `--public-key` | Public key (hex or `npub`) used as author filter                                                               | - |
| `-k`, `--kinds` | Event kinds to download                                                                                           | `[1,30023,6,30003,30007,30008,30009,2003,2004,1063,42,41,40,0,1984,14]` |
| `-t`, `--tags` | Values applied to `#t` tag filter                                                                                  | `[]` |
| `-m`, `--mentioned` | Uses `#p=<public-key>` filter instead of author filter (requires `--public-key`)                               | `false` |
| `-o`, `--timeout` | Per-page timeout (seconds)                                                                                        | `30` |
| `--filter` | Optional Nostr filter JSON object for additional constraints (e.g. `ids`, `since`, `until`, `search`, extra tags) | - |
| `--filter-file` | Path to JSON file containing the same object accepted by `--filter`                                                  | - |
| `--filter-merge` | Merge strategy between JSON filter and explicit flags: `override` or `strict-conflict`                              | `override` |

Filter precedence rule (explicit):

- Base filter comes from `--filter` JSON when provided.
- Specific CLI flags override overlapping fields to preserve existing command behavior:
  - `--kinds` overrides `kinds`
  - `--tags` overrides `#t`
  - `--public-key` sets `authors=[pk]`
  - `--mentioned --public-key` sets `#p=[pk]` and clears `authors`
- With `--filter-merge strict-conflict`, conflicting values fail fast with a clear CLI error.

More examples and advanced behavior: `docs/download-command.md`.

---

### `cron`

Runs configured maintenance jobs from `cron.*` settings.

```bash
nrserver cron [flags]
```

| Flag | Description | Default |
| :--- | :--- | :--- |
| `--list` | List jobs and current enable/schedule state | `false` |
| `--run-once` | Execute selected enabled jobs once and exit | `false` |
| `--job` | Filter by job name (`db_optimization`, `reported_events_fetch`, `delete_old_events`, `nip40`) | all |
| `--timeout` | Per-job execution timeout | `30m` |

Examples:

```bash
nrserver cron
nrserver cron --list
nrserver cron --run-once
nrserver cron --run-once --job db_optimization
nrserver cron --job nip40 --timeout 5m
```

Jobs currently available:

- `db_optimization`
- `reported_events_fetch`
- `delete_old_events`
- `nip40`

Only enabled jobs run (from `conf.yaml`).

---

## Admin API and Panel

The internal server exposes:

* `/admin/*` → Admin API
* `/panel` → Embedded admin SPA
* `/metrics` → Prometheus metrics endpoint

If `admin_token` is set in `conf.yaml`, requests to `/admin/*` must include:

```text
X-Admin-Token: <your_token>
```

This is useful to protect administrative operations without exposing them publicly.

For browser users and operators:

- `/panel` is intended for trusted internal networks, VPNs, or reverse-proxied admin environments.
- `/admin/*` is a backend/admin surface and should not be exposed to untrusted public traffic without deliberate controls.
- The dashboard uses the internal admin API, not the external NIP-86 JSON-RPC surface.

---

## NIP-86 Relay Management

NIP-86 support is available but **disabled by default**.

When enabled, the relay accepts JSON-RPC management requests on the same public root endpoint used for WebSocket upgrade.

### What it does

- `supportedmethods`
- `banpubkey` / `unbanpubkey`
- `allowpubkey` / `unallowpubkey`
- `banevent` / `allowevent`
- `blockip` / `unblockip`
- `changerelayname` / `changerelaydescription`

### How it is protected

All NIP-86 requests require:

- `Content-Type: application/nostr+json+rpc`
- `Authorization: Nostr <base64-event>`
- NIP-98 event `kind:27235`
- valid signature
- matching `method` tag
- matching `u` tag for the relay URL
- matching `payload` tag with SHA-256 of the request body
- caller pubkey equal to `admin_pubkey`

### Minimum config to enable it

```yaml
admin_pubkey: "<hex-or-npub-admin-pubkey>"

nip86:
  enabled: true
  auth_window_seconds: 60
  cache_ttl_seconds: 300
```

### Important notes for operators

- NIP-86 is optional. Leaving `nip86.enabled: false` keeps the relay behavior unchanged.
- Enabling NIP-86 without a stable public `relay_information.url` is not supported.
- `blockip` disconnects active WebSocket sessions on the current relay node immediately after persistence succeeds.
- In multi-node deployments, durable IP blocks propagate through shared persistence, but immediate disconnect of already-open sessions is only guaranteed on the local node unless you add your own cross-node coordination.
- The admin dashboard does **not** sign NIP-98 requests in the browser. It continues to use the internal `/admin/*` API.

---

## Configuration

The runtime configuration file is:

```text
conf.yaml
```

Below is a representative example with the main sections:

```yaml
port: 9090
app_env: development
admin_token: ""
admin_pubkey: ""

nip86:
  enabled: false
  auth_window_seconds: 60
  cache_ttl_seconds: 300

ws:
  rate_limit: 1
  burst: 5
  auth_mode: flexible

relay_information:
  url: http://localhost:9090
  name: Nostr Relay Server
  description: High-performance Nostr relay in Go
  pub_key: ""
  priv_key: ""
  supported_nips: [1, 2, 4, 9, 11, 17, 18, 25, 40, 42, 45, 50, 62, 77, 96, 98]
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
  filter_limit: 9999999999
  reporting_limit: 5
  enable_anonymous_req: true
  max_tag_value_length: 150

db:
  max_conns: 10
  min_conns: 1
  postgres_uri: postgres://postgres:Strong@P4ssword@127.0.0.1:5432/nostr

stream:
  relays:
    - wss://nostr.azzamo.net
    - wss://relay.damus.io
  stream_up: false
  stream_down: false

store:
  enabled: true
  api_path: http://localhost:9090/upload
  media_path: http://localhost:9090/blob
  accepted_mimetypes:
    - image/jpeg
    - image/png
    - video/mp4
  allow_adult_content: false
  allow_violent_content: false
  names: []

cron:
  enabled: true
```

### Important configuration notes

Some especially relevant settings:

* `ws.auth_mode`: `strict` | `flexible` | `optional` | `none`
* `cron.*`: cron job toggles and schedules
* `cron.nip40.*`: expiration cleanup configuration
* `stream.*`: upstream/downstream relay streaming
* `store.*`: Blossom media server settings
* `nip29.*`: optional relay-based group support, admission, moderation, invite, PoW and timeline-reference rules
* `admin_token`: enables token protection for internal admin API
* `admin_pubkey`: required only when `nip86.enabled=true`
* `nip86.*`: enables and tunes the external relay-management surface

For the complete schema and production-oriented examples, see:

* `docs/configuration.md`

### Recommended production posture

- Keep `admin_token` set when using the embedded admin panel or internal admin API.
- Keep `nip86.enabled: false` unless you explicitly need Nostr-native remote relay management.
- Never commit `db.postgres_uri`, relay private keys, or production admin values into public repositories.

---

## Limitations and Operational Caveats

Before enabling all optional features, keep these limitations in mind:

- PostgreSQL is mandatory; Redis is optional.
- WebSocket payloads still have a hard upper bound and request limits may reject abusive filters.
- NIP-86 is intentionally narrow: it covers relay management actions, not a full browser-facing admin workflow.
- `blockip` is immediate only for sessions known by the local process.
- Runtime relay metadata overrides persist in PostgreSQL; they do not rewrite `conf.yaml`.
- The internal dashboard currently uses the internal admin API as its source of truth; it is not a direct NIP-86 client.
- Large frontend bundles can trigger Vite chunk-size warnings until more route-level code splitting is introduced.

For end users running a public relay:

- If you only want a normal relay plus dashboard, keep NIP-86 disabled.
- If you expose `/panel`, place it behind a trusted network boundary or reverse proxy authentication.
- If you enable groups, Blossom, stream federation, and NIP-86 together, test each feature set independently before combining them in production.

---

## NIP-29 Groups

The relay now includes an optional NIP-29 groups module.

When disabled, the relay behaves exactly as before.

When enabled, it adds:

- Relay-scoped group state stored in PostgreSQL
- Relay-generated state events `39000`, `39001`, `39002`, `39003`
- Validation for NIP-29 moderation and membership flows
- Optional invite-code flow via `kind:9009` + `9021` `code` tag
- Optional PoW enforcement for group writes and moderation events
- Optional timeline-reference validation using `previous` tags
- Redis-backed hot-path caches for group metadata, membership, bans, invites and recent timeline references

### High-level behavior

- Group content continues to use the main `event` table.
- Group state lives in dedicated `nip29_*` tables.
- The relay signs generated metadata/admin/member/role events with `relay_information.priv_key`.
- `REQ` and `COUNT` are filtered by group visibility rules when applicable.

### Main configuration block

Representative example:

```yaml
nip29:
  enabled: false
  relay_scope: ""
  cache_ttl_seconds: 60
  membership_cache_ttl_seconds: 30
  ban_cache_ttl_seconds: 30
  timeline_cache_ttl_seconds: 300
  group_creator_role: admin
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
  pow:
    enabled: false
    default_min_difficulty: 0
    moderation_min_difficulty: 0
  timeline:
    enabled: false
    required_on_moderation: false
    min_references: 0
    recent_window: 50
```

See also:

- `docs/configuration.md`
- `docs/data-schema.md`
- `docs/nip29-coordination.md`

---

## Monitoring with Prometheus and Grafana

The internal server exposes Prometheus metrics at:

```text
http://localhost:9091/metrics
```

### Prometheus example

Create a `prometheus.yml` file:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "nostr-relay"
    scrape_interval: 5s
    static_configs:
      - targets: ["host.docker.internal:9091"]
```

If you are not using Docker, replace `host.docker.internal:9091` with:

```text
localhost:9091
```

Run Prometheus:

```bash
docker run -d \
  --name prometheus \
  -p 9092:9090 \
  -v ./prometheus.yml:/etc/prometheus/prometheus.yml \
  --add-host=host.docker.internal:host-gateway \
  prom/prometheus
```

> `9092` is used above to avoid conflicting with the relay’s default port `9090`.

### Grafana example

Run Grafana:

```bash
docker run -d \
  --name grafana \
  -p 3000:3000 \
  --add-host=host.docker.internal:host-gateway \
  grafana/grafana-enterprise
```

Then:

1. Open `http://localhost:3000`
2. Login with the default credentials: `admin` / `admin`
3. Add Prometheus as a data source
4. Import the project dashboard if available

### Negentropy metrics to watch

When Negentropy is enabled, these metrics are especially useful:

- `nostr_negentropy_v2_requests_total{operation,result}`: protocol operations by outcome
- `nostr_negentropy_v2_cache_total{backend,result}`: cache hit/miss/error by backend (memory/redis)
- `nostr_negentropy_v2_sessions_active`: currently active reconciliation sessions
- `nostr_negentropy_v2_protocol_errors_total`: protocol-level errors returned to clients
- `nostr_negentropy_v2_events_imported_total`: events imported during synchronization

Quick PromQL examples:

```promql
sum by (operation, result) (rate(nostr_negentropy_v2_requests_total[5m]))
```

```promql
sum by (backend, result) (rate(nostr_negentropy_v2_cache_total[5m]))
```

### NIP-29 metrics to watch

When NIP-29 is enabled, these metrics are especially useful:

- `nostr_nip29_groups_created_total`
- `nostr_nip29_groups_active`
- `nostr_nip29_events_received_total{kind}`
- `nostr_nip29_events_rejected_total{reason}`
- `nostr_nip29_invites_generated_total`
- `nostr_nip29_invites_consumed_total`
- `nostr_nip29_processing_seconds{operation}`
- `nostr_nip29_cache_total{cache,result}`

---

## Supported Protocols

### NIPs

The project currently documents support for:

* NIP-01 — Basic protocol
* NIP-02 — Follow list
* NIP-04 — Encrypted direct messages
* NIP-09 — Event deletion
* NIP-13 — Proof of work
* NIP-11 — Relay information document
* NIP-17 — Relay list metadata
* NIP-18 — Public chat
* NIP-29 — Relay-based groups *(optional)*
* NIP-25 — Reactions
* NIP-40 — Expiration timestamp
* NIP-42 — Authentication of clients to relays
* NIP-45 — Event counts
* NIP-50 — Search capability
* NIP-62 — Request to vanish
* NIP-77
* NIP-86 — Relay management *(optional)*
* NIP-96 — File storage / Blossom
* NIP-98 — HTTP auth

### Blossom / BUDs

* BUD-01
* BUD-02

---

## Documentation

Project documentation is primarily organized under `docs/`, with unified setup guides at the repository root:

* `docs/api-spec.md` — WebSocket protocol, HTTP routes, admin endpoints, payloads, and error envelopes
* `docs/configuration.md` — Complete `conf.yaml` schema, defaults, and production examples
* `docs/nip86-management.md` — NIP-86 architecture, schema additions, auth model, and runtime trade-offs
* `docs/architecture.md` — C4 architecture, stack, flow, and module layout
* `docs/policies.md` — Event and REQ policy rules
* `docs/data-schema.md` — PostgreSQL and Redis schema, indexes, and cache/pubsub keys
* `docs/decisions.md` — ADR history and architecture decisions
* `docs/cli.md` — CLI behavior, command UX, flags, and operational guidance
* `docs/download-command.md` — Download command flow, `--filter` semantics, precedence, and troubleshooting
* `docs/nip29-coordination.md` — implementation status, risks, toggles, metrics and migration notes for NIP-29
* `docs/wot.md` — Web of Trust calculation, caching mechanism, and architecture
* `docs/todo.md` — Roadmap and implementation checklist
* `nrserver.adoc` — Main Technical Documentation (AsciiDoc)
* `nrserver.md` — Unified environment setup and operations guide (Markdown)

---

## Frontend Development

The admin SPA source code lives in:

```text
infra/dash
```

### Run locally

```bash
cd infra/dash
pnpm install
pnpm dev
```

By default, Vite proxies `/admin` and `/metrics` to:

```text
http://localhost:4870
```

Adjust with `ADMIN_PROXY_TARGET` if needed.

### Rebuild embedded assets

To rebuild the frontend assets embedded into the Go application:

```bash
cd infra/dash
pnpm build
```

---

## Build

Build the binary:

```bash
go build -o nrserver ./cmd/nrserver
```

Or use the `Makefile` targets:

* `make linux-pc`
* `make linux-rpi`
* `make windows`
* `make windows32`

---

## FAQ

### Can I run this on a Raspberry Pi?

Yes. A typical setup is:

1. Install Go or Docker on the Raspberry Pi
2. Run PostgreSQL in a compatible ARM image or natively
3. Adjust `conf.yaml`
4. Build for ARM or use an ARM-compatible deployment flow

If you already use containerized infrastructure, Docker is usually the easiest operational path.

---

### Can I expose the relay through Tor?

Yes. A common approach is:

1. Install and configure Tor on the server
2. Create a `HiddenService` in `torrc`
3. Point the hidden service to the local relay port
4. Update `canonical_url` in `conf.yaml` to the `.onion` address
5. Optionally use `HTTP_PROXY` / environment-based proxy routing if required by your deployment model

---

### Is Redis required?

No. PostgreSQL is required. Redis is optional.

---

### Does the project provide an admin UI?

Yes. The embedded admin panel is exposed at:

```text
http://localhost:9091/panel
```

It supports user moderation, event inspection, Negentropy synchronization, bulk event downloading, NIP-29 group management, and Web of Trust (WoT) trusted pubkey administration.

---

## Contributing

Contributions are welcome.

You can help by:

* reporting bugs
* suggesting improvements
* submitting pull requests
* improving documentation
* enhancing dashboards and operational tooling

### Suggested workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Open a Pull Request with a clear description

If you found a bug or want to discuss an idea first, open an issue.

---

## License

This project does not currently define a license.

Until a license is added, review the repository terms carefully before redistributing or using the code in ways that depend on explicit licensing permissions.
