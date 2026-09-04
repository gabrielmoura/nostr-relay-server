# Prometheus Metrics for Privacy Networks — Technical Proposal

**Status:** proposal only — no implementation in this document  
**Scope:** Tor, I2P, and Yggdrasil privacy-network services under `infra/net/privacy`  
**Decision principle:** export only facts that the current service lifecycle or a dependency's public API exposes cleanly. Do not modify dependencies, scrape daemon-specific admin panels, or add interception/proxy layers merely to manufacture unavailable traffic metrics.

## 1. Current integration surface

The relay already has a Prometheus integration:

- `github.com/prometheus/client_golang v1.23.2` is a direct dependency.
- `infra/metrics/` defines the project's collectors and registers them with the default registry through `RegisterMetrics()` / `prometheus.MustRegister`.
- `cmd/server.go` invokes `metrics.RegisterMetrics()` during startup.
- `infra/net/router.go` already exposes the default registry through `GET /metrics` using `promhttp.Handler()`.
- Existing metric names use the `nostr_` prefix; `grafana.json` is an existing dashboard, but it currently has no privacy panels.

Therefore, the work is an additive collector integration into an existing endpoint, not a second registry, endpoint, or Prometheus subsystem.

## 2. Reality map: native library capabilities

### 2.1 Tor — `github.com/cretz/bine v0.2.0`

The service uses `tor.Start`, `(*tor.Tor).Listen`, and `*tor.OnionService`.

| Public API / service fact | Naturally available | Suitable metric use |
|---|---:|---|
| `Start` success/failure | Yes, from current service lifecycle | health / availability gauge |
| `OnionService.ID` / advertised onion address | Yes | address-present gauge; log only the address, not a metric label |
| Time since successful `Start` | Yes, maintained by the application | uptime gauge |
| `(*tor.Tor).EnableNetwork`, `Close` | Yes | lifecycle state only |
| `(*tor.OnionService).Accept`, `Addr`, `Network`, `Close` | Yes | listener exists / lifecycle only |
| Circuits, relays, peers, bootstrap progress | **No high-level bine metrics API** | not proposed |
| TX/RX bytes | **No** | not proposed |

`*tor.Tor` has a public `Control *control.Conn` field, but using Tor-control commands to query implementation-specific statistics is outside bine's high-level metrics API and creates a daemon/protocol coupling. It is deliberately out of scope for this proposal.

**Conclusion:** Tor supports lifecycle-derived health, uptime, and address-availability metrics only. No traffic, circuit, or peer metric should be invented.

**Source / confidence:** HIGH — pinned source in module cache: `github.com/cretz/bine@v0.2.0/tor/{tor.go,dialer.go,listen.go}`; upstream: <https://github.com/cretz/bine/tree/v0.2.0/tor>.

### 2.2 I2P — current supported implementation is SAM v3, not `go-i2p`

The premise mentions `go-i2p`, but this repository intentionally does **not** depend on it. `go.mod` contains no `go-i2p` module. The supported path is the local minimal SAM v3 client in `infra/net/privacy/sam.go`, connected to an external i2pd or Java-I2P router. `i2p.go` explicitly states that the embedded `go-i2p` route is experimental and not wired for eepsite serving.

| Public service/client fact | Naturally available | Suitable metric use |
|---|---:|---|
| TCP SAM connect + HELLO + SESSION CREATE success/failure | Yes | health / availability gauge |
| B32 address derived from the destination | Yes | address-present gauge; never use destination/address as a label |
| Time since successful session creation | Yes, maintained by the application | uptime gauge |
| Session close | Yes | lifecycle state only |
| Router peer count / router uptime / tunnel count | **No** from the supported SAM client | not proposed |
| TX/RX bytes | **No** | not proposed |

Querying i2pd's web console or adding a second daemon-specific protocol client would not be an integration with the current library/service API. It is outside this clean initial scope.

**Conclusion:** I2P supports lifecycle-derived health, uptime, and address-availability metrics only.

**Source / confidence:** HIGH — repository sources `infra/net/privacy/{i2p.go,sam.go}`; SAM protocol interaction used by this application is limited to HELLO and SESSION CREATE. The absence of `go-i2p` is verified in `go.mod`.

### 2.3 Yggdrasil — `github.com/voluminor/ratatoskr v1.1.0` + `yggdrasil-go v0.5.14`

Yggdrasil is materially different: the in-process `*ratatoskr.Obj` exposes native public observability APIs. `Obj.Core()` returns the public `ratatoskr/mod/core.Interface`.

| Public API | Native facts available | Initial metric suitability |
|---|---|---|
| `Obj.Address`, `Subnet`, `PublicKey`, `MTU` | identity / network capability | address-present, MTU gauges |
| `Obj.GetPeers()` | `PeerInfo`: `Up`, `Inbound`, `RXBytes`, `TXBytes`, `RXRate`, `TXRate`, `Uptime`, `Latency`, plus routing metadata | peer count/state, aggregate rates; current-snapshot bytes/latency are feasible |
| `Obj.Core().GetSessions()` | `SessionInfo`: `RXBytes`, `TXBytes`, `Uptime` | active-session count and current aggregate byte snapshots |
| `Obj.Core().GetSelf()` | `SelfInfo.RoutingEntries` | routing-entry gauge |
| `Obj.Core().GetTree()` / `GetPaths()` | topology entries | topology count gauges, if operationally useful |
| `PeerManagerActive()` | managed peer URI list | managed peer count gauge |

Yggdrasil traffic values require semantic care:

- `PeerInfo` and `SessionInfo` both expose byte counters. They must **not** be summed together because they describe different views and can double-count the same traffic.
- These counters belong to currently live peers/sessions and can disappear or reset after reconnect/restart. They are not automatically safe Prometheus `Counter`s.
- The safe first export is a **Gauge** for the aggregate current session snapshot, for example `nostr_privacy_yggdrasil_session_rx_bytes` and `..._tx_bytes`.
- A future process-lifetime Prometheus Counter is possible only if a dedicated exporter defines and tests reset/reconnect semantics. It must be derived from one canonical source (recommended: sessions), never by a traffic-interception proxy.

**Conclusion:** Yggdrasil can cleanly export real active peer/session state, current aggregate traffic snapshots, current rates, latency aggregates, routing entries, MTU, and lifecycle health.

**Source / confidence:** HIGH — pinned sources in module cache: `github.com/voluminor/ratatoskr@v1.1.0/{ratatoskr.go,mod/core/interface.go}` and `github.com/yggdrasil-network/yggdrasil-go@v0.5.14/src/core/api.go`; upstream: <https://github.com/voluminor/ratatoskr/tree/v1.1.0> and <https://github.com/yggdrasil-network/yggdrasil-go/tree/v0.5.14>.

## 3. Proposed metric contract

All labels are intentionally low-cardinality. Do **not** label metrics with onion addresses, I2P destinations/B32 addresses, public keys, peer URIs, paths, or error strings.

### 3.1 Common service lifecycle metrics

| Metric | Type | Labels | Semantics |
|---|---|---|---|
| `nostr_privacy_network_up` | Gauge | `network`, `mode` | `1` only after the service's `Start` succeeded; otherwise `0`. |
| `nostr_privacy_network_uptime_seconds` | Gauge | `network`, `mode` | Seconds since the current successful start; `0` while down. |
| `nostr_privacy_network_address_configured` | Gauge | `network`, `mode` | `1` when an advertised address is available; does not expose it as a label. |
| `nostr_privacy_network_start_failures_total` | Counter | `network`, `mode` | Increment once for each failed `Start` attempt observed by the service lifecycle. |

These are supported for Tor, I2P, and Yggdrasil because they come from the application's existing `Service.Status()` contract, not fabricated dependency metrics.

### 3.2 Yggdrasil-only native metrics

| Metric | Type | Labels | Native source / semantics |
|---|---|---|---|
| `nostr_privacy_yggdrasil_peers` | Gauge | `state` (`up` / `down`) | Count `GetPeers()` entries by `PeerInfo.Up`. |
| `nostr_privacy_yggdrasil_peers_inbound` | Gauge | none | Count live peer records with `Inbound=true`. |
| `nostr_privacy_yggdrasil_sessions` | Gauge | none | `len(Core().GetSessions())`. |
| `nostr_privacy_yggdrasil_session_rx_bytes` | Gauge | none | Sum current `SessionInfo.RXBytes`; a current snapshot, not a process counter. |
| `nostr_privacy_yggdrasil_session_tx_bytes` | Gauge | none | Sum current `SessionInfo.TXBytes`; a current snapshot, not a process counter. |
| `nostr_privacy_yggdrasil_peer_rx_bytes_per_second` | Gauge | none | Sum current `PeerInfo.RXRate`. |
| `nostr_privacy_yggdrasil_peer_tx_bytes_per_second` | Gauge | none | Sum current `PeerInfo.TXRate`. |
| `nostr_privacy_yggdrasil_routing_entries` | Gauge | none | `Core().GetSelf().RoutingEntries`. |
| `nostr_privacy_yggdrasil_tree_entries` | Gauge | none | `len(Core().GetTree())`; optional first release metric. |
| `nostr_privacy_yggdrasil_path_entries` | Gauge | none | `len(Core().GetPaths())`; optional first release metric. |
| `nostr_privacy_yggdrasil_mtu_bytes` | Gauge | none | `Obj.MTU()`. |

`PeerInfo.Latency` and `PeerInfo.Uptime` are genuinely available but should be deferred unless there is a defined aggregation policy. Per-peer labels would expose identifiers and create unbounded cardinality. If added later, export bounded aggregates (e.g. max/mean) rather than raw per-peer series.

## 4. Clean architecture and registration proposal

### Boundary

The privacy domain must remain Prometheus-free:

- `infra/net/privacy` continues to own service lifecycle and a read-only status/inspection surface.
- `infra/metrics` owns Prometheus types, naming, collectors, and registration.
- A collector/adaptor reads the privacy service snapshot and, only for a Yggdrasil service, invokes the existing public `ratatoskr.Obj` / `Core()` APIs.
- The command/bootstrap layer wires the running privacy manager into the collector after it is constructed. No global Prometheus calls belong in the service implementations.

This preserves a one-way dependency: observability adapts the domain; the domain does not know Prometheus exists.

### Registration

Use the existing default registry and `/metrics` endpoint. Add a dedicated registration unit (for example, `RegisterPrivacyMetrics`) rather than changing router behavior or creating another HTTP listener. Since `cmd/server.go` currently registers base metrics before the privacy manager is constructed, privacy registration belongs after manager construction or receives a nil-safe provider that reports all services down until the manager is available.

Registration must be idempotency-safe in tests (consistent with the project's existing registration style) and must not use `MustRegister` in a path that can execute repeatedly.

### Collection model

Prefer a custom Prometheus collector or scrape-time callback over a background polling goroutine:

1. the scrape invokes the collector;
2. the collector takes a bounded, read-only snapshot;
3. it emits gauges/counters from that snapshot;
4. it does not change network configuration, dial peers, or create sessions.

This makes collection passive, avoids lifecycle races, and keeps metrics fresh without adding a second control loop.

## 5. Explicit non-goals

This proposal does **not** include:

- modifying `bine`, `ratatoskr`, `yggdrasil-go`, `go-i2p`, Tor, i2pd, or Yggdrasil;
- wrapping listeners/connections or adding interception proxies to count bytes;
- scraping Tor Control, i2pd admin/web APIs, or daemon logs for undocumented statistics;
- exporting addresses, destinations, public keys, peer URIs, topology paths, or raw errors as Prometheus labels;
- claiming Tor or I2P traffic/peer metrics that their supported integration does not provide;
- implementing the collector, tests, dashboards, or configuration changes at this stage.

## 6. Approval gate

If approved, implementation should be limited to the contract above, beginning with common lifecycle metrics and Yggdrasil's native public API metrics. Tor and I2P remain intentionally limited to lifecycle/address availability. Any later request for Tor/I2P daemon-specific metrics or process-lifetime Yggdrasil byte counters requires a separate design review.
