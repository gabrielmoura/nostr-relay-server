// @vitest-environment jsdom
import { describe, expect, it, vi } from "vitest"
import { createNostrifyReadTransport } from "./relay-client"
import type { EventTemplate, ImportSigner, NostrEvent } from "./types"

const signer: ImportSigner = {
  getPublicKey: async () => "owner",
  signEvent: async (template: EventTemplate) => ({ id: "auth", pubkey: "owner", sig: "sig", ...template }),
  dispose() {},
}

const event = (id: string): NostrEvent => ({ id, pubkey: "owner", created_at: 1, kind: 4, tags: [], content: "ciphertext", sig: "sig" })

describe("Nostrify source relay transport", () => {
  it("reads raw events until EOSE and closes the relay", async () => {
    const close = vi.fn(async () => undefined)
    let authenticate: ((challenge: string) => Promise<NostrEvent>) | undefined
    const transport = createNostrifyReadTransport((_url, options) => ({
      close,
      async *req() {
        yield ["EVENT", "sub", event("one")] as const
        yield ["EVENT", "sub", event("two")] as const
        yield ["EOSE", "sub"] as const
      },
    }))
    const session = await transport.openReadSession!("wss://source.example", signer)
    await expect(session.query([{ kinds: [4] }])).resolves.toEqual([event("one"), event("two")])
    session.close()
    expect(close).toHaveBeenCalledOnce()
    expect(authenticate).toBeUndefined()
  })

  it("signs NIP-42 challenges with the source relay URL", async () => {
    let authenticate: ((challenge: string) => Promise<NostrEvent>) | undefined
    const transport = createNostrifyReadTransport((_url, options) => {
      authenticate = options.auth
      return { close: async () => undefined, async *req() { yield ["EOSE", "sub"] as const } }
    })
    const session = await transport.openReadSession!("wss://source.example", signer)
    const auth = await authenticate!("challenge")
    expect(auth.kind).toBe(22242)
    expect(auth.tags).toEqual([["relay", "wss://source.example"], ["challenge", "challenge"]])
    session.close()
  })

  it("maps a cancelled request timeout to the import timeout error", async () => {
    vi.useFakeTimers()
    const transport = createNostrifyReadTransport(() => ({
      close: async () => undefined,
      async *req(_filters, options) {
        await new Promise<void>((_resolve, reject) => options?.signal?.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")), { once: true }))
      },
    }))
    const session = await transport.openReadSession!("wss://source.example", signer)
    const result = session.query([{ kinds: [4] }])
    const expected = expect(result).rejects.toThrow("tempo de conexão esgotado")
    await vi.advanceTimersByTimeAsync(30_000)
    await expected
    session.close()
    vi.useRealTimers()
  })
})
