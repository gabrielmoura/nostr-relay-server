import { useQuery } from "@tanstack/react-query"
import { useNostr } from "@nostrify/react"

import type { EventRecord } from "@/types/admin"

export function useReferencedNostrEvent(eventID: string) {
  const { nostr } = useNostr()

  return useQuery({
    queryKey: ["nostr-reference-event", eventID],
    enabled: Boolean(eventID),
    queryFn: async () => {
      const events = await nostr.query([{ ids: [eventID], limit: 1 }])
      const event = events[0]
      if (!event) {
        return null
      }

      const record: EventRecord = {
        id: event.id,
        pubkey: event.pubkey,
        kind: event.kind,
        created_at: event.created_at,
        content: event.content,
        tags: event.tags,
        sig: event.sig,
      }

      return record
    },
  })
}
