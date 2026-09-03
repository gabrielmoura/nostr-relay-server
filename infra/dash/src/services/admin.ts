import type { PrivacyStatus, RelayOverview, StreamStatus } from "@/types/admin"
import { adminApolloClient } from "@/graphql/client"
import { AdminOverviewDocument, AdminStreamStatusDocument, PrivacyStatusDocument } from "@/graphql/generated/operations"
import { formatCount } from "@/lib/utils"
import { env } from "@/lib/env"
import { mockConnections, seedBannedUsers } from "@/mocks/admin"

export class ApiError extends Error {
  status?: number
  details?: unknown
  requestId?: string

  constructor(message: string, status?: number, details?: unknown, requestId?: string, options?: { cause?: unknown }) {
    super(message, options)
    this.name = "ApiError"
    this.status = status
    this.details = details
    this.requestId = requestId
  }
}

export type PageParams = {
  limit: number
  offset: number
}

function isMockEnabled() {
  return env.mockOnFailure
}

export async function getRelayOverview(): Promise<RelayOverview> {
  try {
    const result = await adminApolloClient.query({ query: AdminOverviewDocument })
    const payload = result.data?.adminOverview
    if (!payload) throw new ApiError("GraphQL query returned no data")

    return {
      activeConnections: payload.activeConnections,
      authedConnections: payload.authedConnections,
      bannedUsers: payload.bannedUsers,
      status: payload.relayStatus === "operational" ? "operational" : "degraded",
      cards: [
        { label: "Conexoes ativas", value: formatCount(payload.activeConnections) },
        { label: "Conexoes logadas", value: formatCount(payload.authedConnections), tone: "success" },
        { label: "Usuarios banidos", value: formatCount(payload.bannedUsers), tone: "danger" },
        { label: "Eventos indexados", value: formatCount(payload.indexedEvents) },
        { label: "Eventos / min", value: formatCount(payload.eventsPerMinute) },
        { label: "Status do relay", value: payload.relayStatus, tone: payload.relayStatus === "operational" ? "success" : "warning" },
      ],
    }
  } catch (error) {
    if (!isMockEnabled()) {
      throw error
    }

    return {
      activeConnections: mockConnections.length,
      authedConnections: mockConnections.filter((connection) => Boolean(connection.authed)).length,
      bannedUsers: seedBannedUsers.length,
      status: "operational",
      cards: [
        { label: "Conexoes ativas", value: formatCount(mockConnections.length) },
        { label: "Conexoes logadas", value: formatCount(mockConnections.filter((connection) => Boolean(connection.authed)).length), tone: "success" },
        { label: "Usuarios banidos", value: formatCount(seedBannedUsers.length), tone: "danger" },
        { label: "Eventos indexados", value: "42,3M" },
        { label: "Eventos / min", value: "2.418" },
        { label: "Status do relay", value: "operational", tone: "success" },
      ],
    }
  }
}

export async function getStreamStatus(): Promise<StreamStatus> {
  try {
    const result = await adminApolloClient.query({ query: AdminStreamStatusDocument })
    const payload = result.data?.adminStreamStatus
    if (!payload) throw new ApiError("GraphQL query returned no data")
    return {
      config: payload.config as StreamStatus["config"],
      dispatcher: payload.dispatcher as StreamStatus["dispatcher"],
      pool: payload.pool as StreamStatus["pool"],
      counters: payload.counters as StreamStatus["counters"],
    }
  } catch (error) {
    if (!isMockEnabled()) {
      throw error
    }

    return {
      config: {
        stream_up: true,
        stream_down: false,
        relays: ["wss://relay.damus.io", "wss://nos.lol"],
      },
      dispatcher: {
        started: true,
        worker_count: 2,
        event_queue_len: 0,
        event_queue_cap: 1024,
        request_queue_len: 1,
        request_queue_cap: 256,
        dropped_event_jobs: 0,
        dropped_request_jobs: 0,
      },
      pool: {
        initialized: true,
        connected_relays: 1,
        total_relays: 2,
        relays: [
          { url: "wss://relay.damus.io", connected: true, failure_count: 0 },
          { url: "wss://nos.lol", connected: false, failure_count: 2, last_error: "dial timeout" },
        ],
      },
      counters: {
        forwarded_events: 124,
        forwarded_requests: 38,
        forward_failures: 3,
      },
    }
  }
}

export function getPrivacyStatus(): Promise<PrivacyStatus> {
  return getPrivacyStatusImpl().catch((error: unknown) => {
    if (!isMockEnabled()) {
      throw error
    }
    return {
      enabled: true,
      persistence: true,
      state_dir: "data/privacy",
      networks: [
        {
          id: "tor",
          name: "Tor",
          mode: "onion",
          enabled: true,
          started: true,
          status: "operational",
          addresses: ["abcdefghijklmnop.onion"],
          metrics: { tx_bytes: 1048576, rx_bytes: 2097152, connections: 3 },
          uptime_ms: 86400000,
        },
        {
          id: "yggdrasil",
          name: "Yggdrasil",
          mode: "tun",
          enabled: true,
          started: true,
          status: "operational",
          addresses: ["201:dead:beef::1"],
          metrics: { tx_bytes: 512000, rx_bytes: 1024000, peers: 5, connections: 0 },
          uptime_ms: 3600000,
        },
      ],
    }
  })
}

async function getPrivacyStatusImpl(): Promise<PrivacyStatus> {
  const result = await adminApolloClient.query({ query: PrivacyStatusDocument })
  const payload = result.data?.privacyStatus
  if (!payload) throw new ApiError("GraphQL query returned no data")
  return {
    enabled: payload.enabled,
    persistence: payload.persistence,
    state_dir: payload.stateDir ?? undefined,
    networks: (payload.networks ?? []).map((network) => ({
      id: network.id,
      name: network.name,
      mode: network.mode,
      enabled: network.enabled,
      started: network.started,
      status: network.status,
      addresses: network.addresses ?? [],
      metrics: network.metrics
        ? {
            tx_bytes: network.metrics.txBytes,
            rx_bytes: network.metrics.rxBytes,
            peers: network.metrics.peers ?? null,
            connections: network.metrics.connections ?? null,
          }
        : null,
      error: network.error ?? null,
      uptime_ms: network.uptimeMs,
    })),
  }
}

export function isFeatureDisabledError(error: any): boolean {
  if (!error) return false
  const status = error.response?.status
  const message = error.response?.data?.error || ""
  return (status === 404 || status === 503) && (message.includes("disabled") || message.includes("not enabled"))
}
