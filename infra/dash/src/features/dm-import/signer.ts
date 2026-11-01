import { finalizeEvent, getPublicKey, nip19 } from "nostr-tools"
import type { EventTemplate, ImportSigner, NostrEvent } from "./types"

type NIP07 = { getPublicKey(): Promise<string>; signEvent(event: EventTemplate): Promise<NostrEvent> }

export function extensionSigner(nostr: NIP07): ImportSigner {
  return { getPublicKey: () => nostr.getPublicKey(), signEvent: (event) => nostr.signEvent(event), dispose: () => undefined }
}

export function secretSigner(value: string): ImportSigner {
  const decoded = value.startsWith("nsec") ? nip19.decode(value).data : hexBytes(value)
  if (!(decoded instanceof Uint8Array)) throw new Error("Chave privada inválida")
  const secret = decoded.slice()
  return {
    async getPublicKey() { return getPublicKey(secret) },
    async signEvent(event) { return finalizeEvent(event, secret) as NostrEvent },
    dispose() { secret.fill(0) },
  }
}

function hexBytes(value: string) {
  if (!/^[0-9a-f]{64}$/i.test(value)) throw new Error("nsec deve ser nsec1 ou hexadecimal de 32 bytes")
  return Uint8Array.from(value.match(/../g)!.map((part) => Number.parseInt(part, 16)))
}
