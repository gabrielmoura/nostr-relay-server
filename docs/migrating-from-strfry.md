# Migrating from Strfry

`nrserver migrate-strfry` imports Nostr events maintained by a local Strfry
instance into the PostgreSQL database configured for this relay. It is intended
for a controlled operational migration, not continuous replication.

## Requirements and preflight

Before running the migration:

1. Back up the target PostgreSQL database and retain the Strfry data directory.
2. Ensure the target `conf.yaml` contains the intended `db.postgres_uri` and
   passes `nrserver conf validate`.
3. Verify the target schema is current with `nrserver migrate status`.
4. Verify that the selected `strfry` executable can read the source database
   and that the account running `nrserver` can read the database directory.
5. For a consistent cutover, stop writes to the source relay (or schedule a
   maintenance window) before the final export. An export made while the source
   accepts writes is not a point-in-time replication protocol.

The target is loaded exclusively through the existing YAML/Viper configuration
flow (`conf.yaml`). Do not create or pass a target TOML configuration. The
command creates a short-lived, permission-restricted Strfry configuration only
for the source subprocess; it contains the selected source database path and is
removed when the command exits.

## Export usage

Run a validation first. It invokes Strfry and checks each event ID and
signature, but does not connect to or write the target database:

```sh
nrserver migrate-strfry \
  --strfry-db /var/lib/strfry/db \
  --strfry-bin /usr/local/bin/strfry \
  --dry-run \
  --batch-size 1000 \
  --fail-on-error
```

After reviewing the validation output, execute the import:

```sh
nrserver migrate-strfry \
  --strfry-db /var/lib/strfry/db \
  --strfry-bin /usr/local/bin/strfry \
  --batch-size 1000 \
  --fail-on-error
```

To migrate only events at or after a Unix timestamp, use `--since`:

```sh
nrserver migrate-strfry \
  --strfry-db /var/lib/strfry/db \
  --since 1725148800 \
  --batch-size 500
```

The command runs:

```text
<strfry-bin> --config <temporary-source-config> export [--since <unix>]
```

Exported JSONL is streamed with a bounded line size. Valid events are written
in batches using the existing COPY migration path; row-level database failures
fall back to the relay's batch insertion path. Existing IDs are handled by the
database conflict policy, so rerunning a completed import is safe for already
imported events. Operational persistence failures and Strfry subprocess
failures stop the command.

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `--strfry-db` | required | Path to the source Strfry database. |
| `--strfry-bin` | `strfry` | Strfry executable to invoke for `--source=export`. |
| `--source` | `export` | Source implementation: `export` or reserved `lmdb`. |
| `--since` | `0` | Unix timestamp lower bound; `0` exports all events. |
| `--batch-size` | `1000` | Valid events per target persistence batch. |
| `--dry-run` | `false` | Validate exported events without target database writes. |
| `--fail-on-error` | `false` | Return non-zero when invalid or rejected exported events are found. |

## Direct LMDB reader

`--source=lmdb` is reserved for a future direct reader and fails explicitly;
it never falls back to export. During validation against the supplied live
Strfry database, its DBI layout did not match the locally available schema-v3
reference, so treating the internal format as compatible would risk skipping or
misreading events. Use `--source=export` until a reader is verified against the
exact source layout.

## Cutover and verification

Keep the source database intact until the target relay has been verified. Check
the command summary (`read`, `valid`, `persisted`, `invalid`, and `rejected`), inspect relay
logs for rejected data, and compare representative event IDs and query results
against the source. Before directing clients to the new relay, confirm target
database health and the relay's normal startup/readiness behavior.

If writes resumed on Strfry during the initial import, repeat the migration with
an appropriate `--since` value during the cutover window. Event-ID conflict
handling avoids duplicating events already migrated.
