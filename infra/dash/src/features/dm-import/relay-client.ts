import { NRelay1, type NostrFilter } from "@nostrify/nostrify"
import type { EventTemplate, ImportSigner, NostrEvent, PublishResult, RelaySession, RelayTransport } from "./types"

const requestTimeoutMs = 30_000
const retryDelayMs = 500
const rateLimitRetryDelayMs = 1_000
const maxPendingEvents = 8
const maxQueuedEvents = 16
const maxPublishRetries = 1

type PendingEvent = {
  event: NostrEvent
  resolve: (result: PublishResult) => void
  timer?: number
  retryTimer?: number
  retryAttempts: number
  retryAfterAuth: boolean
}

type PendingQuery = {
  filters: Record<string, unknown>[]
  events: NostrEvent[]
  bytes: number
  resolve: (events: NostrEvent[]) => void
  reject: (error: Error) => void
  timer?: number
  retryAfterAuth: boolean
  signal?: AbortSignal
  abort?: () => void
}

export const browserRelayTransport: RelayTransport = {
  async openSession(url, signer, authRelayURL, signal) { return BrowserRelaySession.open(url, signer, authRelayURL, signal) },
  async query(url, filters, signer) {
    const session = await BrowserRelaySession.open(url, signer)
    try { return await session.query(filters) } finally { session.close() }
  },
  async publish(url, event, signer) {
    const session = await BrowserRelaySession.open(url, signer)
    try { return await session.publish(event) } finally { session.close() }
  },
}

type NostrifyRelay = {
  req(filters: NostrFilter[], options?: { signal?: AbortSignal }): AsyncIterable<["EVENT", string, NostrEvent] | ["EOSE", string] | ["CLOSED", string, string]>
  close(): Promise<void>
}

type NostrifyRelayFactory = (url: string, options: { auth(challenge: string): Promise<NostrEvent> }) => NostrifyRelay

const maxReadEvents = 500
const maxReadBytes = 8 * 1024 * 1024

/**
 * Uses Nostrify's reconnecting NRelay1 for source-relay reads. Publication is
 * intentionally delegated to BrowserRelaySession because it preserves OK
 * receipt details needed by the import summary.
 */
export function createNostrifyReadTransport(createRelay: NostrifyRelayFactory = (url, options) => new NRelay1(url, options)): Pick<RelayTransport, "query" | "openReadSession"> {
  const openReadSession = async (url: string, signer: ImportSigner) => new NostrifyReadSession(createRelay(url, { auth: (challenge) => signedAuth(url, challenge, signer) }))
  return {
    openReadSession,
    async query(url, filters, signer) {
      const session = await openReadSession(url, signer)
      try { return await session.query(filters) } finally { session.close() }
    },
  }
}

export const nostrifyReadTransport = createNostrifyReadTransport()

export const importRelayTransport: RelayTransport = {
  ...browserRelayTransport,
  openReadSession: nostrifyReadTransport.openReadSession,
  query: nostrifyReadTransport.query,
}

class NostrifyReadSession implements RelaySession {
  private closed = false

  constructor(private readonly relay: NostrifyRelay) {}

  async query(filters: Record<string, unknown>[], signal?: AbortSignal): Promise<NostrEvent[]> {
    if (this.closed) throw new Error("conexão fechada")
    const controller = new AbortController()
    const abort = () => controller.abort()
    signal?.addEventListener("abort", abort, { once: true })
    if (signal?.aborted) controller.abort()
    const timer = window.setTimeout(() => controller.abort(), requestTimeoutMs)
    const events: NostrEvent[] = []
    let bytes = 0
    try {
      for await (const message of this.relay.req(filters as NostrFilter[], { signal: controller.signal })) {
        if (message[0] === "EOSE") break
        if (message[0] === "CLOSED") throw new Error(message[2] || "relay fechou a consulta")
        const event = message[2]
        const eventBytes = JSON.stringify(event).length
        if (events.length >= maxReadEvents || bytes + eventBytes > maxReadBytes) {
          throw new Error("relay excedeu o limite da página de importação")
        }
        bytes += eventBytes
        events.push(event)
      }
      return events
    } catch (error) {
      if (controller.signal.aborted && !this.closed) {
        if (signal?.aborted) throw new DOMException("The operation was aborted", "AbortError")
        throw new Error("tempo de conexão esgotado")
      }
      throw error
    } finally {
      window.clearTimeout(timer)
      signal?.removeEventListener("abort", abort)
    }
  }

  async publish(): Promise<PublishResult> {
    return { status: "rejected", reason: "sessão de leitura não publica eventos" }
  }

  async close() {
    if (this.closed) return
    this.closed = true
    await this.relay.close().catch(() => undefined)
  }
}

class BrowserRelaySession implements RelaySession {
  private constructor(private readonly ws: WebSocket, private readonly authRelay: string, private readonly signer: ImportSigner) {}
  private readonly events = new Map<string, PendingEvent>()
  private readonly queuedEvents: PendingEvent[] = []
  private readonly queries = new Map<string, PendingQuery>()
  private authPromise: Promise<void> | undefined
  private authPending: { id: string; timer: number; resolve: () => void; reject: (error: Error) => void } | undefined
  private rateLimitRetryAt = 0
  private closed = false

  static open(url: string, signer: ImportSigner, authRelayURL = url, signal?: AbortSignal): Promise<BrowserRelaySession> {
    return new Promise((resolve, reject) => {
      const ws = new WebSocket(url)
      let settled = false
      let timer: number | undefined
      const finish = (callback: () => void) => {
        if (settled) return
        settled = true
        if (timer !== undefined) window.clearTimeout(timer)
        signal?.removeEventListener("abort", abort)
        callback()
      }
      const abort = () => finish(() => { ws.close(); reject(new DOMException("The operation was aborted", "AbortError")) })
      if (signal?.aborted) { abort(); return }
      timer = window.setTimeout(() => finish(() => { ws.close(); reject(new Error("tempo de conexão esgotado")) }), requestTimeoutMs)
      signal?.addEventListener("abort", abort, { once: true })
      ws.onerror = () => finish(() => reject(new Error("falha de WebSocket")))
      ws.onopen = () => {
        finish(() => {
          const session = new BrowserRelaySession(ws, authRelayURL, signer)
          session.bind()
          resolve(session)
        })
      }
    })
  }

  query(filters: Record<string, unknown>[], signal?: AbortSignal): Promise<NostrEvent[]> {
    const id = crypto.randomUUID()
    return new Promise((resolve, reject) => {
      const pending: PendingQuery = { filters, events: [], bytes: 0, resolve, reject, retryAfterAuth: false, signal }
      if (signal?.aborted) { reject(new DOMException("The operation was aborted", "AbortError")); return }
      pending.abort = () => {
        if (this.queries.get(id) !== pending) return
        this.queries.delete(id)
        this.finishQuery(pending)
        this.send(["CLOSE", id])
        reject(new DOMException("The operation was aborted", "AbortError"))
      }
      signal?.addEventListener("abort", pending.abort, { once: true })
      this.queries.set(id, pending)
      if (this.authPromise) pending.retryAfterAuth = true
      else this.sendQuery(id, pending)
    })
  }

  publish(event: NostrEvent): Promise<PublishResult> {
    return new Promise((resolve) => {
      const pending: PendingEvent = { event, resolve, retryAttempts: 0, retryAfterAuth: Boolean(this.authPromise) }
      if (this.events.size >= maxPendingEvents || this.authPromise) {
        if (this.queuedEvents.length >= maxQueuedEvents) {
          resolve({ status: "rejected", reason: "fila local de publicação está cheia" })
          return
        }
        this.queuedEvents.push(pending)
        return
      }
      this.startEvent(pending)
    })
  }

  close() {
    if (this.closed) return
    this.closed = true
    this.ws.close()
    if (this.authPending) { window.clearTimeout(this.authPending.timer); this.authPending.reject(new Error("conexão fechada")); this.authPending = undefined }
    for (const pending of this.events.values()) { this.pauseEvent(pending); pending.resolve("uncertain") }
    for (const pending of this.queuedEvents) pending.resolve("uncertain")
    for (const pending of this.queries.values()) { this.finishQuery(pending); pending.reject(new Error("conexão fechada")) }
    this.events.clear(); this.queuedEvents.length = 0; this.queries.clear()
  }

  private bind() {
    this.ws.onerror = () => this.close()
    this.ws.onclose = () => this.close()
    this.ws.onmessage = (message) => void this.receive(message.data)
  }

  private async receive(data: unknown) {
    let frame: unknown[]
    try { frame = JSON.parse(String(data)) } catch { return }
    if (frame[0] === "AUTH" && typeof frame[1] === "string") {
      for (const pending of this.events.values()) { pending.retryAfterAuth = true; this.pauseEvent(pending) }
      for (const pending of this.queuedEvents) pending.retryAfterAuth = true
      for (const pending of this.queries.values()) { pending.retryAfterAuth = true; this.pauseQuery(pending) }
      try { await this.authenticate(frame[1]) } catch (error) { this.rejectAuthenticatedPending(error) }
      return
    }
    if (frame[0] === "EVENT" && typeof frame[1] === "string" && frame[2] && typeof frame[2] === "object") {
      const pending = this.queries.get(frame[1])
      if (!pending) return
      const encoded = JSON.stringify(frame[2])
      if (pending.events.length >= 500 || pending.bytes + encoded.length > 8 * 1024 * 1024) {
        this.queries.delete(frame[1]); this.finishQuery(pending); this.send(["CLOSE", frame[1]]); this.close(); pending.reject(new Error("relay excedeu o limite da página de importação")); return
      }
      pending.bytes += encoded.length; pending.events.push(frame[2] as NostrEvent); return
    }
    if (frame[0] === "EOSE" && typeof frame[1] === "string") {
      const pending = this.queries.get(frame[1])
      if (pending) { this.queries.delete(frame[1]); this.finishQuery(pending); pending.resolve(pending.events) }
      return
    }
    if (frame[0] === "OK" && typeof frame[1] === "string") {
      if (this.authPending?.id === frame[1]) {
        const pending = this.authPending
        this.authPending = undefined
        window.clearTimeout(pending.timer)
        if (frame[2] === true) pending.resolve()
        else pending.reject(new Error(String(frame[3] ?? "autenticação NIP-42 recusada")))
        return
      }
      const pending = this.events.get(frame[1])
      if (!pending) return
      const reason = String(frame[3] ?? "")
      if (frame[2] === true) {
        this.finishEvent(frame[1], pending, reason.startsWith("duplicate:") ? "duplicate" : "inserted")
      } else if (reason.includes("auth-required")) {
        pending.retryAfterAuth = true
      } else if (isRetryableImportResult(reason)) {
        this.retryEvent(pending, reason.includes("outcome uncertain") ? "uncertain" : "rejected", reason.startsWith("rate-limited:") ? this.nextRateLimitDelay() : retryDelayMs)
      } else this.finishEvent(frame[1], pending, { status: "rejected", reason })
      return
    }
    if (frame[0] === "CLOSED" && typeof frame[1] === "string") {
      const pending = this.queries.get(frame[1])
      if (!pending) return
      const reason = String(frame[2] ?? "relay fechou a consulta")
      if (reason.includes("auth-required")) pending.retryAfterAuth = true
      else { this.queries.delete(frame[1]); this.finishQuery(pending); pending.reject(new Error(reason)) }
    }
  }

  private authenticate(challenge: string): Promise<void> {
    if (this.authPromise) return this.authPromise
    this.authPromise = (async () => {
      const auth = await signedAuth(this.authRelay, challenge, this.signer)
      await new Promise<void>((resolve, reject) => {
        const timer = window.setTimeout(() => reject(new Error("tempo de autenticação esgotado")), requestTimeoutMs)
        this.authPending = { id: auth.id, timer, resolve, reject }
        this.send(["AUTH", auth])
      })
    })().then(() => {
      for (const [id, pending] of this.queries) if (pending.retryAfterAuth) { pending.retryAfterAuth = false; this.sendQuery(id, pending, true) }
      for (const pending of this.events.values()) if (pending.retryAfterAuth) { pending.retryAfterAuth = false; this.sendEvent(pending, true) }
      this.flushQueuedEvents()
    }).finally(() => { this.authPromise = undefined; this.flushQueuedEvents() })
    return this.authPromise
  }

  private startEvent(pending: PendingEvent) {
    this.events.set(pending.event.id, pending)
    this.sendEvent(pending)
  }

  private sendEvent(pending: PendingEvent, authenticated = false) {
    if (this.authPromise && !authenticated) { pending.retryAfterAuth = true; return }
    if (pending.timer) window.clearTimeout(pending.timer)
    pending.timer = window.setTimeout(() => {
      if (this.events.get(pending.event.id) !== pending) return
      this.retryEvent(pending, "uncertain")
    }, requestTimeoutMs)
    this.send(["EVENT", pending.event])
  }

  private retryEvent(pending: PendingEvent, exhaustedResult: "uncertain" | "rejected", delay = retryDelayMs) {
    if (pending.retryAttempts >= maxPublishRetries) {
      this.finishEvent(pending.event.id, pending, exhaustedResult === "uncertain" ? "uncertain" : { status: "rejected", reason: "relay local não aceitou o evento após nova tentativa" })
      return
    }
    pending.retryAttempts += 1
    this.pauseEvent(pending)
    pending.retryTimer = window.setTimeout(() => {
      pending.retryTimer = undefined
      if (this.events.get(pending.event.id) === pending) this.sendEvent(pending)
    }, delay)
  }

  private finishEvent(id: string, pending: PendingEvent, result: PublishResult) {
    if (this.events.get(id) !== pending) return
    this.events.delete(id)
    this.pauseEvent(pending)
    pending.resolve(result)
    this.flushQueuedEvents()
  }

  private flushQueuedEvents() {
    while (!this.closed && !this.authPromise && this.events.size < maxPendingEvents && this.queuedEvents.length > 0) {
      const pending = this.queuedEvents.shift()!
      this.startEvent(pending)
    }
  }

  private rejectAuthenticatedPending(error: unknown) {
    const reason = error instanceof Error ? error.message : "autenticação NIP-42 recusada"
    for (const [id, pending] of this.events) {
      if (pending.retryAfterAuth) this.finishEvent(id, pending, { status: "rejected", reason })
    }
    for (let index = this.queuedEvents.length - 1; index >= 0; index--) {
      const pending = this.queuedEvents[index]!
      if (!pending.retryAfterAuth) continue
      this.queuedEvents.splice(index, 1)
      pending.resolve({ status: "rejected", reason })
    }
    for (const [id, pending] of this.queries) {
      if (!pending.retryAfterAuth) continue
      this.queries.delete(id)
      this.finishQuery(pending)
      pending.reject(new Error(reason))
    }
  }

  private sendQuery(id: string, pending: PendingQuery, authenticated = false) {
    if (this.authPromise && !authenticated) { pending.retryAfterAuth = true; return }
    this.pauseQuery(pending)
    pending.timer = window.setTimeout(() => {
      if (this.queries.get(id) !== pending) return
      this.queries.delete(id)
      this.finishQuery(pending)
      this.send(["CLOSE", id])
      pending.reject(new Error("tempo de conexão esgotado"))
    }, requestTimeoutMs)
    this.send(["REQ", id, ...pending.filters])
  }

  private pauseEvent(pending: PendingEvent) {
    if (pending.timer) { window.clearTimeout(pending.timer); pending.timer = undefined }
    if (pending.retryTimer) { window.clearTimeout(pending.retryTimer); pending.retryTimer = undefined }
  }

  private pauseQuery(pending: PendingQuery) {
    if (pending.timer) { window.clearTimeout(pending.timer); pending.timer = undefined }
  }

  private finishQuery(pending: PendingQuery) {
    this.pauseQuery(pending)
    if (pending.abort) pending.signal?.removeEventListener("abort", pending.abort)
  }

  private nextRateLimitDelay() {
    const now = Date.now()
    const retryAt = Math.max(now + rateLimitRetryDelayMs, this.rateLimitRetryAt)
    this.rateLimitRetryAt = retryAt + rateLimitRetryDelayMs
    return retryAt - now
  }

  private send(frame: unknown[]) { if (!this.closed) this.ws.send(JSON.stringify(frame)) }
}

function isRetryableImportResult(reason: string) {
  return reason.startsWith("rate-limited:") || reason.includes("import outcome uncertain (timed out after queueing)") || reason.includes("import queue wait timed out")
}

async function signedAuth(relay: string, challenge: string, signer: ImportSigner) {
  const parsed = new URL(relay)
  if (!challenge || (parsed.protocol !== "ws:" && parsed.protocol !== "wss:")) throw new Error("desafio NIP-42 inválido")
  const template: EventTemplate = { kind: 22242, created_at: Math.floor(Date.now() / 1000), content: "", tags: [["relay", parsed.toString().replace(/\/$/, "")], ["challenge", challenge]] }
  return signer.signEvent(template)
}
