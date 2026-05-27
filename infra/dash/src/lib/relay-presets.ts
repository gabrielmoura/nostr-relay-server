import { useRelayPresetsStore } from "@/stores/relay-presets-store"

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
  return normalizeRelayList(useRelayPresetsStore.getState().getScopeRelays(relayStorageKey(scope), normalizeRelayList(fallback)))
}

export function writeRelayStorage(scope: string, relays: string[]) {
  useRelayPresetsStore.getState().setScopeRelays(relayStorageKey(scope), normalizeRelayList(relays))
}
