# NIP Support Matrix

This file is the canonical source for protocol-support claims. `supported_nips` in the NIP-11 response is derived from enabled relay capabilities; it must not be used to advertise client-only behavior or arbitrary event kinds.

## Relay capabilities advertised through NIP-11

| NIP | Relay behavior | Advertisement rule |
|---|---|---|
| 01 | WebSocket protocol, event validation, `REQ`, `EVENT`, and `CLOSE` | Always |
| 09 | Author deletion events | Always |
| 11 | Relay information document | Always |
| 13 | Minimum proof-of-work validation | `relay.minimum_pow_limit > 0` |
| 29 | Relay-managed groups | `nip29.enabled=true` |
| 40 | Expiration-tag validation and cleanup | Always |
| 42 | WebSocket authentication | An auth mode is enabled |
| 45 | `COUNT` command | Always |
| 50 | Full-text `search` filter with relevance ordering | Always |
| 62 | Request to vanish | `relay.vanish_event=true` |
| 70 | Protected-event rules | `nip70.enabled=true` |
| 77 | Negentropy synchronization | `enable_negentropy=true` |
| 86 | Relay management JSON-RPC | `nip86.enabled=true` |
| 96 | HTTP file-storage information and endpoints | `store.enabled=true` |
| 98 | HTTP authorization used by NIP-86 | `nip86.enabled=true` |

NIP-29 additionally exposes `nip29: {"subgroups": true}`. NIP-11 fields are omitted when no real relay policy backs them. In particular, `max_event_tags` is an advertisement; the enforced value remains `security.limits.max_event_tags`.

NIP-50 searches `content` (weight A) and `description` tag values (weight B) with PostgreSQL's `simple` text-search configuration. Results with text terms are ordered by `ts_rank_cd`, then `created_at` and `id`; `limit` is applied after that ordering. The relay recognizes `domain:`, `language:`, `sentiment:`, `nsfw:`, and `include:` extensions case-insensitively, removes them from the free-text expression, and currently ignores them rather than claiming SQL filtering. A search with no remaining terms deliberately returns no results.

## Accepted event formats without relay-specific semantics

The relay stores and serves ordinary Nostr events generically. This is not a claim of dedicated relay logic and these client-side NIPs are not added automatically to `supported_nips`:

| NIP | Correct meaning |
|---|---|
| 02 | Follow lists |
| 04 | Encrypted direct messages |
| 17 | Private direct messages |
| 18 | Reposts |
| 25 | Reactions |
| 32 | Labels; the admin API can create and query kind `1985` label events |

## File storage and Blossom

NIP-96 is the HTTP file-storage protocol advertised at `/.well-known/nostr/nip96.json`. Blossom/BUD endpoints and authorization are related storage functionality but are separate specifications; neither name is a substitute for the other.

## NIP-11 interoperability

The root HTTP endpoint returns the document only for `Accept: application/nostr+json`, with that media type and the required CORS headers. The document supports the standard metadata and limitations plus operator-configured fields recognized by Amethyst: `privacy_policy`, `retention`, `nip50`, `supported_nip_extensions`, and `supported_grasps`. These extensions are metadata only and must not be used to claim unsupported behavior.
