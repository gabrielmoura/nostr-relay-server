import { describe, expect, it, vi } from "vitest"

import { getRelayInformation, relayWebSocketURL } from "./public-relay"

describe("public relay service", () => {
  it("loads NIP-11 from the public root with the Nostr accept header", async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ supported_nips: [1, 11] }) })
    vi.stubGlobal("fetch", fetchMock)

    await expect(getRelayInformation()).resolves.toEqual({ supported_nips: [1, 11] })
    expect(fetchMock).toHaveBeenCalledWith("/", { headers: { Accept: "application/nostr+json" } })
  })

  it("derives a secure WebSocket URL from HTTPS", () => {
    expect(relayWebSocketURL({ protocol: "https:", host: "relay.example" } as Location)).toBe("wss://relay.example/")
  })
})
