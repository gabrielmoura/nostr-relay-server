# RelaySync manifest performance

The finite HTTP cycle remains the historical reconciliation path. The server
now emits at most eight partition summaries per manifest response, so a large
relay can make progress without waiting for one response containing every
partition.

Migration `000007_add_relaysync_partition_index` adds a partial index over the
30-day bucket, one-byte event-ID prefix, and canonical ordering columns. The
partial predicate excludes malformed legacy IDs without modifying those rows.
It matches the snapshot adapter's visibility predicate and is applied to each
database independently when the normal migration runner starts.

The live WebSocket endpoint is intentionally not substituted for historical
reconciliation. It requires a durable journal projector, checkpoint handling,
and a client-side change applier for UPSERT/DELETE and ACK durability. The
current CLI sync operation is finite and therefore uses HTTP for the snapshot
and transfer phases. WebSocket live sync remains separately gated by
`relay_sync.live_enabled`.

Starting the server does not require a separate `sync` service command. The
`server` command exposes `/sync/v2/*`; `sync` is a one-shot client operation.
