import { useQuery } from "@tanstack/react-query"
import { useNostr } from "@nostrify/react"

import { parseAddressPointer } from "@/lib/event-parser"
import type { EventRecord } from "@/types/admin"

function toEventRecord(event: {
  id: string
  pubkey: string
  kind: number
  created_at: number
  content: string
  tags: string[][]
  sig?: string
}): EventRecord {
  return {
    id: event.id,
    pubkey: event.pubkey,
    kind: event.kind,
    created_at: event.created_at,
    content: event.content,
    tags: event.tags,
    sig: event.sig,
  }
}

export function useCommunityAddressEvent(address: string) {
  const { nostr } = useNostr()
  const pointer = parseAddressPointer(address)

  return useQuery({
    queryKey: ["nostr-community-address", address],
    enabled: Boolean(pointer && pointer.kind === 34550),
    queryFn: async () => {
      if (!pointer) {
        return null
      }

      const events = await nostr.query([{
        kinds: [pointer.kind],
        authors: [pointer.pubkey],
        "#d": [pointer.identifier],
        limit: 1,
      }])

      const event = events[0]
      return event ? toEventRecord(event) : null
    },
  })
}
