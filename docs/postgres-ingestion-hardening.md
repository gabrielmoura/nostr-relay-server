## PostgreSQL Ingestion Hardening

### Context

Runtime ingestion is failing with:

```text
ERROR: index row requires 13920 bytes, maximum size is 8191 (SQLSTATE 54000)
```

The failure happens on batched event writes through:

- `infra/db.(*Queries).InsertEventBatch`
- `infra/ingestion.insertBatch`
- `infra/ingestion.(*worker).flush`

### Problems Found

#### 1. Oversized PostgreSQL index tuples on `event`

Current schema defines two concurrent B-tree indexes with wide `INCLUDE` payloads:

```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_covering_author
ON public.event (pubkey, created_at DESC)
INCLUDE (kind, content, tags, sig);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_covering
ON public.event (pubkey, created_at DESC)
INCLUDE (kind, content, tags, sig);
```

These indexes duplicate each other and include unbounded payload columns:

- `content` (`text`)
- `tags` (`jsonb`)
- `sig` (`text`)

This is the most probable root cause of `index row requires ... maximum size is 8191` during insert.

#### 2. SQL placeholder style is ambiguous in helper-generated queries

`infra/db/helper/sql_builder.go` currently builds SQL with `?` and then rewrites it with `sqlx.Rebind(...)`.

Although the final SQL becomes PostgreSQL-compatible, the intermediate query shape is confusing during review and makes runtime diagnosis harder when searching for invalid placeholder usage such as `WHERE id IN (?)`.

#### 3. Batch insert error logs are too weak for production diagnosis

Current batch failure logging only reports batch size and duration. The inner database error is wrapped as `failed to insert event <index>` without structured event metadata.

#### 4. NIP-42 failures are hard to debug

Current auth failure logging emits `evt.String()` and loses the exact rejection reason.

Important observation: `go-nostr/nip42.ValidateAuthEvent` already normalizes trailing `/`, but it still requires exact match for:

- scheme: `ws` vs `wss`
- host
- path: `/` vs `/relay`

The relay websocket endpoint is mounted on `/`, while the current config default still advertises `ws://localhost:<port>/relay`.

#### 5. Embedded schema migration splitting is fragile

`infra/db/db.go` splits `schema.sql` on semicolons but only understands single quotes. It does not treat dollar-quoted PostgreSQL bodies (`$$ ... $$`) as atomic blocks.

This is unsafe because `schema.sql` contains:

- `CREATE FUNCTION ... $$ ... $$;`
- `DO $$ ... $$;`

### Production Impact

#### Mandatory fixes

1. Inserts of large Nostr events can fail even when the table itself accepts the row.
2. Ingestion workers can keep failing on the same schema condition until operators manually change indexes.
3. NIP-42 incidents are difficult to diagnose due to missing structured reasons.
4. Schema evolution through embedded SQL is not reliable enough for index maintenance.

#### Medium-term risks

1. Current `tagvalues` filtering is fast, but it does not preserve tag-name specificity as accurately as `tags @> ...` on `jsonb`.
2. Batch insert path does not currently distinguish database duplicates from newly persisted rows when `ON CONFLICT DO NOTHING` applies.
3. Duplicate or oversized indexes increase write amplification and disk usage.

### Recommended Corrections

#### Immediate

1. Drop the two wide covering indexes concurrently.
2. Replace them with a narrow index on `(pubkey, created_at DESC)`.
3. Keep B-tree indexes only on small, selective columns.
4. Emit native PostgreSQL placeholders directly from the SQL builder.
5. Add structured insert/auth logs with event and SQLSTATE metadata.
6. Fix the default `relay_information.canonical_url` to match the real websocket route `/`.
7. Make embedded schema splitting understand dollar-quoted blocks.

#### Short-term tag strategy

Keep the current storage shape:

```sql
tags jsonb NOT NULL
```

Use exact containment queries on `tags` for Nostr filters and back them with GIN `jsonb_path_ops`. Keep `tagvalues` only as a compatibility/generated helper while avoiding new B-tree indexes over `tags`, `content`, raw payloads, or `tags::text`.

#### Medium-term tag strategy

Maintain the dedicated GIN index for exact Nostr tag containment queries:

```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_tags_gin
ON public.event
USING gin (tags jsonb_path_ops);
```

Use `jsonb_path_ops` when the hot path is mostly containment:

- `tags @> '[["p","..."]]'::jsonb`
- `tags @> '[["e","..."]]'::jsonb`
- `tags @> '[["t","..."]]'::jsonb`

Prefer `jsonb_ops` only if the workload later depends on broader operators beyond simple containment.

#### Long-term tag architecture for high volume

If the relay grows into heavy analytics, ranking, or hot tag filtering, add a normalized helper table:

```sql
event_tag (
    event_id text not null,
    tag_name text not null,
    tag_value text not null,
    kind integer not null,
    pubkey text not null,
    created_at integer not null
)
```

Use it only for hot analytical paths. Keep the canonical Nostr event in `event.tags jsonb`.

### Safe Index Set For Current Query Shapes

Current query code justifies this event index set:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS ididx
ON public.event USING btree (id text_pattern_ops);

CREATE INDEX IF NOT EXISTS pubkeyprefix
ON public.event USING btree (pubkey text_pattern_ops);

CREATE INDEX IF NOT EXISTS timeidx
ON public.event (created_at DESC);

CREATE INDEX IF NOT EXISTS kindidx
ON public.event (kind);

CREATE INDEX IF NOT EXISTS kindtimeidx
ON public.event (kind, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_pubkey_created_at
ON public.event (pubkey, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_author_kind
ON public.event (pubkey, kind, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_deletions
ON public.event (created_at DESC, id)
WHERE deleted_by IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_event_created_at_id
ON public.event (created_at, id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_tags_gin
ON public.event USING gin (tags jsonb_path_ops);

CREATE INDEX IF NOT EXISTS content_search_idx
ON public.event USING gin (content_search);
```

Notes:

- `idx_event_covering_author` and `idx_event_covering` must be removed.
- `idx_event_recent` can stay if the operator explicitly wants a time-window partial index, but it is optional and should be reviewed periodically.
- `tagvalues` may remain as a generated helper column, but it should not be the primary filter path for exact `#p`, `#e`, and `#t` semantics.

### Files To Change

- `docs/data-schema.md`
- `docs/configuration.md`
- `docs/decisions.md`
- `docs/todo.md`
- `infra/db/schema.sql`
- `infra/db/db.go`
- `infra/db/helper/sql_builder.go`
- `infra/db/helper/common_test.go`
- `infra/db/event_write_query.go`
- `infra/ingestion/ingestion.go`
- `infra/handler/auth/auth.go`
- `config/defaults.go`

### Mandatory Patches

1. remove wide covering indexes from bootstrap schema
2. add safe replacement author-time index
3. harden SQL builder to emit native `$n` placeholders
4. add structured batch insert error propagation
5. add structured NIP-42 auth failure reasons
6. align default canonical websocket URL with `/`
7. harden embedded schema statement splitting for PostgreSQL dollar-quoted bodies

### Optional Improvements

1. move tag filtering from `tagvalues` overlap to `tags @>` with a GIN `jsonb_path_ops` index
2. add per-item duplicate accounting in batch insert path
3. add a dead-letter or quarantine mechanism for repeatedly failing events
4. add integration tests with a real PostgreSQL instance for large `content` and `tags`
5. normalize hot tags into `event_tag` for analytics-heavy workloads

### Safe Execution Order

1. update documentation
2. fix schema bootstrap and migration splitter
3. drop dangerous indexes and create the narrow replacement index concurrently
4. fix SQL builder placeholder generation
5. improve insert/auth logs
6. run focused unit tests
7. validate large-event inserts against PostgreSQL
