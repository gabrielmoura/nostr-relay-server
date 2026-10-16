# RelaySync v2 interoperability specification

Status: experimental, private protocol.

RelaySync v2 synchronizes a bounded, policy-defined set of Nostr events
between relays.  This document defines the interoperable behavior exposed by
the RelaySync v2 protobuf schema and HTTP adapters in this repository.  Field
and message names below are protobuf names from
`pkg/relaySync/proto/relaysync/v2/relaysync.proto`; their wire encoding is
protobuf binary, not JSON.

## 1. Scope and non-goals

RelaySync v2 provides capability discovery, finite snapshot synchronization,
partitioned reconciliation, event transfer, and an optional live journal
transport.  A synchronization result is meaningful only for the agreed
`Scope` and the immutable snapshot used by a session.

It is independent from NIP-77.  In particular, it does not change or reuse
NIP-77 `NEG-OPEN`, `NEG-MSG`, `NEG-CLOSE`, `NEG-ERR`, NIP-01 relay messages,
NIP-77 WebSocket state, or their wire representation.  A client may fall back
to existing NIP-77 when RelaySync v2 is unavailable, but that is operational
fallback rather than a shared protocol state.

This version does not define a public relay-discovery standard, a universal
credential scheme, policy equivalence across relays, or convergence outside
the negotiated scope.  Implementations must not infer any of these properties
from a matching event count or manifest digest alone.

## 2. Capability discovery

`GET /sync/v2/capabilities` returns a `Capabilities` protobuf message.  The
endpoint accepts `application/x-protobuf`, `application/*`, or `*/*` through
`Accept`; an absent `Accept` is accepted.  It responds with
`Content-Type: application/x-protobuf`, uses `Vary: Accept`, and returns 406
when no supported representation is acceptable.

Before opening a session, a client must verify that `Capabilities.protocol_versions`
contains `2`, select a mutually supported `Compression`, and respect the
advertised `Limits`.  `Capabilities.scopes` and `Capabilities.live_sync` are
capability hints: authorization and feature gates remain authoritative at the
requested operation.  `implementation` is informational only.

## 3. Encoding, HTTP bodies, and frames

Every defined HTTP request body and successful response body is a serialized
protobuf message with `Content-Type: application/x-protobuf`; clients should
send `Accept: application/x-protobuf`.  Error responses produced by session
errors also use `ErrorResponse` with that media type.  Clients must reject a
successful response with another media type.

HTTP payload size is bounded by `Limits.max_batch_bytes`.  A peer must also
enforce the applicable `Event` and batch limits from `Limits`; it must not
treat an advertised zero value as permission for an unbounded input.

`Compression` negotiation is represented in `Capabilities` and
`OpenSessionRequest.compressions`.  `COMPRESSION_NONE` is available in the
current session response.  An HTTP client may advertise `Accept-Encoding:
zstd` and may accept a `Content-Encoding: zstd` response only when it has a
Zstandard decoder configured.  The decompressed representation remains
subject to the response limit.

The reusable binary frame format is `RS`, byte `2`, a flags byte, a four-byte
big-endian original-size field, and the payload.  Only flag bit 0 (zstd) is
defined.  The declared original size and decoded payload must not exceed the
frame limit; an uncompressed frame's declared size must equal its payload
length.  The live adapter currently sends raw protobuf WebSocket binary
messages rather than applying this framing layer.

## 4. Authentication and admission boundary

The protocol requires the host to resolve each HTTP request to a stable
`Principal` and authorize the requested `Action` for its `Scope`.  Credential
type is intentionally outside this specification.  Deployments may use an
external authentication mechanism before this resolver.

The NRServer integration currently uses an IP/CIDR allowlist.  Its principal
is the peer source address (or the host-provided `X-RelaySync-Effective-IP`),
and an empty allowlist prevents runtime construction.  This is an integration
admission policy, not a portable identity protocol.  Reverse proxies must
only provide the effective-IP header from a trusted boundary.

`OpenSessionResponse.resume_token` is opaque and must be protected as a
secret.  Clients encode it as base64url without padding in `Authorization:
Bearer <token>` for session completion and deletion.  Current HTTP clients
also send it for manifest, reconciliation, pull, and push; current server
adapters enforce it for completion/deletion, while authorization for the
other finite operations is re-evaluated from the resolved principal and scope.
Interoperating implementations must therefore enforce the authentication
contract they advertise and must not rely on URL secrecy or session IDs as
credentials.

## 5. Finite session lifecycle

The finite HTTP routes registered by the adapters are:

| Operation | Method and path | Request | Response |
| --- | --- | --- | --- |
| open | `POST /sync/v2/sessions` | `OpenSessionRequest` | `OpenSessionResponse` |
| next manifest page | `POST /sync/v2/sessions/{base64url session ID}/manifest:next` | `ManifestRequest` | `Manifest` |
| reconcile | `POST /sync/v2/sessions/{base64url session ID}/reconcile` | `ReconcileRequest` | `ReconcileResponse` |
| pull | `POST /sync/v2/sessions/{base64url session ID}/events:pull` | `PullEventsRequest` | `PullEventsResponse` |
| push | `POST /sync/v2/sessions/{base64url session ID}/events:push` | `PushEventsRequest` | `PushEventsResponse` |
| complete | `POST /sync/v2/sessions/{base64url session ID}/complete` | `CompleteSessionRequest` | `CompleteSessionResponse` |
| delete/cancel | `DELETE /sync/v2/sessions/{base64url session ID}` | none | no body |

`OpenSessionRequest.protocol_version` must be 2 and it must include `scope`.
An implementation authorizes `ActionOpenSession`, negotiates limits and
compression, creates an opaque 16-byte `session_id`, and returns an opaque
32-byte `resume_token`.  When snapshots are available, the response includes
`Snapshot`; its `version`, `watermark`, and expiry bind the finite cycle.

`idempotency_key` makes an open retry safe only for the same principal, scope,
and identical deterministic `OpenSessionRequest`.  Reuse with different
request content is a conflict.  Sessions expire at their advertised or
configured TTL.  `CompleteSession` and deletion release the server's finite
session/snapshot state; a peer must not resume an expired or completed cycle.

## 6. Snapshots, manifests, and partition identity

A snapshot is immutable for the finite cycle.  Clients must retain and compare
the `Manifest.snapshot_version` and `Manifest.watermark` with the values from
`OpenSessionResponse.snapshot`; a snapshot mismatch or expiry requires a new
session.

For algorithm version 1, `ManifestRequest.continuation` is either empty for
the first page or an unsigned 64-bit big-endian zero-based partition offset.
`Manifest.continuation` carries the next offset in the same representation.
Repeated calls paginate the stable partition set until `Manifest.complete` is
true. An empty terminal page is valid when an offset is at or beyond the final
partition. A client may request fewer partitions through `max_partitions`, but
must handle the server page limit.

`Partition` describes a deterministic identity, not a relay-local shard.  For
algorithm version 1, each event is assigned to a 30-day UTC Unix-time bucket
(with mathematical floor for pre-epoch timestamps) and a masked prefix of its
32-byte event ID.  `Partition.id` is the SHA-256 digest, domain-separated by
`nostr-relaysync/partition/v1`, of fixed-width algorithm version, bucket,
prefix bit count, and full masked prefix.  Thus fields that look equivalent
but have a different algorithm version or prefix width are not interchangeable.

`PartitionSummary.digest` is a SHA-256 digest, domain-separated by
`nostr-relaysync/metadata/v1`, over the record count followed by ordered
`(created_at as signed 64-bit big-endian representation, event_id)` entries.
The canonical order is `created_at`, then bytewise event ID; duplicate IDs or
non-canonical ordering invalidate the summary.  `event_count` aids planning
but does not change the partition identity.

## 7. Reconciliation state machine

For each manifest partition whose digest differs, a client performs bounded
Negentropy rounds:

1. Send `ReconcileRequest` with the session ID, that exact `partition_id`, a
   non-empty `negentropy_payload`, and `round` in `1..max_concurrent_rounds`.
2. The receiver rebuilds its vector from the immutable snapshot partition and
   responds with the same partition ID and round in `ReconcileResponse`.
3. Continue the Negentropy exchange until `complete` is true; then use the
   discovered missing IDs to pull or push events according to the session
   direction.

The payload for one round must not exceed `max_frame_bytes`. A peer must reject
a partition that cannot be reconciled within the negotiated bounded metadata
limits; partition subdivision is not part of v1 and requires a future manifest
algorithm version. An unknown partition, expired snapshot, malformed frame,
or incompatible scope terminates that partition/cycle and requires the
appropriate retry or new session.  `ReconcileResponse.missing_event_ids` is
defined in the schema but is not populated by the current HTTP service; clients
must rely on the Negentropy payload exchange rather than require it.

## 8. Transfer, idempotency, and receipts

`PullEventsRequest.event_ids` selects events from the immutable snapshot.
`PullEventsResponse` returns found `Event` values and records unavailable IDs
in `missing_event_ids`.  A client must treat a missing ID as a snapshot result,
not substitute a current/live read.  The current service bounds requested IDs,
but does not currently populate pull continuations or checksums; clients must
not require those optional fields for interoperability.

`PushEventsRequest.operation_id` is required and identifies one ordered batch
within a session. A conforming server that advertises retry-safe push must
atomically reserve `(session_id, operation_id)` before invoking its writer and
return the original response for identical bytes; reuse with different bytes
is a conflict. `idempotency_key` exists in the message but current push
idempotency is keyed by session ID and `operation_id`. Until a server provides
that atomic reservation, clients must treat a lost push response as ambiguous
and must not retry it automatically.

The response has exactly one `EventReceipt` per submitted event, in input
order.  Receipts must correspond to the submitted event IDs and report a
terminal per-event outcome: `WRITE_ACCEPTED`, `WRITE_ALREADY_EXISTS`,
`WRITE_REJECTED_PERMANENT`, or `WRITE_FAILED_TEMPORARY`.  A malformed event
receives `WRITE_REJECTED_PERMANENT` with code `INVALID_EVENT`; other valid
events in the batch may still be written.  A permanent rejection is visible
and terminal for that attempt, while a temporary failure or an operation-level
error has unknown/non-terminal outcome. Clients may retry it with the same
operation ID only when the server provides the atomic guarantee above. Clients
must reject a receipt list with a missing entry, wrong
length, non-corresponding ID, or unspecified status.

## 9. HTTP statuses and errors

Clients must honor `Allow` where supplied and distinguish transport failure
from a protobuf `ErrorResponse`.  Current adapters use the following mapping:

| Status | Meaning |
| --- | --- |
| 200 | successful protobuf operation |
| 204 | successful session deletion |
| 400 | malformed identifier or protobuf body |
| 401 | no principal/token where required, invalid token, or unauthorized peer |
| 404 | unknown session/partition/resource |
| 405 | unsupported HTTP method |
| 406 | unsupported `Accept` on capability discovery |
| 409 | conflicting idempotency or invalid reconciliation round/state |
| 410 | expired session or immutable snapshot |
| 413 | session-open body exceeds the adapter limit |
| 500 | internal or otherwise unmapped failure |

For mapped session errors, `ErrorResponse.code` distinguishes invalid message,
unauthorized, snapshot expired, and internal errors; `retryable` is true for
server errors in the current adapter.  The schema also reserves limit,
journal-gap, and reconcile-required codes for transports/services that can
produce them.  Clients should use `retry_after_ms` when present and otherwise
apply bounded backoff only to retryable failures.

## 10. Safety and interoperability requirements

- Validate all opaque identifiers at their protocol width: session IDs,
  partition IDs, event IDs, scope hashes, tokens, and continuations.  Do not
  log tokens or raw event payloads merely for diagnostics.
- Enforce limits before allocation, after decompression, and for each event,
  batch, and reconciliation frame.  Reject unsupported frame flags and
  decompression output above the configured maximum.
- Bind every finite read to its snapshot and every action to current
  authorization.  Do not mix data from a new snapshot into an old manifest or
  reconciliation cycle.
- Treat `Scope.name`, `Scope.version`, and `Scope.hash` as a single policy
  identity.  A relay may have different retention, visibility, or admission
  policy even when event IDs overlap.
- Make retries explicit: retain the same open idempotency key or push
  operation ID after an ambiguous failure; never generate a new identifier
  merely because a response was lost.
- Keep RelaySync resources and overload isolated from NIP-77/NIP-01 resources.

## 11. Optional live synchronization

Live synchronization is separately feature-gated and is not a prerequisite
for finite RelaySync v2 interoperability.  A peer must advertise
`Capabilities.live_sync = true` only when its durable journal, catch-up,
checkpoint, and apply prerequisites are enabled and healthy.

The current live route is `GET /sync/v2/live` upgraded to a binary WebSocket.
The first client message is a protobuf `ClientFrame` containing
`ClientHello` with protocol version 2.  The server returns `ServerFrame` /
`ServerHello`; subsequent frames carry `FlowControl`, `Ack`, `Heartbeat`, and
`DeltaBatch` as defined by the schema.  `Cursor(epoch, sequence)` is the
journal position.  Epoch change, journal gap, snapshot expiry, or a
`ReconcileRequired` frame means the peer must return to finite reconciliation;
it must not assume a live stream repairs a missed history by itself.

## 12. Conformance requirements

An interoperable implementation should provide automated tests that prove:

1. protobuf round trips and unknown/invalid input rejection for every HTTP
   request and response message used above;
2. capability media-type negotiation, exact endpoint method handling, and
   protobuf content-type validation;
3. authorization denial, allowlist boundaries (when used), expiry, completion,
   deletion, and snapshot isolation;
4. deterministic v1 partition IDs and metadata digests across independent
   implementations, including pre-epoch timestamps and tie-breaking IDs;
5. manifest paging, terminal empty pages, reconciliation bounds, and a full
   differing-partition pull/push convergence cycle;
6. lost-response retries for session opening and push, conflicting reuse of
   keys/operation IDs, and ordered receipt validation including a permanent
   invalid-event rejection beside accepted events;
7. compressed and uncompressed size limits, malformed frame rejection, and
   decompression-bomb resistance; and
8. optional live behavior only when enabled, including resume/gap recovery,
   flow control, ACK/checkpoint handling, and proof that finite HTTP sync
   remains conformant with live disabled.

## 13. Client resilience and optional orchestration

This section describes local implementation policy, not additional wire
messages. A client may expose a disabled-by-default resilience mode that
continues independent manifest partitions after a recoverable error. It must
log the phase and safe partition coordinates (bucket and masked prefix), keep
a final count of failed partitions, and return a partial/failure result rather
than claim convergence.

For NRServer, these optional client controls are deliberately additive and do
not alter an existing `relay_sync` server configuration:

```yaml
relay_sync:
  client:
    resilience:
      enabled: false
      max_attempts: 3
      initial_backoff_milliseconds: 250
      max_backoff_milliseconds: 5000
    fetch_then_subscribe:
      enabled: false
    multi_relay:
      enabled: false
      max_parallel_relays: 1
```

The current resilience implementation uses the flag to isolate recoverable
partition failures; its retry controls are validated and reserved for the
next safe retry increment. They do not cause automatic retries of an
ambiguous write.

The following remain terminal for the whole finite cycle: cancellation or
deadline, authentication or token failure, session/snapshot expiry, conflicting
idempotency, and a missing, mismatched, or otherwise invalid push receipt.
Permanent per-event rejections are terminal only for that event and must be
visible to operators. A client must not automatically retry an ambiguous push
unless the server provides atomic operation-id idempotency; it must retain the
same operation ID for any safe retry.

The Cordn-inspired `fetch_then_subscribe` and Concord-inspired `multi_relay`
strategies are local feature flags, not protocol capabilities. The former may
be enabled only once an implementation can hand off a completed finite
snapshot/catch-up to a durable live cursor. The latter requires independent
sessions, tokens, snapshots, and operation IDs for each explicitly configured
remote. Until those guarantees exist, implementations should fail closed when
such a flag is enabled instead of silently accepting a no-op configuration.

## References

- [RelaySync/NIP-77 boundary](relaysync-protocol-boundary.md)
- [RelaySync compatibility and coexistence](relaysync/compatibility.md)
- [RelaySync protobuf schema](../pkg/relaySync/proto/relaysync/v2/relaysync.proto)
