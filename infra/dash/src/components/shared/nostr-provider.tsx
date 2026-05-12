import { type ReactNode, useRef } from "react"
import { NPool, NRelay1, type NostrEvent } from "@nostrify/nostrify"
import { NostrContext } from "@nostrify/react"

function getLocalRelayURL() {
  if (typeof window === "undefined") {
    return "ws://localhost:4870/"
  }

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
  return `${protocol}//${window.location.host}/`
}

interface NostrProviderProps {
  children: ReactNode
}

export function NostrProvider({ children }: NostrProviderProps) {
  const poolRef = useRef<NPool | null>(null)
  const relayURL = getLocalRelayURL()

  if (!poolRef.current) {
    poolRef.current = new NPool({
      open(url: string) {
        return new NRelay1(url)
      },
      reqRouter(filters) {
        return new Map([[relayURL, filters]])
      },
      eventRouter(_event: NostrEvent) {
        return [relayURL]
      },
    })
  }

  return <NostrContext.Provider value={{ nostr: poolRef.current }}>{children}</NostrContext.Provider>
}
