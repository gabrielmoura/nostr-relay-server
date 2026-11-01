// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest"
import { browserRelayTransport } from "./relay-client"
import type { ImportSigner } from "./types"

let socket: any
const sent: unknown[][] = []

class FakeWebSocket {
  onopen: (() => void) | undefined; onmessage: ((event: { data: string }) => void) | undefined; onerror: (() => void) | undefined; onclose: (() => void) | undefined
  constructor() { socket = this; queueMicrotask(() => this.onopen?.()) }
  send(raw: string) { sent.push(JSON.parse(raw)) }
  close = vi.fn()
}

const signer: ImportSigner = { getPublicKey: async () => "owner", signEvent: async (template) => ({ id: "auth-id", pubkey: "owner", sig: "sig", ...template }), dispose() {} }
const event = (id = "event") => ({ id, pubkey: "owner", created_at: 1, kind: 4, tags: [], content: "x", sig: "sig" })

afterEach(() => { sent.length = 0; vi.useRealTimers() })

describe("browserRelayTransport NIP-42", () => it("retries EVENT only after AUTH OK", async () => {
  ;(globalThis as any).WebSocket = FakeWebSocket
  const promise = browserRelayTransport.publish("wss://relay.example", event(), signer)
  await new Promise((resolve) => setTimeout(resolve, 0)); expect(sent[0]?.[0]).toBe("EVENT")
  socket.onmessage({ data: JSON.stringify(["AUTH", "challenge"]) }); await new Promise((resolve) => setTimeout(resolve, 0))
  expect(sent[1]?.[0]).toBe("AUTH"); expect(sent).toHaveLength(2)
  socket.onmessage({ data: JSON.stringify(["OK", "auth-id", true, ""]) }); await new Promise((resolve) => setTimeout(resolve, 0))
  expect(sent[2]?.[0]).toBe("EVENT")
  socket.onmessage({ data: JSON.stringify(["OK", "event", true, ""]) })
  await expect(promise).resolves.toBe("inserted")
}))

it("shares one authenticated socket and resumes all pending events", async () => {
  ;(globalThis as any).WebSocket = FakeWebSocket
  const session = await browserRelayTransport.openSession!("wss://relay.example", signer)
  const first = session.publish(event("one"))
  const second = session.publish(event("two"))
  socket.onmessage({ data: JSON.stringify(["AUTH", "challenge"]) })
  await new Promise((resolve) => setTimeout(resolve, 0))
  socket.onmessage({ data: JSON.stringify(["OK", "auth-id", true, ""]) })
  await new Promise((resolve) => setTimeout(resolve, 0))
  expect(sent.filter((frame) => frame[0] === "AUTH")).toHaveLength(1)
  expect(sent.filter((frame) => frame[0] === "EVENT" && (frame[1] as any).id === "one")).toHaveLength(2)
  expect(sent.filter((frame) => frame[0] === "EVENT" && (frame[1] as any).id === "two")).toHaveLength(2)
  socket.onmessage({ data: JSON.stringify(["OK", "one", true, ""]) })
  socket.onmessage({ data: JSON.stringify(["OK", "two", true, ""]) })
  await expect(first).resolves.toBe("inserted")
  await expect(second).resolves.toBe("inserted")
  session.close()
})

it("closes an opening socket when the import is aborted", async () => {
  let openingSocket: { close: ReturnType<typeof vi.fn>; onopen?: () => void; onmessage?: (event: { data: string }) => void; onerror?: () => void; onclose?: () => void } | undefined
  class HangingWebSocket {
    onopen: (() => void) | undefined; onmessage: ((event: { data: string }) => void) | undefined; onerror: (() => void) | undefined; onclose: (() => void) | undefined
    close = vi.fn()
    constructor() { openingSocket = this }
    send() {}
  }
  ;(globalThis as any).WebSocket = HangingWebSocket
  const controller = new AbortController()
  const opening = browserRelayTransport.openSession!("wss://relay.example", signer, undefined, controller.signal)
  const expected = expect(opening).rejects.toMatchObject({ name: "AbortError" })
  controller.abort()
  await expected
  expect(openingSocket?.close).toHaveBeenCalledOnce()
})

it("uses a trusted canonical relay tag while connecting to a local alias", async () => {
  ;(globalThis as any).WebSocket = FakeWebSocket
  const session = await browserRelayTransport.openSession!("ws://localhost:4869", signer, "wss://relay.example")
  const publish = session.publish(event())
  socket.onmessage({ data: JSON.stringify(["AUTH", "challenge"]) })
  await new Promise((resolve) => setTimeout(resolve, 0))
  expect((sent.find((frame) => frame[0] === "AUTH")?.[1] as any).tags).toContainEqual(["relay", "wss://relay.example"])
  socket.onmessage({ data: JSON.stringify(["OK", "auth-id", true, ""]) })
  await new Promise((resolve) => setTimeout(resolve, 0))
  socket.onmessage({ data: JSON.stringify(["OK", "event", true, ""]) })
  await expect(publish).resolves.toBe("inserted")
  session.close()
})

it("closes and reports a source relay that exceeds a page event limit", async () => {
  ;(globalThis as any).WebSocket = FakeWebSocket
  const session = await browserRelayTransport.openSession!("wss://relay.example", signer)
  const query = session.query([{ kinds: [4] }])
  const requestID = sent[0]![1]
  for (let index = 0; index <= 500; index++) socket.onmessage({ data: JSON.stringify(["EVENT", requestID, event(`event-${index}`)]) })
  await expect(query).rejects.toThrow("relay excedeu o limite da página de importação")
  expect(socket.close).toHaveBeenCalled()
})

it("rearms the publish timeout when AUTH resumes an in-flight event", async () => {
  vi.useFakeTimers()
  ;(globalThis as any).WebSocket = FakeWebSocket
  const session = await browserRelayTransport.openSession!("wss://relay.example", signer)
  const publish = session.publish(event())
  await vi.advanceTimersByTimeAsync(29_900)
  socket.onmessage({ data: JSON.stringify(["AUTH", "challenge"]) })
  await Promise.resolve(); await Promise.resolve()
  socket.onmessage({ data: JSON.stringify(["OK", "auth-id", true, ""]) })
  await Promise.resolve(); await Promise.resolve()
  await vi.advanceTimersByTimeAsync(1_000)
  socket.onmessage({ data: JSON.stringify(["OK", "event", true, ""]) })
  await expect(publish).resolves.toBe("inserted")
  session.close()
})

it("pauses pending work while a NIP-07 prompt takes longer than the request timeout", async () => {
  vi.useFakeTimers()
  ;(globalThis as any).WebSocket = FakeWebSocket
  let approveAuth: ((value: any) => void) | undefined
  const slowSigner: ImportSigner = {
    ...signer,
    signEvent: async (template) => new Promise((resolve) => { approveAuth = () => resolve({ id: "auth-id", pubkey: "owner", sig: "sig", ...template }) }),
  }
  const session = await browserRelayTransport.openSession!("wss://relay.example", slowSigner)
  const publish = session.publish(event())
  socket.onmessage({ data: JSON.stringify(["AUTH", "challenge"]) })
  await vi.advanceTimersByTimeAsync(0)
  expect(approveAuth).toBeDefined()
  await vi.advanceTimersByTimeAsync(31_000)
  expect(sent.filter((frame) => frame[0] === "EVENT")).toHaveLength(1)
  approveAuth!(undefined)
  await vi.advanceTimersByTimeAsync(0)
  expect(sent.filter((frame) => frame[0] === "AUTH")).toHaveLength(1)
  socket.onmessage({ data: JSON.stringify(["OK", "auth-id", true, ""]) })
  await Promise.resolve(); await Promise.resolve()
  socket.onmessage({ data: JSON.stringify(["OK", "event", true, ""]) })
  await expect(publish).resolves.toBe("inserted")
  session.close()
})

it("reissues a CLOSED auth-required REQ after AUTH", async () => {
  ;(globalThis as any).WebSocket = FakeWebSocket
  const session = await browserRelayTransport.openSession!("wss://relay.example", signer)
  const query = session.query([{ kinds: [4] }])
  const requestID = sent[0]![1]
  socket.onmessage({ data: JSON.stringify(["CLOSED", requestID, "auth-required: authenticate"]) })
  socket.onmessage({ data: JSON.stringify(["AUTH", "challenge"]) })
  await new Promise((resolve) => setTimeout(resolve, 0))
  socket.onmessage({ data: JSON.stringify(["OK", "auth-id", true, ""]) })
  await new Promise((resolve) => setTimeout(resolve, 0))
  expect(sent.filter((frame) => frame[0] === "REQ" && frame[1] === requestID)).toHaveLength(2)
  socket.onmessage({ data: JSON.stringify(["EOSE", requestID]) })
  await expect(query).resolves.toEqual([])
  session.close()
})

it("sends CLOSE and rejects a pending REQ when the caller aborts", async () => {
  ;(globalThis as any).WebSocket = FakeWebSocket
  const session = await browserRelayTransport.openSession!("wss://relay.example", signer)
  const controller = new AbortController()
  const query = session.query([{ kinds: [4] }], controller.signal)
  const expected = expect(query).rejects.toMatchObject({ name: "AbortError" })
  const requestID = sent[0]![1]
  controller.abort()
  await expected
  expect(sent).toContainEqual(["CLOSE", requestID])
  session.close()
})

it("backs off once when the local import queue is full", async () => {
  vi.useFakeTimers()
  ;(globalThis as any).WebSocket = FakeWebSocket
  const session = await browserRelayTransport.openSession!("wss://relay.example", signer)
  const publish = session.publish(event())
  socket.onmessage({ data: JSON.stringify(["OK", "event", false, "error: import queue wait timed out"]) })
  await vi.advanceTimersByTimeAsync(500)
  expect(sent.filter((frame) => frame[0] === "EVENT")).toHaveLength(2)
  socket.onmessage({ data: JSON.stringify(["OK", "event", false, "error: import queue wait timed out"]) })
  await expect(publish).resolves.toEqual({ status: "rejected", reason: "relay local não aceitou o evento após nova tentativa" })
  session.close()
})

it("paces a bounded retry when the relay rate-limits before queueing", async () => {
  vi.useFakeTimers()
  ;(globalThis as any).WebSocket = FakeWebSocket
  const session = await browserRelayTransport.openSession!("wss://relay.example", signer)
  const publish = session.publish(event())
  socket.onmessage({ data: JSON.stringify(["OK", "event", false, "rate-limited: DM import rate limit exceeded"]) })
  await vi.advanceTimersByTimeAsync(999)
  expect(sent.filter((frame) => frame[0] === "EVENT")).toHaveLength(1)
  await vi.advanceTimersByTimeAsync(1)
  expect(sent.filter((frame) => frame[0] === "EVENT")).toHaveLength(2)
  socket.onmessage({ data: JSON.stringify(["OK", "event", true, ""]) })
  await expect(publish).resolves.toBe("inserted")
  session.close()
})

it("keeps a queueing timeout uncertain after its bounded retry", async () => {
  vi.useFakeTimers()
  ;(globalThis as any).WebSocket = FakeWebSocket
  const session = await browserRelayTransport.openSession!("wss://relay.example", signer)
  const publish = session.publish(event())
  socket.onmessage({ data: JSON.stringify(["OK", "event", false, "error: import outcome uncertain (timed out after queueing)"]) })
  await vi.advanceTimersByTimeAsync(500)
  socket.onmessage({ data: JSON.stringify(["OK", "event", false, "error: import outcome uncertain (timed out after queueing)"]) })
  await expect(publish).resolves.toBe("uncertain")
  session.close()
})
