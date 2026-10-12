# RelaySync

RelaySync is an experimental, transport-agnostic relay-to-relay synchronization
library. This package currently provides its NIP-01 event domain model, bounded
contracts, independent Negentropy V1 primitives, zstd/framing codecs, in-memory
test stores, `net/http` session/manifest/transfer adapters, an HTTP client, and
a transport-agnostic bounded live replay coordinator. The coordinator is
driven by the caller (it starts no listener), persists only contiguous ACKs
through `CheckpointStore`, and requires a new HTTP reconciliation after a
journal gap or epoch change. WebSocket wiring and NRServer integration remain
disabled.

Consumers must provide durable implementations of the snapshot, writer,
journal, checkpoint and session contracts before enabling synchronization.

The generated protobuf package is `proto/relaysync/v2`; regenerate it from the
repository root with `make generate-relaysync`. The target checks the expected
`protoc` major/minor line and formats the generated Go source.

The package deliberately does not configure TLS or HTTP/2. Those concerns stay
with the caller's router or reverse proxy, as required by the transport
contract.

To expose capability discovery, construct `NewHTTPAdapter` with a configured
`relaysync.v2.Capabilities` protobuf and call `Register` on the host's
`http.ServeMux`. It registers `GET /sync/v2/capabilities`, which responds with
`application/x-protobuf` when accepted by the client.

Session routes are registered separately with `RegisterSessionService` and
`RegisterTransferServices`. A successful finite cycle must call
`HTTPClient.CompleteSession` (or `SessionService.Complete`) so the server can
discard its session, snapshot and temporary reconciliation state. Checkpoints,
when configured, are independent of that lifecycle.

For the standard library, use `relaySync/nethttp.NewHandler` with an existing
`http.Server`. For Fiber v2, use `relaySync/fiber.RegisterServices` with the
existing `*fiber.App`, an `HTTPAdapter`, the optional services and a
`PrincipalResolver`. The adapters register under `/sync/v2` and never create a
second listener or configure TLS/HTTP2; those remain router/proxy concerns.

NRServer integration is disabled by default. To enable the experimental
integration, configure an explicit source-IP allowlist:

```yaml
relay_sync:
  enabled: true
  http_enabled: true
  allowed_ips: ["10.0.0.0/8"]
  apply_enabled: true
  projector_enabled: true
  journal_epoch: 1
```

The allowlist is applied before capability discovery and live WebSocket
upgrade. Session resume tokens are protocol lifecycle values, not peer
authentication credentials. PostgreSQL migrations through `000006` must be
applied before enabling the integration.
