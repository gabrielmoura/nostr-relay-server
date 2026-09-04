import { beforeEach, describe, expect, it, vi } from "vitest"

const apolloClient = vi.hoisted(() => ({ query: vi.fn() }))

vi.mock("@/graphql/client", () => ({ adminApolloClient: apolloClient }))

import { getPrivacyStatus } from "@/services/admin"

describe("getPrivacyStatus", () => {
  beforeEach(() => {
    apolloClient.query.mockReset()
  })

  it("drops null and primitive networks while preserving a valid network", async () => {
    apolloClient.query.mockResolvedValue({
      data: {
        privacyStatus: {
          enabled: true,
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
          persistence: false,
          stateDir: null,
          networks: "not-an-array",
        },
      },
    })

    await expect(getPrivacyStatus()).resolves.toMatchObject({ networks: [] })
  })
})
