# Configuration Conflicts and Precedence

## Objective

Document configuration pairs that are mutually exclusive, overlapping, or effectively make one another redundant in the current relay implementation.

## `ws.auth` vs `ws.auth_mode`

### What each one does

- `ws.auth`: legacy boolean switch kept for compatibility.
- `ws.auth_mode`: canonical authentication mode selector with `none`, `optional`, `flexible`, or `strict`.

### What happens together

- If `ws.auth_mode` is set to a valid value, it takes precedence.
- `ws.auth=true` only forces `strict` when `ws.auth_mode` is empty or invalid.

### Which one wins

- `ws.auth_mode` wins.

### Recommendation

- Prefer `ws.auth_mode` only.
- Use `ws.auth` only for legacy configs that have not yet been migrated.

## `relay.whitelist_kinds` vs `relay.blacklist_kinds`

### What each one does

- `relay.whitelist_kinds`: allows only the listed kinds.
- `relay.blacklist_kinds`: rejects the listed kinds.

### What happens together

- Whitelist is checked first.
- If a kind is not in the whitelist, blacklist is never consulted for that event.
- If a kind is in both lists, the whitelist passes it to the next check and the blacklist then rejects it.

### Which one wins

- Effective result is the stricter combination; overlap ends in rejection.

### Recommendation

- Use only one of them for a given deployment policy.
- Use whitelist for highly curated relays; use blacklist for mostly-open relays.

## `relay.enable_anonymous_req` vs `ws.auth_mode=strict`

### What each one does

- `relay.enable_anonymous_req`: relaxes some request-side checks for unauthenticated users.
- `ws.auth_mode=strict`: requires NIP-42 authentication before REQ is accepted.

### What happens together

- `strict` authentication rejects unauthenticated REQ before anonymous-request flexibility matters.

### Which one wins

- `ws.auth_mode=strict` wins on the REQ entry path.

### Recommendation

- For private/member relays, use `strict` and ignore `enable_anonymous_req`.
- For public read relays, keep `auth_mode=none` or `optional` if anonymous REQ should remain available.

## `relay_information.limitation.auth_required` vs `ws.auth_mode`

### What each one does

- `relay_information.limitation.auth_required`: advertises a requirement in NIP-11.
- `ws.auth_mode`: actually enforces auth in runtime behavior.

### What happens together

- They can diverge if operators set the NIP-11 field manually.
- Runtime behavior still follows `ws.auth_mode`.

### Which one wins

- Runtime enforcement is controlled by `ws.auth_mode`.

### Recommendation

- Keep the advertised NIP-11 limitation aligned with the effective `ws.auth_mode` to avoid client confusion.

## `nip29.enabled=false` vs any other `nip29.*` option

### What each one does

- `nip29.enabled`: master switch for the groups module.
- Other `nip29.*` options tune group creation, moderation, admission, cache, invites, PoW and timeline behavior.

### What happens together

- When `nip29.enabled=false`, all other `nip29.*` settings are inert at runtime.

### Which one wins

- `nip29.enabled=false` wins.

### Recommendation

- Treat `nip29.enabled` as the feature flag.
- Only tune nested NIP-29 options in environments where the module is intentionally active.

## `nip29.enabled=true` vs missing `relay_information.priv_key`

### What each one does

- `nip29.enabled=true`: activates the groups module.
- `relay_information.priv_key`: provides the relay signing key used for relay-generated `39000`-`39003` events.

### What happens together

- If NIP-29 is enabled without a relay private key, startup fails.

### Which one wins

- The runtime aborts initialization; there is no safe fallback.

### Recommendation

- Always configure a valid relay private key before enabling NIP-29 in production.

## `nip29.admission.default_restricted=false` vs `nip29.admission.require_membership_for_write=true`

### What each one does

- `default_restricted=false`: new groups are not marked as write-restricted by default.
- `require_membership_for_write=true`: write validation still requires membership on group content events.

### What happens together

- The global admission rule makes the practical behavior restrictive even when the per-group default says unrestricted.

### Which one wins

- `require_membership_for_write=true` wins on the effective write policy.

### Recommendation

- For truly open group posting, set both `default_restricted=false` and `require_membership_for_write=false`.
- For member-write groups, keep `require_membership_for_write=true` and consider `default_restricted=true` to make the stored group policy match runtime behavior.

## `nip29.admission.default_private=true` vs `nip29.moderation.allow_private_groups=false`

### What each one does

- `default_private=true`: new groups start private by default.
- `allow_private_groups=false`: moderation policy is intended to disallow private groups.

### What happens together

- The defaults pull in opposite directions and can create operator confusion about whether private groups are allowed by policy.

### Which one wins

- Behavior is policy-inconsistent; the safe operational assumption is that this combination is misconfigured.

### Recommendation

- If private groups are not allowed, keep `default_private=false`.
- If private groups are a supported product mode, keep `allow_private_groups=true`.

## `nip29.invite.enabled=false` vs `nip29.permissions.create_invite=true`

### What each one does

- `nip29.invite.enabled`: enables invite-code lifecycle support.
- `nip29.permissions.create_invite`: exposes invite creation as an allowed permission in the role model.

### What happens together

- Roles may advertise or carry invite permission while the invite subsystem itself is disabled.

### Which one wins

- `nip29.invite.enabled=false` wins operationally; invite flow stays unavailable.

### Recommendation

- When invites are disabled, also disable invite permission exposure for clarity.

## `nip29.timeline.enabled=false` vs `nip29.timeline.required_on_moderation=true`

### What each one does

- `nip29.timeline.enabled`: turns timeline reference enforcement on.
- `required_on_moderation=true`: makes moderation require `previous` references.

### What happens together

- If timeline enforcement is disabled globally, `required_on_moderation` has no runtime effect.

### Which one wins

- `nip29.timeline.enabled=false` wins.

### Recommendation

- Only tune `required_on_moderation` and `min_references` when timeline validation is enabled.

## `nip29.pow.enabled=false` vs non-zero PoW thresholds

### What each one does

- `nip29.pow.enabled`: master switch for group-specific PoW checks.
- `default_min_difficulty` and `moderation_min_difficulty`: define the required difficulty.

### What happens together

- Non-zero thresholds remain dormant while PoW is disabled.

### Which one wins

- `nip29.pow.enabled=false` wins.

### Recommendation

- Keep thresholds at zero when PoW is off, or enable PoW explicitly where abuse pressure justifies it.

## `nip29.advanced.emit_member_list_events=false` / `emit_role_events=false` vs clients expecting relay-generated state

### What each one does

- `emit_member_list_events`: controls relay emission of `39002` updates.
- `emit_role_events`: controls relay emission of `39003` updates.

### What happens together

- Group state still exists in PostgreSQL, but clients that depend on those relay-generated events may observe incomplete metadata/state propagation.

### Which one wins

- The disabled emission flag wins for outbound state visibility.

### Recommendation

- Leave both enabled for full NIP-29 client interoperability.
- Disable only in constrained environments where state fan-out cost matters more than client completeness.
