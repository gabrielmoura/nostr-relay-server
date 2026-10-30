# RelaySync/NIP-77 boundary

RelaySync v2 is an independent protocol. It has its own capability endpoint,
HTTP session state, protobuf messages, snapshot, manifest, reconciliation,
transfer and (when durably enabled) WebSocket live state.

It does not reuse or alter `NEG-OPEN`, `NEG-MSG`, `NEG-CLOSE`, NIP-01 relay
messages, or NIP-77 session state. The existing NIP-77 implementation remains
available unchanged. The CLI may select that existing path when a peer does
not advertise RelaySync v2; this is operational fallback, not protocol
compatibility or shared state.
