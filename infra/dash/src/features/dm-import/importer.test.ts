import { describe, expect, it } from "vitest"
import { discoverRelays, importDMs } from "./importer"
import type { ImportSigner, RelayTransport } from "./types"

const signer: ImportSigner = { getPublicKey: async () => "owner", signEvent: async () => { throw new Error("not used") }, dispose() {} }
const event = { id: "one", pubkey: "owner", created_at: 1, kind: 4, tags: [], content: "cipher", sig: "sig" }

describe("importDMs", () => it("deduplicates remote events and preserves them while publishing", async () => {
  const published: unknown[] = []
  const transport: RelayTransport = {
    query: async () => [event, event],
    publish: async (_url, value) => { published.push(value); return "duplicate" },
  }
  const result = await importDMs({ transport, signer, bootstrap: ["wss://relay.example"], localRelay: "ws://local" })
  expect(result.found).toBe(1); expect(result.duplicates).toBe(1); expect(published).toEqual([event])
}))

describe("DM importer protocol", () => {
  it("separates private messages from other authored events even if a relay returns extra events", async () => {
    const note = { ...event, id: "note", kind: 1, content: "public" }
    const giftwrap = { ...event, id: "gift", kind: 1059, pubkey: "other", tags: [["p", "owner"]] }
    const unrelated = { ...event, id: "unrelated", pubkey: "other" }
    for (const scope of ["dm", "events"] as const) {
      const published: string[] = []
      const transport: RelayTransport = {
        query: async (_url, filters) => "since" in filters[0]! ? [event, note, giftwrap, unrelated] : [],
        publish: async (_url, value) => { published.push(value.id); return "inserted" },
      }
      await importDMs({ transport, signer, bootstrap: ["wss://relay.example"], localRelay: "ws://local", scope })
      expect(published.sort()).toEqual(scope === "dm" ? ["gift", "one"] : ["note"])
    }
  })
  it("uses only DM filters with bounded since/until pagination", async () => {
    const calls: Record<string, unknown>[][] = []
    let request = 0
    const transport: RelayTransport = {
      query: async (_url, filters) => { calls.push(filters); request++; return request <= 2 ? [event] : [] },
      publish: async () => "inserted",
    }
    await importDMs({ transport, signer, bootstrap: ["wss://relay.example"], localRelay: "ws://local" })
    const paged = calls.flat()
    expect(paged).toEqual(expect.arrayContaining([{ kinds: [1059], "#p": ["owner"], since: 0, until: expect.any(Number), limit: 500 }, { kinds: [4], authors: ["owner"], since: 0, until: expect.any(Number), limit: 500 }, { kinds: [4], "#p": ["owner"], since: 0, until: expect.any(Number), limit: 500 }]))
  })

  it("keeps a source session from relay discovery through collection, then awaits its close", async () => {
    let opened = 0
    let closed = 0
    const source = {
      query: async (filters: Record<string, unknown>[]) => {
        const kinds = filters[0]?.kinds as number[]
        if (kinds.includes(10050)) return [{ ...event, kind: 10050, tags: [] }]
        return kinds.includes(4) ? [event] : []
      },
      publish: async () => "inserted" as const,
      close: async () => { closed++ },
    }
    const transport: RelayTransport = {
      query: async () => [],
      publish: async () => "inserted",
      openReadSession: async () => { opened++; return source },
      openSession: async () => ({ query: async () => [], publish: async () => "inserted", close: () => undefined }),
    }
    const result = await importDMs({ transport, signer, bootstrap: ["wss://bootstrap.example"], localRelay: "ws://local" })
    expect(opened).toBe(1)
    expect(closed).toBe(1)
    expect(result.imported).toBe(1)
  })

  it("stops after an external abort without opening the local publisher", async () => {
    const controller = new AbortController()
    let localOpened = false
    const source = {
      query: async (filters: Record<string, unknown>[], signal?: AbortSignal) => {
        const kinds = filters[0]?.kinds as number[]
        if (kinds.includes(10050)) return [{ ...event, kind: 10050, tags: [] }]
        if (kinds.includes(10002)) return []
        controller.abort()
        expect(signal?.aborted).toBe(true)
        throw new DOMException("aborted", "AbortError")
      },
      publish: async () => "inserted" as const,
      close: async () => undefined,
    }
    const transport: RelayTransport = {
      query: async () => [],
      publish: async () => "inserted",
      openReadSession: async () => source,
      openSession: async () => { localOpened = true; return source },
    }
    const result = await importDMs({ transport, signer, bootstrap: ["wss://bootstrap.example"], localRelay: "ws://local", signal: controller.signal })
    expect(result.phase).toBe("cancelled")
    expect(result.incomplete).toBe(true)
    expect(localOpened).toBe(false)
  })

  it("does not publish or count an uncertain result when opening the local relay is aborted", async () => {
    const controller = new AbortController()
    let published = 0
    const source = {
      query: async (filters: Record<string, unknown>[]) => {
        const kinds = filters[0]?.kinds as number[]
        if (kinds.includes(10050) || kinds.includes(10002)) return []
        return kinds.includes(4) ? [event] : []
      },
      publish: async () => "inserted" as const,
      close: async () => undefined,
    }
    const transport: RelayTransport = {
      query: async () => [],
      publish: async () => { published++; return "inserted" },
      openReadSession: async () => source,
      openSession: async () => {
        controller.abort()
        throw new DOMException("aborted", "AbortError")
      },
    }
    const result = await importDMs({ transport, signer, bootstrap: ["wss://bootstrap.example"], localRelay: "ws://local", signal: controller.signal })
    expect(result.phase).toBe("cancelled")
    expect(result.uncertain).toBe(0)
    expect(result.errors).toEqual([])
    expect(published).toBe(0)
  })

  it("continues after a failed relay and reports progress", async () => {
    const updates: number[] = []
    const transport: RelayTransport = { query: async (url) => { if (url.includes("bad")) throw new Error("offline"); return [] }, publish: async () => "inserted" }
    const result = await importDMs({ transport, signer, bootstrap: ["wss://bad.example", "wss://ok.example"], localRelay: "ws://local", onProgress: (p) => updates.push(p.completedRelays) })
    expect(result.failedRelays).toBeGreaterThan(0); expect(updates.at(-1)).toBe(result.relays)
  })

  it("discovers 10050 first, then falls back to readable NIP-65 relays", async () => {
    const calls: Record<string, unknown>[][] = []
    const transport: RelayTransport = { query: async (_url, filters) => { calls.push(filters); return calls.length === 1 ? [] : [{ ...event, kind: 10002, created_at: 9, tags: [["r", "wss://read.example"], ["r", "wss://write.example", "write"]] }] }, publish: async () => "inserted" }
    const relays = await discoverRelays(transport, signer, ["wss://bootstrap.example"])
    expect(calls[0]).toEqual([{ kinds: [10050], authors: ["owner"], limit: 1 }])
    expect(calls[1]).toEqual([{ kinds: [10002], authors: ["owner"], limit: 1 }])
    expect(relays).toContain("wss://read.example"); expect(relays).not.toContain("wss://write.example")
  })
  it("uses a present 10050 relay list without querying NIP-65", async () => {
    const calls: Record<string, unknown>[][] = []
    const transport: RelayTransport = { query: async (_url, filters) => { calls.push(filters); return [{ ...event, kind: 10050, tags: [["relay", "wss://dm.example"]] }] }, publish: async () => "inserted" }
    const relays = await discoverRelays(transport, signer, ["wss://bootstrap.example"])
    expect(calls).toHaveLength(1); expect(relays).toContain("wss://dm.example")
  })

  it("falls back to NIP-65 when the 10050 query fails or has no valid relay", async () => {
    const calls: Record<string, unknown>[][] = []
    const transport: RelayTransport = {
      query: async (_url, filters) => {
        calls.push(filters)
        if ((filters[0]?.kinds as number[]).includes(10050) && calls.length === 1) throw new Error("unsupported")
        if ((filters[0]?.kinds as number[]).includes(10050)) return [{ ...event, kind: 10050, tags: [["relay", "https://not-a-relay.example"]] }]
        return [{ ...event, kind: 10002, tags: [["r", "wss://nip65.example"]] }]
      },
      publish: async () => "inserted",
    }
    const first = await discoverRelays(transport, signer, ["wss://bootstrap.example"])
    expect(first).toContain("wss://nip65.example")
    const second = await discoverRelays(transport, signer, ["wss://bootstrap-two.example"])
    expect(second).toContain("wss://nip65.example")
  })

  it("keeps pagination cursors independent and reports an ambiguous same-second page", async () => {
    const calls: Record<string, unknown>[][] = []
    const sameSecond = { ...event, id: "same-second", created_at: Math.floor(Date.now() / 1000) }
    const transport: RelayTransport = {
      query: async (_url, filters) => {
        calls.push(filters)
        const kinds = filters[0]?.kinds as number[]
        return kinds.includes(4) ? [sameSecond] : []
      },
      publish: async () => "inserted",
    }
    const result = await importDMs({ transport, signer, bootstrap: ["wss://relay.example"], localRelay: "ws://local" })
    expect(result.incomplete).toBe(true)
    const queryFilters = calls.flat().filter((filter) => "since" in filter)
    expect(queryFilters.filter((filter) => (filter.kinds as number[]).includes(4))).toHaveLength(2)
  })

  it("bounds total collected events and marks the import incomplete", async () => {
    const many = Array.from({ length: 10_001 }, (_, index) => ({ ...event, id: `event-${index}` }))
    const transport: RelayTransport = {
      query: async (_url, filters) => (filters[0]?.kinds as number[]).includes(4) ? many : [],
      publish: async () => "inserted",
    }
    const result = await importDMs({ transport, signer, bootstrap: ["wss://relay.example"], localRelay: "ws://local" })
    expect(result.found).toBe(10_000)
    expect(result.incomplete).toBe(true)
    expect(result.imported).toBe(10_000)
  })
})
