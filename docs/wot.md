# Web of Trust (WoT)

## Overview

The Web of Trust (WoT) feature provides a decentralized, community-driven moderation layer for the Nostr Relay Server. By defining a set of highly trusted identities (`TargetPubkey`) and following their interactions (who they follow), the relay iteratively constructs a secure trust graph within memory. This ensures the relay actively filters SPAM and malicious actors without relying on centralized arbitrary bans, honoring the social map of the instance owner.

## How it Works

The WoT subsystem analyzes specific Nostr events—especially Follow Lists (Kind 3)—originating from the configured `TargetPubkey` and configured seed relays (`SeedRelays`). It maps out an expanding web of recognized individuals using breadth-first processing up to the defined limits.

Any incoming `EVENT` must belong to a pubkey present in the active trust network map constraint. If it does not, the event is immediately discarded during the `validateStorageEvent` policy evaluation, saving resources and protecting storage.

## Calculation Mechanism

The continuous WoT graph compilation is optimized for performance without overloading the database or blocking the ingestion paths.

1.  **Initial Seeding**: During bootstrap, a background goroutine begins by fetching the `TargetPubkey`'s follow list utilizing `nostr.KindFollowList` through the `nostrpool` client querying across seed relays.
2.  **Breadth Expansion**: It enumerates all contacts within those lists ("p" tags), incrementing their "follower counts". It proceeds to iteratively fetch the follow lists for users who satisfy the follower count threshold up to a defined depth.
3.  **In-Memory Graph Construction**: Every discovered pubkey matching the minimum interactions (`MinimumFollowers`) is appended to a memory-mapped graph.
4.  **Limits**: This algorithm strictly guards against unbounded memory growth using `MaxTrustNetwork` and limits evaluation radius using `MaxOneHopNetwork`.
5.  **Periodic Refresh**: The graph is automatically cleared and rebuilt based on the `RefreshIntervalHours` configuration (default: 3 hours). This prevents stale data structures and dynamically reflects the current trusted network state without causing blocking operations for live traffic.
6.  **Concurrency Optimization**: Graph compilation and batch downloads are executed using parallel routines and contexts capped by semaphores. The pointers are swapped automatically with a new graph, causing zero downtime for the relay operations.

## Caching

To guarantee nano-second evaluations on the event validation hot-path, **the WoT subsystem relies entirely on an in-memory mapped representation of the trust graph (`map[string]bool`)**.

*   No database queries are performed per-event. 
*   No Redis lookups are performed per-event.
*   By operating at `O(1)` against the resident RAM map safely locked behind a `RWMutex` (or through atomic pointer swaps), it guarantees negligible processing delays while maintaining accurate security enforcement.

## Example Flow

```text
TargetPubkey -> Follows [A, B] -> Fetch A, B's Follows -> Expand into TrustNetworkMap[A,B,C,D...]
```

When an event arrives via WebSocket or Batch Ingest:

1.  Policy Hub decodes the `EVENT`.
2.  Delegates `wot.Validate(evt.PubKey)`.
3.  Reads `TrustNetworkMap[evt.PubKey]`.
4.  If `true`: Approved, pushes to ingestion queue.
5.  If `false`: Rejected with `"restricted: pubkey not in web of trust"`.
