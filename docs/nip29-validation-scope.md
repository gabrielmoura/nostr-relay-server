# NIP-29 Validation Scope Fix

## Objective

Correct the NIP-29 guard so the relay only applies group existence and permission checks when the message is actually inside the NIP-29 protocol scope.

## Reported Production Symptom

- With `nip29.enabled=true`, the relay rejects unrelated publications with `invalid: group does not exist`.
- The rejection happens even when the event is not a NIP-29 moderation/state event.

## Expected Behavior

### EVENT

- Accept all events by default.
- Only apply NIP-29 group validation when the event kind is explicitly part of the NIP-29 surface handled by this relay:
  - moderation and membership flow: `9000`-`9022`
  - relay-generated metadata/state flow: `39000`-`39003`
- For kinds outside that set, ignore `h`/`d` group lookup on the write path.

### REQ / COUNT

- When a filter explicitly uses `#h`, treat it as a NIP-29 group read and validate access before executing or delivering results.
- When a filter does not explicitly target a group, keep the normal relay behavior.
- Even on normal queries, private and hidden group events must still be filtered out before delivery when the requester lacks access.

## Current Root Cause

- `internal/groups/helpers.go` exposes `groupIDFromEvent()` that reads any `h` or `d` tag.
- `internal/groups/state.go` uses that helper in `isRelevantEvent()`, so many unrelated events become NIP-29 candidates.
- `internal/groups/validation.go` then performs group lookup and rejects missing groups globally.
- `internal/groups/state.go` also makes `shouldHandleFilter()` overly broad for the pre-query read gate.

## Planned Code Changes

1. Add explicit helpers for NIP-29 write kinds and NIP-29 read-filter scope.
2. Restrict `ValidateIncomingEvent()` to explicit NIP-29 kinds.
3. Restrict `ValidateFilter()` / `QueryEvents()` fast-path activation to explicit group reads.
4. Preserve `canReadEvent()` as the post-query guard for private and hidden group material.
5. Add regression tests around the reported failure mode.

## Non-Goals

- No schema changes.
- No changes to Redis or PostgreSQL data contracts.
- No change to generic relay validation unrelated to NIP-29.
