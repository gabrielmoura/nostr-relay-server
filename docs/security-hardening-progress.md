# Security Hardening Progress

## Current Goal

Introduce an incremental security layer for the relay without rewriting the existing pipeline. The current stage focuses on configurable whitelist, request and event limits, standardized Nostr rejection reasons, protocol integrity metrics, per-IP connection control, and Redis-backed progressive defense hooks.

## Technical Decisions

- Reuse the existing global initialization pattern instead of introducing a new container or service graph.
- Keep the policy hub as the main validation seam and add a focused `internal/security` package for new hardening concerns.
- Preserve the current handler flow (`ws -> req/event -> policies -> db/ingestion`) and add security checks at the smallest viable extension points.
- Keep Redis progressive defense optional and fail-soft when Redis is disabled.
- Publish new security metrics in a Prometheus-compatible shape without coupling the rest of the code to Prometheus APIs.

## Planned / Applied Extension Points

- `config/*`: add central security configuration structs and defaults.
- `infra/handler/ws/connection.go`: enforce configurable max message length and per-IP connection limit.
- `infra/handler/event/service.go`: inject per-connection whitelist and progressive defense context before policy validation.
- `internal/policies/hub.go`: enforce standardized reasons, request shaping, and event payload limits.
- `infra/metrics/security.go`: add protocol integrity and hardening counters.

## Files Changed In This Stage

- `config/conf-data.go`
- `config/config.go`
- `config/security.go`
- `cmd/server.go`
- `internal/dto/ws.go`
- `internal/policies/hub.go`
- `internal/security/context.go`
- `internal/security/defense.go`
- `internal/security/errors.go`
- `internal/security/service.go`
- `internal/security/whitelist.go`
- `infra/handler/count/count.go`
- `infra/handler/event/service.go`
- `infra/handler/http/root.go`
- `infra/handler/req/req.go`
- `infra/handler/ws/connection.go`
- `infra/metrics/security.go`

## Pending Validation

- Verify all rejection reasons remain protocol-compatible for current clients.
- Validate that connection rejection writes do not race with websocket shutdown on Fiber internals.
- Confirm Redis progressive defense thresholds against production traffic patterns before enabling them.
- Decide whether COUNT should emit a protocol envelope instead of a NOTICE-style string on rejection.

## Recommended Next Steps

1. Run formatting and focused tests for config, policies, handlers, and security package.
2. Update public configuration documentation with the new `security` block and operational examples.
3. Add a follow-up pass for Prometheus dashboards / alerts using the new counters.
4. Add targeted unit tests for whitelist normalization, request shaping, and defense escalation.

## Risks

- The project still relies on global singletons, so integration remains simple but limits isolation in tests.
- Existing metrics registration is split between legacy and new collectors; future cleanup should consolidate registration paths.
- Manual bans and whitelist semantics need product confirmation for REQ flows, because current implementation preserves manual REQ bans while bypassing publication restrictions for whitelisted authors.
