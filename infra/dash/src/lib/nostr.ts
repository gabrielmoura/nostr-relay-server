import { nip19 } from "nostr-tools"
import type { AdminLabelTargetType } from "@/types/admin"

export function toNpub(pubkey: string) {
  if (!pubkey) {
    return ""
  }
  if (pubkey.startsWith("npub")) {
    return pubkey
  }

  try {
    return nip19.npubEncode(pubkey)
  } catch {
    return ""
  }
}

export function toNprofile(pubkey: string, relays: string[] = []) {
  if (!pubkey) {
    return ""
  }

  try {
    return nip19.nprofileEncode({ pubkey, relays })
  } catch {
    return ""
  }
}

export function toNevent(eventID: string, author?: string, relays: string[] = []) {
  if (!eventID) {
    return ""
  }

  try {
    return nip19.neventEncode({ id: eventID, author, relays })
  } catch {
    return ""
  }
}

export function toNote(eventID: string) {
  if (!eventID) {
    return ""
  }

  try {
    return nip19.noteEncode(eventID)
  } catch {
    return ""
  }
}

export function normalizeNip19TargetInput(targetType: AdminLabelTargetType, value: string) {
  const trimmed = value.trim()
  if (!trimmed) {
    return ""
  }

  if (!trimmed.startsWith("n")) {
    return trimmed
  }

  try {
    const decoded = nip19.decode(trimmed)

    if (targetType === "event") {
      if (decoded.type === "note") {
        return decoded.data as string
      }
      if (decoded.type === "nevent") {
        return (decoded.data as { id: string }).id
      }
    }

    if (targetType === "pubkey") {
      if (decoded.type === "npub") {
        return decoded.data as string
      }
      if (decoded.type === "nprofile") {
        return (decoded.data as { pubkey: string }).pubkey
      }
    }

    if (targetType === "address" && decoded.type === "naddr") {
      const data = decoded.data as { kind: number; pubkey: string; identifier: string }
      return `${data.kind}:${data.pubkey}:${data.identifier}`
    }

    return trimmed
  } catch {
    return trimmed
  }
}

export function normalizeFilterIdentifier(kind: "event" | "pubkey" | "address" | "generic", value: string) {
  const trimmed = value.trim()
  if (!trimmed) {
    return ""
  }
  if (!trimmed.startsWith("n")) {
    return trimmed
  }

  try {
    const decoded = nip19.decode(trimmed)
    if (kind === "event") {
      if (decoded.type === "note") {
        return decoded.data as string
      }
      if (decoded.type === "nevent") {
        return (decoded.data as { id: string }).id
      }
    }
    if (kind === "pubkey") {
      if (decoded.type === "npub") {
        return decoded.data as string
      }
      if (decoded.type === "nprofile") {
        return (decoded.data as { pubkey: string }).pubkey
      }
    }
    if (kind === "address" && decoded.type === "naddr") {
      const data = decoded.data as { kind: number; pubkey: string; identifier: string }
      return `${data.kind}:${data.pubkey}:${data.identifier}`
    }
    if (kind === "generic") {
      if (decoded.type === "note") {
        return decoded.data as string
      }
      if (decoded.type === "nevent") {
        return (decoded.data as { id: string }).id
      }
      if (decoded.type === "npub") {
        return decoded.data as string
      }
      if (decoded.type === "nprofile") {
        return (decoded.data as { pubkey: string }).pubkey
      }
      if (decoded.type === "naddr") {
        const data = decoded.data as { kind: number; pubkey: string; identifier: string }
        return `${data.kind}:${data.pubkey}:${data.identifier}`
      }
    }
    return trimmed
  } catch {
    return trimmed
  }
}
