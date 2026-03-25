import { nip19 } from "nostr-tools"

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
