# Privacy Networks (Tor, I2P, Yggdrasil)

> **Opt-in feature.** Nothing changes unless you enable `privacy.enabled: true` in `conf.yaml`.

This module exposes the relay over three privacy networks, so users can reach it
without revealing their IP address — and the operator can accept connections
without exposing a cleartext TCP port to the public internet.

## Quick Start

```yaml
# conf.yaml
privacy:
  enabled: true
  tor:
    mode: native          # starts and manages a local `tor` process from PATH
  yggdrasil:
    mode: native          # embeds a Yggdrasil mesh node (ratatoskr)
  i2p:
    mode: external        # requires a running i2pd or Java-I2P with SAM enabled
    sam_port: 7656
```

Start the relay as usual:

```bash
./nostr-relay-server server -c conf.yaml
```

On startup, the privacy manager logs the advertised addresses:

```
privacy network started  network=tor    addresses=[abc123.onion]
privacy network started  network=yggdrasil addresses=[200:db8::abcd/128]
privacy network started  network=i2p    addresses=[abcdefghijklmnop.b32.i2p]
```

These addresses are automatically added to the NIP-11 relay information
document (`/` with `Accept: application/nostr+json`) under the
`privacy_addresses` field, so clients can discover them.

## Supported Networks

### Tor (Onion Services)

| Mode | How it works | When to use |
|------|-------------|-------------|
| `native` | bine spawns `tor` from PATH, creates a v3 onion service. Default SOCKS port: 9050. | Quick dev/test; single-server relays |
| `external` | Connects to an already-running Tor daemon on SOCKS port (default 9050). Requires manual torrc or Docker setup. | Production; Docker/Kubernetes deployments |

**Default virtual port:** 80 (maps to the relay's HTTP listener on the configured port).
After a native onion descriptor is published, its active `ws://` URL is accepted
for NIP-42 authentication. Port 80 is represented without an explicit port;
other configured `remote_ports` retain their port. Only URLs from the active
process are authorized — a persisted `tor.json` diagnostic manifest never
authorizes an onion address after the service stops.

### Yggdrasil (Encrypted Mesh)

| Mode | How it works | When to use |
|------|-------------|-------------|
| `native` | Embeds a full Yggdrasil node via [ratatoskr](https://github.com/voluminor/ratatoskr). The relay becomes reachable at its Yggdrasil IPv6 address. | Peer-to-peer relays without any ISP dependency |

Yggdrasil is always embedded (there is no external daemon); `native` and `external`
both run the embedded node. The embedded node discovers peers via links and
multicast. The relay's port is forwarded from the Yggdrasil IPv6 address to
`127.0.0.1:<relay_port>`. Its NIP-42 URL is advertised only after that forward
is active, as `ws://[IPv6]:effective_port`.

### I2P (Eepsite)

| Mode | How it works | When to use |
|------|-------------|-------------|
| `external` **(default)** | Minimal SAM v3 client connects to an already-running i2pd or Java-I2P router on port 7656, creates `STREAM FORWARD SILENT=true` to the relay loopback listener, then publishes the `.b32.i2p` address. | Production; recommended path |
| `native` | Starts a managed Go I2P router plus a loopback-only SAM bridge, then creates `STREAM FORWARD SILENT=true` to the relay. No Java-I2P/i2pd process is required. | Development and self-contained deployments; validate against your target I2P network |

`native` is isolated under `i2p.data_dir` and does not alter NRServer's global
configuration. `external` remains the best interoperability choice when an
operator already runs Java-I2P or i2pd.

## Configuration Reference

```yaml
privacy:
  enabled: false           # Master switch (default: false — opt-in)
  persistence: true        # Persist/reuse identities across restarts (default: true)
  state_dir: ./data/privacy # Root for persistent identity keys (0700 on disk)

  tor:
    mode: native           # native | external | auto | disabled
    data_dir: ""           # native: persistent Tor DataDir (empty = temp)
    control_port: 0        # native: control port (0 = auto)
    socks_port: 9050       # external: SOCKS proxy port
    remote_ports: [80]     # onion virtual ports
    onion_port: 0          # local port the onion forwards to (0 = relay port)
    v3: true               # Use v3 onion services (default: true)

  i2p:
    mode: external         # native | external | auto | disabled
    sam_host: 127.0.0.1    # SAM API host
    sam_port: 7656         # SAM API port
    i2cp_port: 7654        # native: embedded router I2CP listener
    session_name: nostr-relay  # SAM session identifier
    data_dir: ""           # native: embedded router data directory

  yggdrasil:
    mode: native           # native | external | disabled
    peers: []              # static peers (currently informational; mesh discovers peers)
    data_dir: ""           # embedded node state (empty = ephemeral)
    listen_port: 0         # Yggdrasil-internal port (0 = relay port)
```

## Persistent Identity

By default the relay **reuses the same privacy identities across restarts**
(`privacy.persistence: true`). Each network's address stays stable:

| Network | Persisted artifact | Stable because |
|---------|-------------------|----------------|
| Tor | v3 onion **ed25519 key** (`tor.key`) | Same key → same `.onion` |
| Yggdrasil | node **ed25519 private key** (`ygg.key`) | Same key → same IPv6 |
| I2P | SAM **destination blob** (`i2p.key`) and router state under `i2p.data_dir` | Same destination → same `.b32.i2p` |

Keys are stored under `privacy.state_dir` (default `./data/privacy`) with
**0600 file / 0700 directory** permissions, written **atomically** (temp file +
rename) so a crash can never corrupt an existing identity.

Native Tor additionally writes a non-secret `tor.json` diagnostic manifest with
the published onion address, onion-service version, active relay URLs, virtual
ports, local target and publication time. A manifest write failure is logged but does not
change the active service or its authorization set. Active I2P and Yggdrasil
forwards similarly write non-secret `i2p.json` and `ygg.json` manifests. These
diagnostics do not authorize addresses after their live services stop.

Set `privacy.persistence: false` (or leave `state_dir` unset) to rotate
identities every run — e.g. for disposable/test relays that must not be
reachable at a known address.

### I2P note

Native I2P keeps its router state in `i2p.data_dir`; external I2P reuse requires
the SAM daemon (i2pd/Java-I2P) to accept the same `DESTINATION` blob on session
create. For the strongest external persistence guarantee, run the router with a
fixed destination configured for this relay.

## Architecture

```
                    ┌──────────────────┐
                    │  Privacy Manager │
                    │  (opt-in)        │
                    └──────┬───────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
         ┌────┴────┐ ┌────┴────┐ ┌────┴────┐
         │   Tor   │ │   I2P   │ │ Yggdrasil│
         │  bine   │ │ SAM v3  │ │ratatoskr│
         └────┬────┘ └────┬────┘ └────┬────┘
              │           │           │
              └───────────┼───────────┘
                          │
                   127.0.0.1:<port>
                          │
                    ┌─────┴─────┐
                    │   Relay   │
                    │  Fiber HTTP│
                    └───────────┘
```

Each service dials the relay's local listener (`127.0.0.1:<port>`) or sets up
a forward mapping to it. The relay itself is unaware of privacy networks — it
just listens on its configured port.

## NIP-11 Privacy Addresses

When privacy is enabled, the NIP-11 relay information document includes a
`privacy_addresses` field:

```json
{
  "name": "My Relay",
  "privacy_addresses": [
    "abc123.onion",
    "200:db8::abcd",
    "abcdefghijklmnop.b32.i2p"
  ]
}
```

Clients that support privacy networks can parse this field to discover
alternative connection URLs.

## Dependencies

| Network | Library | License |
|---------|---------|---------|
| Tor | `github.com/cretz/bine` | MIT |
| I2P | `github.com/go-i2p/go-i2p` and the maintained `go-sam-bridge` fork; external mode uses the in-repo SAM v3 client | MIT |
| Yggdrasil | `github.com/voluminor/ratatoskr` | BSD-3 |

The native path owns a Go router, its I2CP client and a loopback-only SAM bridge.
It never falls back to an operator's external SAM daemon: if its local listener,
router or inbound forward cannot start, the privacy manager applies the configured
required/fail-open policy.

## Production Checklist

- [ ] `privacy.enabled: true` in `conf.yaml`
- [ ] Tor: ensure `tor` binary is in PATH (native) or a Tor daemon is running (external)
- [ ] I2P external: ensure i2pd/Java-I2P is running with SAM enabled on port 7656; native has no external daemon prerequisite
- [ ] Yggdrasil: no external setup needed; the embedded node handles peer discovery
- [ ] Verify addresses appear in NIP-11 (`curl -H 'Accept: application/nostr+json' http://localhost:PORT/`)
- [ ] Confirm identities persist: restart the relay and check the addresses are unchanged (verify files under `privacy.state_dir`)
- [ ] Test each network independently (set `disabled` for networks you don't want)

## Troubleshooting

**Tor native fails to start:** Ensure `tor` is installed and accessible in PATH.
Check `tor --version` works.

**I2P external connection refused:** Ensure i2pd or Java-I2P is running with SAM
enabled. Check `nc -z 127.0.0.1 7656` succeeds. SAM must be enabled in the router
config (`i2pd.conf` or `router.config`).

**Yggdrasil no peers:** The embedded node discovers peers via multicast on local
networks and static peers in config. For internet peers, configure `yggdrasil.peers`
or rely on known peering endpoints.

**No privacy_addresses in NIP-11:** Ensure `privacy.enabled: true` and check relay
logs for startup errors from the privacy manager.
