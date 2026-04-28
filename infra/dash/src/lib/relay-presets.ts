const relayStoragePrefix = "nrserver.relays"

export const commonRelayPresets = [
  "wss://relay.damus.io",
  "wss://relay.primal.net",
  "wss://nos.lol",
  "wss://relay.nostr.band",
  "wss://nostr.mom",
]

export function normalizeRelayList(values: string[]) {
  const unique = new Set<string>()
  for (const raw of values) {
    const value = raw.trim()
    if (!value || !/^wss?:\/\//.test(value)) {
      continue
    }
    unique.add(value)
  }
  return [...unique]
}

export function parseRelayCSV(value: string) {
  return normalizeRelayList(value.split(","))
}

export function relayStorageKey(scope: string) {
  return `${relayStoragePrefix}.${scope}`
}

export function readRelayStorage(scope: string, fallback: string[] = commonRelayPresets) {
  if (typeof window === "undefined") {
    return normalizeRelayList(fallback)
  }
  const stored = window.localStorage.getItem(relayStorageKey(scope))
  if (!stored) {
    return normalizeRelayList(fallback)
  }
  try {
    const parsed = JSON.parse(stored)
    if (!Array.isArray(parsed)) {
      return normalizeRelayList(fallback)
    }
    return normalizeRelayList(parsed.map(String))
  } catch {
    return normalizeRelayList(fallback)
  }
}

export function writeRelayStorage(scope: string, relays: string[]) {
  if (typeof window === "undefined") {
    return
  }
  window.localStorage.setItem(relayStorageKey(scope), JSON.stringify(normalizeRelayList(relays)))
}
