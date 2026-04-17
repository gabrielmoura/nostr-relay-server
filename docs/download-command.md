# Download Command Reference

## Overview

`nrserver download` fetches events from one or more relays and persists them in the local database.

This command is intended for operational backfill and targeted ingestion scenarios.

## Command

```bash
nrserver download [flags]
```

## Supported Flags

| Flag | Type | Description | Default |
|---|---|---|---|
| `-r`, `--relay-url` | `string[]` | Relay URLs to connect to | `wss://relay.damus.io` |
| `-p`, `--public-key` | `string` | Author public key (hex or `npub`) | empty |
| `-k`, `--kinds` | `int[]` | Kinds to download | `[1,30023,6,30003,30007,30008,30009,2003,2004,1063,42,41,40,0,1984,14]` |
| `-t`, `--tags` | `string[]` | Values for `#t` tag filter | empty |
| `-m`, `--mentioned` | `bool` | Match by `#p=<public-key>` instead of author | `false` |
| `-o`, `--timeout` | `int` | Per-page timeout in seconds | `30` |
| `--filter` | `string` | Optional Nostr filter JSON object | empty |
| `--filter-file` | `string` | Path to JSON file with filter object | empty |
| `--filter-merge` | `string` | Merge strategy: `override` or `strict-conflict` | `override` |

## Refactored Execution Flow

The command flow is now split into clear stages:

1. CLI flag parsing in `cmd/down.go`
2. Option normalization and validation in `cmd/internal/down/options.go`
3. Filter merge (legacy flags + `--filter`) in `cmd/internal/down/options.go`
4. Runtime setup (config, logger, DB) in `cmd/internal/down/download.go`
5. Concurrent relay download orchestration in `cmd/internal/down/download.go`
6. Paginated fetch + persistence loop in `cmd/internal/down/fetch.go`

This separation keeps parsing, validation, relay execution and persistence concerns isolated and easier to evolve.

## `--filter` JSON Semantics

`--filter` and `--filter-file` accept a single JSON object compatible with Nostr filter fields.

Accepted examples:

```json
{"authors":["<hex-pubkey>"],"kinds":[1],"since":1700000000}
```

```json
{"#e":["<event-id>"],"search":"nostr relay"}
```

```json
{"authors":["<hex-pubkey>"],"#p":["<other-pubkey>"],"#t":["go","nostr"]}
```

Important notes:

- The payload must be a JSON object.
- Use only one source at a time: `--filter` or `--filter-file`.
- Invalid JSON returns an explicit CLI error.
- The command performs pagination internally, so request `limit` is always controlled by the command page size.

## Precedence and Interaction Rules

To preserve backward compatibility, specific flags override overlapping fields from `--filter`.

Default merge mode is `override`.

Merge order:

1. Start from `--filter` object (or empty filter)
2. Apply `--kinds` (overrides `kinds`)
3. Apply `--tags` (overrides `#t`)
4. Apply `--public-key`:
   - without `--mentioned`: `authors=[public-key]`
   - with `--mentioned`: `#p=[public-key]` and `authors` is cleared

Validation rule:

- `--mentioned` requires `--public-key`

### `strict-conflict` merge mode

When `--filter-merge strict-conflict` is used, conflicting values between JSON filter and explicit flags fail fast.

Examples of conflict checks:

- JSON `kinds` vs `--kinds`
- JSON `#t` vs `--tags`
- JSON `authors` vs `--public-key`
- JSON `#p`/`authors` vs `--mentioned --public-key`

## Error Behavior

Examples of expected errors:

- Invalid filter JSON:

```text
invalid --filter JSON: ...
```

- Non-object filter payload:

```text
invalid --filter JSON: expected object
```

- Invalid flag combination:

```text
invalid flag combination: --mentioned requires --public-key
```

- Invalid timeout:

```text
invalid --timeout <value>: must be greater than 0
```

- Invalid mixed filter source:

```text
invalid filter source: use only one of --filter or --filter-file
```

- Invalid strict merge strategy:

```text
invalid --filter-merge "...": expected one of override/strict-conflict
```

If all configured relays fail, command returns a final error. If at least one relay succeeds, per-relay failures are logged and execution completes.

## Usage Examples

Basic author download:

```bash
nrserver download --relay-url wss://relay.damus.io --public-key <hex-or-npub>
```

Mention-only mode (`#p`):

```bash
nrserver download --relay-url wss://relay.damus.io --mentioned --public-key <hex-or-npub>
```

Add extra filter constraints via JSON:

```bash
nrserver download --public-key <hex-or-npub> --filter '{"since":1700000000,"search":"nostr"}'
```

Use filter from file:

```bash
nrserver download --filter-file ./filter.json --relay-url wss://relay.damus.io
```

Fail on conflicting filters:

```bash
nrserver download \
  --filter-file ./filter.json \
  --kinds 1 \
  --filter-merge strict-conflict
```

Multiple relays with explicit kinds and tags:

```bash
nrserver download \
  --relay-url wss://relay.damus.io \
  --relay-url wss://nos.lol \
  --kinds 1,30023 \
  --tags go,nostr
```

## Maintenance Impact

The refactor improves maintenance by:

- keeping filter parsing and merge logic in one place
- reducing hidden coupling between CLI flags and runtime execution
- making edge-case validation explicit and testable
- isolating pagination/persistence internals for safer future changes

Related test coverage:

- `cmd/internal/down/options_test.go`

## Download Metrics

The refactor also adds dedicated download metrics:

- `nostr_download_events_received_total{relay}`
- `nostr_download_events_persisted_total{relay}`
- `nostr_download_duplicates_total{relay}`
- `nostr_download_failures_total{relay}`
- `nostr_download_page_latency_seconds{relay}`

These metrics allow per-relay visibility for ingestion quality and performance.
