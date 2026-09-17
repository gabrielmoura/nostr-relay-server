# NIP-50 Search Plan

## Status

| Phase | Status | Dependency | Acceptance criterion |
|---|---|---|---|
| Parser and query construction | Complete | None | Safe `simple` tsquery and relevance ordering |
| Weighted search migration | Complete | Parser/query tests | Existing and new events index content and descriptions |
| Validation and documentation | Complete | Migration and unit tests | NIP-11 advertises only implemented behavior |

## Scope

This plan is limited to NIP-50. It excludes generic SQL tuning, batching, REQ multi-filter execution, and NIP-45 aggregate semantics.

## Database rollout

The local inspected database is PostgreSQL 16.11 with approximately 893k events and a historical `content_search` expression using `portuguese`. PostgreSQL 16 cannot replace a stored generated-column expression in place. The migration will create the description extraction function, replace `content_search`, rebuild its GIN index, and run `ANALYZE event`.

This takes an `ACCESS EXCLUSIVE` lock while the generated column is replaced and can rewrite the table. Schedule a maintenance window for large databases. The down migration restores the previous `simple` content-only vector and GIN index. Do not run migrations while the database has a collation-version mismatch; that operational warning must be remediated independently.

## Known limitation

Recognized NIP-50 extensions are intentionally parsed but ignored. They are not advertised as supported filters.

## Validation performed

- `go test -count=1 ./infra/db ./config ./infra/db/helper`
- `go test ./infra/db/helper -run '^$' -bench BenchmarkParseSearchQuery -benchmem -count=5`

The parser benchmark measured 1.59–1.75 µs/op, 600 B/op, and 9 allocs/op on the development CPU. There is no pre-change parser baseline because this component did not exist before this delivery; no comparative performance claim is made.
- The live PostgreSQL inspection confirmed PostgreSQL 16.11, migration version 2, the historical `portuguese` generated vector, and a GIN `content_search_idx`. The NIP-50 migration was not applied to that database.
