import type { ImportProgress, ImportSigner, NostrEvent, RelaySession, RelayTransport } from "./types"

export type ImportScope = "dm" | "events"
const DM_FILTERS = (pubkey: string) => [
  { kinds: [1059], "#p": [pubkey] },
  { kinds: [4], authors: [pubkey] },
  { kinds: [4], "#p": [pubkey] },
]
const EVENT_FILTERS = (pubkey: string) => [{ authors: [pubkey] }]
const maxImportEvents = 10_000
const maxImportBytes = 32 * 1024 * 1024

export async function discoverRelays(transport: RelayTransport, signer: ImportSigner, bootstrap: string[], signal?: AbortSignal, sessions?: Map<string, RelaySession>, scope: ImportScope = "dm") {
  const pubkey = await signer.getPublicKey()
  const urls = new Set(bootstrap.map(normalizeRelay).filter(Boolean))
  const ownedSessions = sessions ?? new Map<string, RelaySession>()
  const query = async (relay: string, filters: Record<string, unknown>[]) => {
    if (!transport.openReadSession) return transport.query(relay, filters, signer)
    let session = ownedSessions.get(relay)
    if (!session) {
      session = await transport.openReadSession(relay, signer)
      ownedSessions.set(relay, session)
    }
    return session.query(filters, signal)
  }
  for (const relay of [...urls]) {
    if (signal?.aborted) break
    try {
      let events: NostrEvent[] = []
      if (scope === "events") {
        events = await query(relay, [{ kinds: [10002], authors: [pubkey], limit: 1 }])
        const newest = events.sort((left, right) => right.created_at - left.created_at)[0]
        for (const tag of newest?.tags ?? []) if (newest?.kind === 10002 && tag[0] === "r" && tag[2] !== "write" && tag[1]) urls.add(normalizeRelay(tag[1]))
        continue
      }
      try { events = await query(relay, [{ kinds: [10050], authors: [pubkey], limit: 1 }]) } catch {
        if (signal?.aborted) break
        /* NIP-65 is the discovery fallback. */
      }
      if (signal?.aborted) break
      if (!events.length) events = await query(relay, [{ kinds: [10002], authors: [pubkey], limit: 1 }])
      if (signal?.aborted) break
      const newest = events.sort((left, right) => right.created_at - left.created_at)[0]
      for (const tag of newest?.tags ?? []) {
        const dmRelay = newest?.kind === 10050 && tag[0] === "relay"
        const readableNIP65 = newest?.kind === 10002 && tag[0] === "r" && tag[2] !== "write"
        if ((dmRelay || readableNIP65) && tag[1]) urls.add(normalizeRelay(tag[1]))
      }
      if (newest?.kind === 10050 && !newest.tags.some((tag) => tag[0] === "relay" && normalizeRelay(tag[1] ?? ""))) {
        const fallback = await query(relay, [{ kinds: [10002], authors: [pubkey], limit: 1 }])
        const fallbackEvent = fallback.sort((left, right) => right.created_at - left.created_at)[0]
        for (const tag of fallbackEvent?.tags ?? []) if (fallbackEvent?.kind === 10002 && tag[0] === "r" && tag[2] !== "write" && tag[1]) urls.add(normalizeRelay(tag[1]))
      }
    } catch {
      if (signal?.aborted) break
      /* Each relay is independent. */
    }
  }
  if (!sessions) await Promise.all([...ownedSessions.values()].map((session) => session.close()))
  return [...urls].filter(Boolean).slice(0, 20)
}

export async function importDMs(options: { transport: RelayTransport; signer: ImportSigner; bootstrap: string[]; localRelay: string; localAuthRelay?: string; scope?: ImportScope; onProgress?: (p: ImportProgress) => void; signal?: AbortSignal }) {
  const pubkey = await options.signer.getPublicKey()
  const scope = options.scope ?? "dm"
  const sourceSessions = new Map<string, RelaySession>()
  const relays = await discoverRelays(options.transport, options.signer, options.bootstrap, options.signal, sourceSessions, scope)
  const progress: ImportProgress = { relays: relays.length, completedRelays: 0, failedRelays: 0, found: 0, imported: 0, duplicates: 0, rejected: 0, uncertain: 0, errors: [], incomplete: false, phase: "collecting" }
  const events = new Map<string, NostrEvent>()
  let eventBytes = 0
  let collectionLimitReached = false
  const emit = () => options.onProgress?.({ ...progress })
  emit()
  if (options.signal?.aborted) return finishCancelled(progress, sourceSessions, emit)
  for (const relay of relays) {
    if (collectionLimitReached) break
    let session: RelaySession | undefined
    try {
      session = sourceSessions.get(relay) ?? await openReadSession(options.transport, relay, options.signer)
      sourceSessions.delete(relay)
      const collection = await fetchPaginated(session, scope === "dm" ? DM_FILTERS(pubkey) : EVENT_FILTERS(pubkey), options.signal)
      for (const event of collection.events) {
        const addressed = event.tags.some((tag) => tag[0] === "p" && tag[1] === pubkey)
        if (scope === "dm" ? !((event.kind === 4 && (event.pubkey === pubkey || addressed)) || (event.kind === 1059 && addressed)) : event.pubkey !== pubkey || event.kind === 4 || event.kind === 1059) continue
        if (events.has(event.id)) continue
        const bytes = JSON.stringify(event).length
        if (events.size >= maxImportEvents || eventBytes + bytes > maxImportBytes) {
          collectionLimitReached = true
          progress.incomplete = true
          break
        }
        events.set(event.id, event)
        eventBytes += bytes
      }
      progress.incomplete ||= collection.incomplete
    } catch (error) {
      if (!options.signal?.aborted) { progress.failedRelays++; progress.errors.push(`${relay}: ${error instanceof Error ? error.message : "falha"}`) }
    }
    finally { await session?.close(); progress.completedRelays++; progress.found = events.size; emit() }
    if (options.signal?.aborted) return finishCancelled(progress, sourceSessions, emit)
  }
  await Promise.all([...sourceSessions.values()].map((session) => session.close()))
  if (options.signal?.aborted) return finishCancelled(progress, sourceSessions, emit)
  progress.phase = "publishing"; emit()
  let local: RelaySession | undefined
  try {
    local = await openSession(options.transport, options.localRelay, options.signer, options.localAuthRelay, options.signal)
    await publishWithLimit(local, [...events.values()], progress, emit, options.signal)
  } catch (error) {
    if (!options.signal?.aborted) {
      progress.errors.push(`relay local: ${error instanceof Error ? error.message : "falha"}`)
      progress.uncertain += events.size
    }
  } finally {
    await local?.close()
  }
  progress.phase = options.signal?.aborted ? "cancelled" : "complete"
  progress.incomplete ||= options.signal?.aborted === true
  emit()
  return progress
}

async function finishCancelled(progress: ImportProgress, sourceSessions: Map<string, RelaySession>, emit: () => void) {
  await Promise.all([...sourceSessions.values()].map((session) => session.close()))
  sourceSessions.clear()
  progress.phase = "cancelled"
  progress.incomplete = true
  emit()
  return progress
}

async function fetchPaginated(session: RelaySession, filters: Record<string, unknown>[], signal?: AbortSignal) {
  const all = new Map<string, NostrEvent>()
  let incomplete = false
  const maxPages = 20
  const maxEvents = maxImportEvents
  for (const filter of filters) {
    let until = Math.floor(Date.now() / 1000)
    for (let page = 0; page < maxPages && all.size < maxEvents; page++) {
      if (signal?.aborted) return { events: [...all.values()], incomplete: true }
      const pageEvents = await session.query([{ ...filter, since: 0, until, limit: 500 }], signal)
      if (!pageEvents.length) break
      let fresh = 0
      let oldest = until
      for (const event of pageEvents) {
        if (!all.has(event.id)) {
          if (all.size >= maxEvents) { incomplete = true; break }
          all.set(event.id, event); fresh++
        }
        oldest = Math.min(oldest, event.created_at)
      }
      if (fresh === 0 || oldest >= until || oldest <= 0) { incomplete = true; break }
      until = oldest - 1
      if (page === maxPages - 1) incomplete = true
    }
    if (all.size >= maxEvents) { incomplete = true; break }
  }
  return { events: [...all.values()], incomplete }
}

async function openSession(transport: RelayTransport, url: string, signer: ImportSigner, authRelayURL?: string, signal?: AbortSignal): Promise<RelaySession> {
  if (transport.openSession) return transport.openSession(url, signer, authRelayURL, signal)
  return {
    query: (filters) => transport.query(url, filters, signer),
    publish: (event) => transport.publish(url, event, signer),
    close: () => undefined,
  }
}

async function openReadSession(transport: RelayTransport, url: string, signer: ImportSigner): Promise<RelaySession> {
  if (transport.openReadSession) return transport.openReadSession(url, signer)
  return openSession(transport, url, signer)
}

async function publishWithLimit(session: RelaySession, events: NostrEvent[], progress: ImportProgress, emit: () => void, signal?: AbortSignal) {
  const concurrency = 8
  let next = 0
  const worker = async () => {
    for (;;) {
      const index = next++
      const event = events[index]
      if (!event || signal?.aborted) return
      try {
        const result = await session.publish(event)
        if (result === "inserted") progress.imported++
        else if (result === "duplicate") progress.duplicates++
        else if (typeof result === "object" && result.status === "rejected") {
          progress.rejected++
          if (progress.errors.length < 50) progress.errors.push(`relay local: ${result.reason}`)
        }
        else progress.uncertain++
      } catch (error) {
        progress.uncertain++
        progress.errors.push(`relay local: ${error instanceof Error ? error.message : "falha"}`)
      }
      emit()
    }
  }
  const abort = () => { void session.close() }
  signal?.addEventListener("abort", abort, { once: true })
  try {
    await Promise.all(Array.from({ length: Math.min(concurrency, events.length) }, worker))
  } finally {
    signal?.removeEventListener("abort", abort)
  }
  if (signal?.aborted) { progress.phase = "cancelled"; progress.incomplete = true }
}

function normalizeRelay(value: string) { try { const url = new URL(value); return url.protocol === "ws:" || url.protocol === "wss:" ? url.toString().replace(/\/$/, "") : "" } catch { return "" } }
