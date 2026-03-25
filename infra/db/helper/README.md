# infra/db/helper

## Overview

This package is responsible for transforming a normalized Nostr filter into SQL fragments and query metadata used by the database layer.

It is intentionally small and focused.

The package does not execute SQL.
The package does not know about transport concerns.
The package does not read from Redis or PostgreSQL directly.

Its responsibility is limited to:

- validating filter limits against relay configuration
- normalizing filters for deterministic query generation
- building SQL and bind parameters
- generating stable filter hashes for cache keys

## Current Role in the Flow

```text
REQ / COUNT handler
    -> policies normalize and validate request intent
    -> infra/db helper normalizes filter shape
    -> helper builds SQL + params
    -> infra/db executes query
    -> cache stores/reuses result
```

## Inputs

The main input is `nostr.Filter` plus `config.RelayConfig`.

Relevant filter fields:

- `IDs`
- `Authors`
- `Kinds`
- `Tags`
- `Since`
- `Until`
- `Search`
- `Limit`

Relevant config fields:

- `QueryLimit`
- `QueryIDsLimit`
- `QueryAuthorsLimit`
- `QueryKindsLimit`
- `QueryTagsLimit`
- `FakeDeletion`

## Responsibilities

### 1. Filter normalization

Normalization exists to make query generation deterministic.

That means:

- sorting ids, authors and kinds
- sorting tag keys and values
- clamping limit to the configured maximum

This is important for:

- readable SQL generation
- deterministic testing
- stable Redis query-cache keys
- better prepared-query routing

### 2. Query validation

The package rejects invalid filter shapes before SQL generation:

- too many ids
- too many authors
- too many kinds
- too many tag values
- empty tag set

### 3. SQL generation

The package builds the final SQL in a predictable order:

1. ids
2. authors
3. kinds
4. tags
5. since/until
6. full-text search
7. fake deletion condition
8. order by
9. limit

This order matters because it improves readability and makes tests deterministic.

### 4. Filter hashing

The package generates a stable hash from:

- normalized filter
- query type (`events` vs `count`)

This hash is used by Redis query cache.

## Non-Goals

This package should not:

- contain business policies
- know anything about websocket envelopes
- maintain prepared statement definitions
- run database queries
- contain cache invalidation policy

## Refactor Goals

The refactor should improve:

- naming clarity
- deterministic behavior
- test readability
- smaller focused helper functions
- explicit separation between normalization, validation and SQL rendering

## Planned Internal Structure

```text
infra/db/helper/
  README.md
  common.go          # public orchestration entrypoints
  normalize.go       # filter normalization
  validate.go        # limit and shape validation
  sql_builder.go     # SQL fragment rendering
  hash.go            # stable filter hashing
  common_test.go     # package-level behavior tests
```

## Testing Strategy

Tests should verify:

- normalization is deterministic
- limits are enforced with clear errors
- event and count SQL are generated correctly
- fake deletion condition is applied only when enabled
- hashes are stable regardless of input order

Tests should avoid debug prints and should assert full behavior, not only partial strings where exact SQL can be validated safely.
