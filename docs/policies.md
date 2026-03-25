# Policies

## Overview

This document consolidates the policies currently enforced by the relay and defines the target refactor for a single policy entrypoint used by WebSocket handlers, batch ingestion, and stream forwarding.

## Current Policies Found

### Event Policies

| Policy | Current Location | Purpose |
|---|---|---|
| Event ID validation | `infra/handler/event/event.go` | Validates that `event.id` matches the serialized SHA256 |
| Signature validation | `infra/handler/event/event.go` | Verifies event signature |
| Banned user check | `internal/policies/event.go`, `internal/policies/req.go` | Rejects events from banned pubkeys |
| Maximum event size | `internal/policies/event.go` | Rejects oversized events |
| Expiration timestamp (NIP-40) | `internal/policies/event.go` | Rejects expired events |
| Minimum POW (NIP-13) | `internal/policies/event.go` | Enforces proof-of-work threshold |
| Large tag values | `internal/policies/event.go` | Rejects large indexable tags |
| Too many indexable tags | `internal/policies/event.go` | Limits single-letter tags |
| Base64 media rejection | `internal/policies/event.go` | Rejects inline media blobs in event content |
| Replaceable semantics | `infra/handler/event/event.go` | Deletes previous replaceable/addressable events before insert |
| Ephemeral semantics | `infra/handler/event/event.go` | Avoids persistence for ephemeral kinds |

### Request Policies

| Policy | Current Location | Purpose |
|---|---|---|
| Banned requester check | `internal/policies/req.go` | Blocks REQ for banned authenticated users |
| Empty filter rejection | `internal/policies/req.go` | Rejects empty subscriptions when disabled |
| Anti-sync-bot rule | `internal/policies/req.go` | Requires author constraint for certain note sync patterns |
| Protected kinds auth check | `internal/policies/req.go` | Restricts protected kinds without auth |
| Public kinds-only access | `internal/policies/req.go` | Restricts unauthenticated REQ filters when auth is enabled |

### Stream / Forwarding Rules

| Rule | Current Location | Purpose |
|---|---|---|
| Upstream forwarding kind allowlist | `infra/stream/event.go` | Limits which kinds are forwarded upstream |
| Downstream request forwarding | `infra/stream/req.go` | Pulls from relay pool when local query is empty |

## Target Refactor

### Single Policy Hub

Create a single policy hub with explicit entrypoints:

- `ValidateIncomingEvent(ctx, ws, evt)`
- `ValidateIncomingReq(ctx, ws, subID, filters)`
- `ValidateBatchEvent(ctx, evt)`
- `ShouldForwardEvent(evt)`
- `NormalizeFilter(filter)`

The hub should return structured decisions instead of mixing validation with transport concerns.

### Policy Execution Order

#### Incoming EVENT

1. Decode payload
2. Validate event ID
3. Validate signature
4. Validate banned pubkey
5. Validate event size
6. Validate expiration
7. Validate POW
8. Validate tag size/count
9. Validate content policy
10. Apply event-kind semantics (ephemeral/replaceable/addressable)
11. Enqueue for ingestion

#### Batch Ingestion

1. Deduplicate event IDs
2. Re-run storage-safe policy validation
3. Partition by semantic type
4. Execute replace/delete preparation
5. Insert valid events in batch
6. Cache and notify listeners
7. Forward eligible events upstream

#### Incoming REQ

1. Decode subscription id
2. Decode filters
3. Normalize filter limits
4. Validate requester auth policy
5. Validate non-empty filters
6. Validate anti-sync-bot rule
7. Validate protected kinds
8. Execute query / stream fallback
9. Register subscription

## Refactor Goals

- Keep transport code in `infra/handler` thin
- Move all policy decisions to one package
- Reuse the same rules in live ingest and batch ingest
- Avoid duplicate checks scattered across handlers
- Keep errors deterministic and protocol-safe

## Non-Goals

- No generic rule engine
- No DSL for policies
- No extra abstraction layers for rules that are used in only one place

## Planned Package Shape

```text
internal/policies/
  hub.go            # public entrypoints
  event_rules.go    # event-only rules
  req_rules.go      # request-only rules
  shared.go         # shared helpers, normalization, ban lookup
  decision.go       # small result structs
```

## Batch Ingestion Policy Requirements

The ingestion pipeline must enforce at least these checks before persistence:

- duplicate event id
- banned pubkey
- invalid signature or invalid id
- expired event
- insufficient POW
- invalid tag constraints
- disallowed content
- replaceable/addressable conflict resolution

## Stream Performance Refactor Requirements

`infra/stream` should:

- avoid synchronous hot-path forwarding in handlers
- use non-blocking enqueue with bounded workers
- separate upstream event forwarding from downstream REQ backfill
- deduplicate repeated forwarded requests when possible
- skip forwarding for unsupported kinds early
