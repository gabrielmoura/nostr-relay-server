export type RelayInformation = {
  name?: string
  description?: string
  pubkey?: string
  supported_nips?: number[]
  limitation?: Record<string, unknown>
  public_endpoints?: { method: string; path: string; description: string }[]
  auth_relay_url?: string
  auth_relay_urls?: string[]
}

export async function getRelayInformation(): Promise<RelayInformation> {
  const response = await fetch("/", { headers: { Accept: "application/nostr+json" } })
  if (!response.ok) throw new Error("Não foi possível carregar o documento NIP-11")
  return response.json() as Promise<RelayInformation>
}

export function relayWebSocketURL(location: Location) {
  const protocol = location.protocol === "https:" ? "wss:" : "ws:"
  return `${protocol}//${location.host}/`
}
