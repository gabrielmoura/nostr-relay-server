# RelaySync

RelaySync is an experimental, transport-agnostic relay-to-relay synchronization
library. This package currently provides only its NIP-01 event domain model,
bounded contracts and in-memory test stores. It does not expose network routes,
Negentropy reconciliation, compression, WebSocket support or NRServer wiring.

Consumers must provide durable implementations of the snapshot, writer,
journal, checkpoint and session contracts before enabling synchronization.

The generated protobuf package is `proto/relaysync/v2`; regenerate it from the
repository root with `make generate-relaysync`. The target checks the expected
`protoc` major/minor line and formats the generated Go source.

The package deliberately does not configure TLS or HTTP/2. Those concerns stay
with the caller's router or reverse proxy, as required by the transport
contract.
