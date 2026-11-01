import { beforeEach, describe, expect, it, vi } from "vitest"

const apolloClient = vi.hoisted(() => ({ query: vi.fn() }))

vi.mock("@/graphql/client", () => ({ adminApolloClient: apolloClient }))

import { getPrivacyStatus, getRelayOverview } from "@/services/admin"

describe("getPrivacyStatus", () => {
  beforeEach(() => {
    apolloClient.query.mockReset()
  })

  it("drops null and primitive networks while preserving a valid network", async () => {
    apolloClient.query.mockResolvedValue({
      data: {
        privacyStatus: {
          enabled: true,
          degraded: false,
          persistence: true,
          stateDir: "data/privacy",
          networks: [
            null,
            "tor",
            42,
            {
              id: "yggdrasil",
              name: "Yggdrasil",
              mode: "native",
              enabled: true,
              started: true,
              status: "operational",
              addresses: ["201:db8::1", null],
              metrics: { txBytes: 0, rxBytes: 0, peers: 0, connections: 0 },
              uptimeMs: 0,
            },
          ],
        },
      },
    })

    await expect(getPrivacyStatus()).resolves.toEqual({
      enabled: true,
      degraded: false,
      persistence: true,
      state_dir: "data/privacy",
      networks: [
        {
          id: "yggdrasil",
          name: "Yggdrasil",
          mode: "native",
          enabled: true,
          started: true,
          status: "operational",
          addresses: ["201:db8::1"],
          metrics: { tx_bytes: 0, rx_bytes: 0, peers: 0, connections: 0 },
          error: null,
          uptime_ms: 0,
        },
      ],
    })
  })

  it("normalizes malformed network collections to an empty list", async () => {
    apolloClient.query.mockResolvedValue({
      data: {
        privacyStatus: {
          enabled: true,
          degraded: false,
          persistence: false,
          stateDir: null,
          networks: "not-an-array",
        },
      },
    })

    await expect(getPrivacyStatus()).resolves.toMatchObject({ networks: [] })
  })
})

describe("getRelayOverview", () => {
  beforeEach(() => {
    apolloClient.query.mockReset()
  })

  it("keeps local ingestion outcomes distinct", async () => {
    apolloClient.query.mockResolvedValue({
      data: {
        adminOverview: {
          activeConnections: 5,
          authedConnections: 3,
          bannedUsers: 1,
          indexedEvents: 40,
          eventsPerMinute: 2,
          relayStatus: "operational",
          ingestion: {
            acceptedEvents: 100,
            insertedEvents: 80,
            duplicates: 12,
            rejected: 3,
            uncertain: 2,
            batchErrors: 1,
            batches: 10,
            backpressure: 4,
            queueDepth: 9,
          },
        },
      },
    })

    await expect(getRelayOverview()).resolves.toMatchObject({
      activeConnections: 5,
      ingestion: {
        accepted_events: 100,
        inserted_events: 80,
        duplicates: 12,
        rejected: 3,
        uncertain: 2,
        batch_errors: 1,
        batches: 10,
        backpressure: 4,
        queue_depth: 9,
      },
    })
  })

  it("does not replace unavailable local metrics with mock values", async () => {
    apolloClient.query.mockRejectedValue(new Error("admin unavailable"))

    await expect(getRelayOverview()).rejects.toThrow("admin unavailable")
  })
})
