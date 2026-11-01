export type NostrEvent = { id: string; pubkey: string; created_at: number; kind: number; tags: string[][]; content: string; sig: string }
export type EventTemplate = Omit<NostrEvent, "id" | "pubkey" | "sig">

export interface ImportSigner {
  getPublicKey(): Promise<string>
  signEvent(template: EventTemplate): Promise<NostrEvent>
  dispose(): void
}

export interface RelayTransport {
  query(url: string, filters: Record<string, unknown>[], signer: ImportSigner): Promise<NostrEvent[]>
  publish(url: string, event: NostrEvent, signer: ImportSigner): Promise<PublishResult>
  // authRelayURL is the NIP-42 relay tag. It may differ from the connection
  // URL when a relay is reached through an explicitly configured proxy alias.
  openSession?(url: string, signer: ImportSigner, authRelayURL?: string, signal?: AbortSignal): Promise<RelaySession>
  // Source relays only need REQ. Keeping this separate prevents the temporary
  // signer and receipt-aware local publisher from leaking into the app pool.
  openReadSession?(url: string, signer: ImportSigner): Promise<RelaySession>
}

export type PublishResult = "inserted" | "duplicate" | "uncertain" | { status: "rejected"; reason: string }

export interface RelaySession {
  query(filters: Record<string, unknown>[], signal?: AbortSignal): Promise<NostrEvent[]>
  publish(event: NostrEvent): Promise<PublishResult>
  close(): void | Promise<void>
}

export type ImportProgress = { relays: number; completedRelays: number; failedRelays: number; found: number; imported: number; duplicates: number; rejected: number; uncertain: number; errors: string[]; incomplete: boolean; phase: string }
