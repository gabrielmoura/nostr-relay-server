# Nostr Relay Server ⚡

![GitHub issues](https://img.shields.io/github/issues/gabrielmoura/nostr-relay-server?style=for-the-badge)
![GitHub forks](https://img.shields.io/github/forks/gabrielmoura/nostr-relay-server?style=for-the-badge)
![GitHub stars](https://img.shields.io/github/stars/gabrielmoura/nostr-relay-server?style=for-the-badge)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/gabrielmoura/nostr-relay-server)

High-performance Nostr relay written in Go, with PostgreSQL storage, optional Redis integration, embedded admin panel, Blossom-compatible media storage, Negentropy sync, Prometheus metrics, and configurable cron jobs.

---

## Overview

Nostr Relay Server is a production-oriented relay implementation focused on performance, observability, and operational simplicity.

It provides:

- WebSocket relay core for the Nostr protocol
- PostgreSQL as the primary storage backend
- Optional Redis cache/pubsub integration
- Embedded admin API and admin panel at `/panel`
- Blossom-compatible media storage
- Negentropy-based relay synchronization
- Prometheus metrics
- Import/export and operational CLI tools
- Cron jobs for maintenance and background routines

---

## Features

- **High performance relay core** for Nostr workloads
- **Admin API + Admin Panel** exposed on the internal server
- **PostgreSQL storage** with support for operational tooling
- **Optional Redis support** for cache and pub/sub scenarios
- **Blossom media storage support** for blobs and uploads
- **Negentropy sync** with compatible relays
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
6. [Configuration](#configuration)
7. [Monitoring with Prometheus and Grafana](#monitoring-with-prometheus-and-grafana)
8. [Supported Protocols](#supported-protocols)
9. [Documentation](#documentation)
10. [Frontend Development](#frontend-development)
11. [Build](#build)
12. [FAQ](#faq)
13. [Contributing](#contributing)
14. [License](#license)

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
* storage settings
* `admin_token` if you want to protect the admin API

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

Runs database migrations / initial database preparation.

```bash
nrserver seed
```

---

### `conf print`

Prints the full configuration with defaults.

```bash
nrserver conf print
```

### `conf write`

Writes the default `conf.yaml` file.

```bash
nrserver conf write
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
| `-b`, `--batch-size`  | Number of events per batch | `100`          |
| `-w`, `--num-workers` | Parallel import workers    | `2`            |

---

### `export`

Exports events to a `.jsonl` file.

```bash
nrserver export [flags]
```

| Flag                     | Description                | Default                  |
| :----------------------- | :------------------------- | :----------------------- |
| `-f`, `--file`           | Destination file           | `export-TIMESTAMP.jsonl` |
| `-b`, `--batch-size`     | Number of events per batch | `100`                    |
| `-w`, `--writer-workers` | Parallel writer workers    | `2`                      |

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

Runs the configured cron scheduler / maintenance jobs.

```bash
nrserver cron
```

Typical responsibilities may include:

* expiration cleanup
* scheduled maintenance routines
* usage/statistics refresh jobs

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

admin_token: ""
```

### Important configuration notes

Some especially relevant settings:

* `ws.auth_mode`: `strict` | `flexible` | `optional` | `none`
* `cron.*`: cron job toggles and schedules
* `cron.nip40.*`: expiration cleanup configuration
* `stream.*`: upstream/downstream relay streaming
* `store.*`: Blossom media server settings
* `admin_token`: enables token protection for admin API

For the complete schema and production-oriented examples, see:

* `docs/configuration.md`

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

---

## Supported Protocols

### NIPs

The project currently documents support for:

* NIP-01 — Basic protocol
* NIP-02 — Follow list
* NIP-04 — Encrypted direct messages
* NIP-09 — Event deletion
* NIP-11 — Relay information document
* NIP-17 — Relay list metadata
* NIP-18 — Public chat
* NIP-25 — Reactions
* NIP-40 — Expiration timestamp
* NIP-42 — Authentication of clients to relays
* NIP-45 — Event counts
* NIP-50 — Search capability
* NIP-62 — Request to vanish
* NIP-77
* NIP-96 — File storage / Blossom
* NIP-98 — HTTP auth

### Blossom / BUDs

* BUD-01
* BUD-02

---

## Documentation

Project documentation is organized under `docs/`:

* `docs/api-spec.md` — WebSocket protocol, HTTP routes, admin endpoints, payloads, and error envelopes
* `docs/configuration.md` — Complete `conf.yaml` schema, defaults, and production examples
* `docs/architecture.md` — C4 architecture, stack, flow, and module layout
* `docs/policies.md` — Event and REQ policy rules
* `docs/data-schema.md` — PostgreSQL and Redis schema, indexes, and cache/pubsub keys
* `docs/decisions.md` — ADR history and architecture decisions
* `docs/download-command.md` — Download command flow, `--filter` semantics, precedence, and troubleshooting
* `docs/todo.md` — Roadmap and implementation checklist

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
