# Marmot MIP-00 Integration Plan

## Objective

Add explicit, optional and highly configurable relay-side support for Marmot `MIP-00` so the project can safely handle Marmot `KeyPackage` and relay-list events without changing baseline relay behavior when disabled.

This plan covers only the relay responsibilities required to support `MIP-00` event flow. It does **not** turn the relay into a full MLS implementation.

## Scope Boundary

### In Scope

- accept, validate, store and query Marmot `kind:30443` KeyPackage events
- accept, validate, store and query Marmot `kind:10051` relay-list events
- optionally accept legacy Marmot `kind:443` KeyPackage events behind an explicit toggle
- preserve NIP-01 addressable replacement semantics for `kind:30443` via `(kind, pubkey, d)`
- expose configuration flags that keep the feature disabled by default
- document exact compatibility level and validation modes

### Out of Scope

- creating MLS credentials or KeyPackages
- decrypting Welcome messages
- parsing or executing MLS group operations from `MIP-01`, `MIP-02` or `MIP-03`
- maintaining client-side `init_key` lifecycle
- enforcing NIP-65 inbox/outbox behavior beyond normal event storage/query capabilities

## Protocol Summary

Based on `ref/marmot/00.md`, the relay-facing pieces of `MIP-00` are:

- `kind:30443` is the canonical KeyPackage event and is addressable
- `kind:10051` advertises relays where a user's KeyPackages are published
- `kind:30443` uses `d` for slot identity and `i` for KeyPackageRef lookup
- the relay must allow queries by author and by `#d` / `#i`
- legacy `kind:443` interoperability is optional and should stay behind an explicit toggle

## Kind Coverage Matrix

This section distinguishes three states:

- **explicitly supported now**: covered by dedicated config and/or validation in this project
- **generic relay support only**: accepted by the normal Nostr pipeline, but without Marmot-specific validation or workflow guarantees
- **out of scope for this phase**: not yet treated as a Marmot feature in project docs or code

| Kind | Purpose | Current project status |
|---|---|---|
| `443` | Legacy KeyPackage | **Explicitly supported now**, but only when `marmot.mip00.accept_legacy_kind_443=true`. Validation is basic and relay-side only. |
| `444` | Welcome | **Generic relay support only**. The relay can store and forward arbitrary events, but does not implement MIP-02 or Welcome-specific guarantees. |
| `445` | Group Event | **Generic relay support only**. The relay can store and query arbitrary events, but does not implement MIP-03 group semantics, ephemeral key requirements, or encrypted payload validation. |
| `447` | Token Request | **Out of scope for this phase**. No dedicated MIP-05 handling exists. |
| `448` | Token List Response | **Out of scope for this phase**. No dedicated MIP-05 handling exists. |
| `449` | Token Removal | **Out of scope for this phase**. No dedicated MIP-05 handling exists. |
| `10050` | Relay List for notifications | **Generic relay support only**. The relay already handles this as a normal Nostr event kind, but there is no Marmot/MIP-05-specific validation or workflow. |
| `10051` | KeyPackage Relay List | **Explicitly supported now** through `marmot.mip00` basic validation and normal replaceable event semantics. |

## What "Implemented" Means Here

For this project phase, a kind is considered explicitly implemented only when at least one of the following is true:

- it has dedicated feature flags
- it has dedicated validation rules in `internal/policies`
- it has dedicated tests proving the intended relay behavior
- it has dedicated observability or compatibility notes in project docs

By that stricter definition, this phase intentionally covers only:

- `30443`
- `10051`
- optional legacy `443`

Everything else in the Marmot kind list remains either generic relay behavior or future work.

## Current Project Fit

The current relay already provides useful building blocks:

- generic event storage in the shared `event` table
- addressable replacement semantics in ingestion
- tag filtering through the generated `tagvalues` column
- NIP-09 deletion support for explicit decommissioning flows

This means the first implementation can stay small and focused on config plus policy validation instead of introducing a new storage model.

## Compatibility Target

### Phase 1 Compatibility

The first project milestone should provide **relay-aware compatibility**, meaning:

- the relay explicitly recognizes Marmot `MIP-00` event kinds
- the relay validates required tags and basic shape
- the relay preserves Nostr replacement and query semantics expected by Marmot clients
- the relay does **not** parse MLS payloads or verify inner MLS cryptographic structures

### Future Strict Compatibility

A later phase may add optional strict validation that:

- base64-decodes KeyPackage content
- parses the MLS KeyPackage or KeyPackageBundle representation
- recomputes the `KeyPackageRef` for the `i` tag
- verifies the MLS BasicCredential identity matches the Nostr pubkey

That later phase requires a production-suitable MLS dependency for Go and should remain explicitly optional.

## Configuration Contract

Add a new top-level config block:

```yaml
marmot:
  enabled: false
  mip00:
    enabled: false
    accept_kind_30443: true
    accept_kind_10051: true
    accept_legacy_kind_443: false
    validation_mode: basic
    require_i_tag: true
    require_base64_encoding_tag: true
    require_relays_tag: true
    require_mls_extensions: true
    require_mls_proposals: true
    require_ws_relay_urls: true
    max_relays_per_event: 10
    max_content_size_bytes: 262144
    advertise_in_relay_document: false
```

### Semantics

- `marmot.enabled`: master switch for all future Marmot work in this relay
- `marmot.mip00.enabled`: enables only relay-side `MIP-00` handling
- `accept_kind_30443`: allows KeyPackage events when the module is enabled
- `accept_kind_10051`: allows relay-list events when the module is enabled
- `accept_legacy_kind_443`: allows compatibility with pre-cutover KeyPackage events
- `validation_mode`:
  - `off`: no Marmot-specific validation beyond generic relay rules
  - `basic`: validate Nostr-level tags and shape only
  - `strict`: reserved for future MLS-aware validation
- `advertise_in_relay_document`: if true, append a non-standard informational marker in project-controlled documentation or operator-facing metadata, but do not treat MIP support as a NIP

## Validation Rules

### `kind:30443` in `basic` mode

The relay should reject the event when any enabled requirement below fails:

- missing or empty `d` tag
- missing `encoding` tag
- `encoding` tag value different from `base64`
- missing `mls_protocol_version` tag
- missing `mls_ciphersuite` tag
- missing `i` tag when `require_i_tag=true`
- invalid hex in the `i` tag
- missing `relays` tag when `require_relays_tag=true`
- `relays` tag with zero relay URLs
- relay URL that is not `ws://` or `wss://` when `require_ws_relay_urls=true`
- more than `max_relays_per_event` URLs in a single `relays` tag
- content larger than `max_content_size_bytes`
- missing `mls_extensions` when `require_mls_extensions=true`
- `mls_extensions` missing `0xf2ee` or `0x000a` when required
- missing `mls_proposals` when `require_mls_proposals=true`
- `mls_proposals` missing `0x000a` when required

### `kind:10051` in `basic` mode

The relay should reject the event when any enabled requirement below fails:

- no `relay` tags present
- relay URL that is not `ws://` or `wss://` when `require_ws_relay_urls=true`
- more than `max_relays_per_event` relay tags

### `kind:443`

Legacy `kind:443` support should remain disabled by default.

If enabled later:

- it must bypass the `d` requirement
- it may accept missing `i` and fall back to content-derived lookup client-side
- it should reuse the same content/tag validation rules where applicable

## Relay Behavior

### Disabled Mode

When `marmot.enabled=false` or `marmot.mip00.enabled=false`:

- no extra startup work runs
- no Marmot-specific validation runs
- no Marmot-specific metrics are emitted
- generic relay behavior remains unchanged

### Enabled Mode

When enabled:

- `EVENT` flow applies Marmot `MIP-00` validation only to configured Marmot kinds
- `REQ` and `COUNT` continue using the normal query pipeline
- no custom storage tables are introduced
- no relay-generated events are emitted

## Storage and Query Model

No new PostgreSQL tables are required for `MIP-00` phase 1.

The shared `event` table remains authoritative for:

- `kind:30443`
- `kind:10051`
- optional legacy `kind:443`

### Required Query Shapes

The relay must keep supporting these normal Nostr filters:

- `{"kinds":[30443],"authors":["<pubkey>"]}`
- `{"kinds":[30443],"#d":["<slot>"]}`
- `{"kinds":[30443],"#i":["<key_package_ref>"]}`
- `{"kinds":[10051],"authors":["<pubkey>"]}`

This is already compatible with the current SQL/tag model because `d` and `i` are single-character tags.

## Integration Points

### Config

- `config/conf-data.go`: add typed `MarmotConfig` and nested `MarmotMIP00Config`
- `config/defaults.go`: set conservative defaults with the feature disabled
- `config/config.go`: append no supported NIP because MIP support is not a NIP advertisement
- config validation: reject invalid `validation_mode`, negative limits or contradictory toggles

### Policies

- add a focused Marmot `MIP-00` event validator under `internal/policies`
- only run it for `kind:30443`, `kind:10051`, and optional `kind:443` when the feature is enabled
- keep generic event validation order intact

### Ingestion

- reuse existing replaceable and addressable handling
- do not add Marmot-specific post-persist side effects in phase 1

### Metrics

Recommended counters:

- `nostr_marmot_mip00_events_total{kind,result}`
- `nostr_marmot_mip00_rejections_total{kind,reason}`

Metrics should only be registered and used if the project wants first-class operational visibility for the module.

## Error Contract

Rejection messages should stay deterministic and operator-actionable, for example:

- `invalid: marmot mip00 keypackage missing d tag`
- `invalid: marmot mip00 keypackage encoding must be base64`
- `invalid: marmot mip00 keypackage missing required i tag`
- `invalid: marmot mip00 relay list contains invalid relay url`

The exact wording may evolve, but the reason should always identify:

- feature scope: `marmot mip00`
- event type: `keypackage` or `relay list`
- failure cause: missing tag, invalid value, invalid URL, oversize payload

## Risks

- accepting malformed `30443` events without explicit validation weakens interoperability claims
- attempting strict MLS parsing too early can introduce fragile dependencies into the relay hot path
- enabling legacy `kind:443` by default would broaden compatibility surface and testing cost without clear operator value
- over-advertising support could mislead clients into expecting full Marmot group support

## Recommended Delivery Order

1. config structs, defaults and validation
2. policy-level `basic` validation for `30443` and `10051`
3. tests for enabled and disabled modes
4. tests proving addressable replacement semantics for `30443`
5. optional metrics
6. optional legacy `443` support
7. optional `strict` MLS-aware validation

## Non-Goals for Phase 1

- no admin CRUD or dashboard for Marmot events
- no dedicated Redis caches
- no database migrations
- no NIP-11 `supported_nips` changes
- no claim of full Marmot compatibility beyond relay-side `MIP-00`
