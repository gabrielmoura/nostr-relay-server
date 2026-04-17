# CLI Reference (`nrserver`)

## Overview

The CLI is built with Cobra and follows a command-oriented operations model:

- runtime control (`server`)
- data maintenance (`seed`, `cron`)
- configuration workflows (`conf`)
- data mobility (`import`, `export`, `download`, `sync`)

This document focuses on command ergonomics and operational behavior for:

- `download`
- `import`
- `export`
- `seed`
- `cron`
- `conf`

## Root Command UX

`nrserver` now provides:

- clearer root description aligned with operational usage
- `SilenceUsage` and `SilenceErrors` for cleaner CLI failures
- removal of placeholder root flag (`--toggle`)

## `download`

See full technical details in `docs/download-command.md`.

Highlights:

- accepts filter from inline JSON (`--filter`) or file (`--filter-file`)
- supports merge strategy (`--filter-merge=override|strict-conflict`)
- emits per-relay metrics for received/persisted/duplicates/failures/page latency

## `seed`

### Purpose

Prepare local DB schema and optionally generate relay bootstrap events.

### Flags

| Flag | Description | Default |
|---|---|---|
| `--bootstrap` | create bootstrap relay events after migration | `false` |
| `--bootstrap-idempotent` | skip bootstrap insertion if marker already exists (requires `--bootstrap`) | `false` |
| `--skip-migrate` | skip migration step (requires `--bootstrap`) | `false` |
| `--dry-run` | print planned actions only | `false` |
| `--timeout` | migration timeout | `30s` |

### Execution Flow

1. Build and validate CLI options
2. If `--dry-run`, print execution plan and exit
3. Load config and initialize logger + DB connection
4. Run migration unless `--skip-migrate`
5. Optionally run bootstrap event creation

### Dependencies

- `conf.yaml` (or default lookup config)
- reachable PostgreSQL (`db.postgres_uri`)

### Side Effects

- migration writes/updates schema objects
- bootstrap inserts events and generates new relay keypair values in memory for event signing
- idempotent mode checks marker tag `nrserver-bootstrap:<canonical_url>` before inserting

### When to Use

- initial environment setup
- schema refresh in controlled environments
- explicit relay bootstrap generation

### When Not to Use

- production runtime during peak load without maintenance window
- repeated bootstrap generation without intentional lifecycle plan

### Common Errors

- invalid option combination:
  - `--skip-migrate` without `--bootstrap`
- timeout too low for migration window
- DB connectivity/auth failures

## `import`

### Purpose

Import Nostr events from JSONL into local database.

### Flags

| Flag | Description | Default |
|---|---|---|
| `--file`, `-f` | source JSONL file | `events.jsonl` |
| `--batch-size`, `-b` | batch size (`0` = line mode) | `100` |
| `--num-workers`, `-w` | parallel import workers | `2` |
| `--stats-interval` | periodic pool/progress logs (`0` disables) | `5s` |
| `--fail-on-error` | return non-zero when row errors occur | `false` |

### Execution Notes

- only `.jsonl` is supported
- line mode can be useful for low-memory troubleshooting
- batch mode improves throughput for larger files

### Common Errors

- unsupported extension (`.json`, `.csv`, etc.)
- malformed JSON lines or invalid signatures
- file I/O permission errors

## `export`

### Purpose

Export local database events to portable file formats.

### Flags

| Flag | Description | Default |
|---|---|---|
| `--file`, `-f` | destination output file | `export-TIMESTAMP.jsonl` |
| `--format` | output format: `jsonl` or `tsv` | `jsonl` |
| `--filter` | optional Nostr filter JSON object | empty |
| `--filter-file` | optional path to JSON file with filter object | empty |
| `--limit` | max exported events (`0` = no limit) | `0` |
| `--segment-size` | events per file segment (`0` = disabled) | `0` |
| `--no-header` | disable TSV header emission | `false` |
| `--overwrite` | allow replacing existing output files | `false` |
| `--batch-size`, `-b` | DB fetch size per batch | `100` |
| `--writer-workers`, `-w` | parallel encoder workers | `2` |

### Format Notes

- `jsonl`: one event per line
- `tsv`: tab-separated records with header
  - columns: `id`, `pubkey`, `created_at`, `kind`, `tags`, `content`, `sig`
  - `tags` is encoded as JSON string in the TSV cell

Segmentation behavior:

- when `--segment-size` is enabled, output rotates after N events
- filenames include a 3-digit index suffix:
  - `export-<timestamp>-001.jsonl`
  - `export-<timestamp>-002.jsonl`
  - `export-<timestamp>-001.tsv`

Filter input behavior:

- use exactly one filter source: `--filter` or `--filter-file`
- `--filter-file` must contain a single JSON object compatible with Nostr filter

### Common Errors

- invalid `--format`
- output path permission issues
- DB connectivity or query failures

## `cron`

### Purpose

Run maintenance jobs configured in `cron.*` either as a long-running scheduler or one-shot execution.

### Flags

| Flag | Description | Default |
|---|---|---|
| `--list` | list jobs with status and schedule | `false` |
| `--run-once` | execute selected enabled jobs once and exit | `false` |
| `--job` | filter by job name | all jobs |
| `--timeout` | per-job timeout | `30m` |

Supported job names:

- `db_optimization`
- `reported_events_fetch`
- `delete_old_events`
- `nip40`

### Execution Modes

- scheduler mode (default): registers enabled jobs and waits for shutdown signal
- one-shot mode (`--run-once`): executes selected enabled jobs once, sequentially
- listing mode (`--list`): prints available jobs and exits

### Selection Rules

- if `--job` is not provided, all configured jobs are considered
- only enabled jobs are runnable
- unknown job names fail fast with explicit error

### Operational Notes

- cron expressions are 6-field format (seconds precision)
- each job executes with timeout guard (`--timeout`)
- scheduler handles SIGINT/SIGTERM and performs graceful stop

### Common Errors

- unknown job name in `--job`
- no enabled jobs for selected filter
- invalid flag combination (`--list` + `--run-once`)

## `conf`

### Purpose

Provide configuration lifecycle commands for operators and developers.

### Command Tree

- `conf print` (`show` alias): print default template
- `conf effective`: print effective loaded config
- `conf validate`: validate config semantics
- `conf write`: write default template file

### Shared Behavior

- output format supported where relevant: `yaml` or `json`
- file-target commands accept `--file` when applicable

### `conf print`

Use for default template discovery and onboarding.

### `conf effective`

Loads config as runtime does and prints resolved structure:

- default search paths when `--file` omitted
- explicit file when `--file` is provided

### `conf validate`

Validates:

- required DB URI
- relay information checks
- cron expressions for enabled jobs
- `reported_events_fetch.relays` when that job is enabled

### `conf write`

Writes default config template to target path with explicit overwrite control (`--force`).

### Common Errors

- invalid output format
- destination file exists and `--force` not provided
- invalid cron expression for enabled job
- missing required DB URI

## Architectural Notes

The CLI refactor separates concerns by command domain:

- `cmd/internal/seed` for seed parsing + execution
- `cmd/internal/cron` for cron options, job registry and runtime orchestration
- `cmd/internal/conf` for config output, write and validation workflows
- `cmd/internal/import` for import options, validation and execution modes
- `cmd/internal/export` for export options, format encoding and streaming pipeline

This keeps command entrypoints (`cmd/*.go`) focused on Cobra wiring and UX while runtime logic remains testable and cohesive.

## Maintenance Impact

Benefits:

- clearer operational surface area
- predictable error handling and flag validation
- reduced logic concentration in top-level Cobra files
- easier extension for new cron jobs and config checks

Tradeoffs:

- additional internal files/packages for command behavior
- stricter validation may reject previously ambiguous invocations
