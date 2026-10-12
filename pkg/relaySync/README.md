# RelaySync

RelaySync is an experimental, transport-agnostic relay-to-relay synchronization
library. This package currently provides its NIP-01 event domain model, bounded
contracts, independent Negentropy V1 primitives, zstd/framing codecs, in-memory
test stores, and an optional `net/http` capability discovery adapter. Session
orchestration, full event-transfer endpoints, live replay, Fiber wiring and
NRServer integration are not enabled yet.

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
