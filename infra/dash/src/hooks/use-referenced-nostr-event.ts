import { useEffect, useState } from "react"
import { useNostr } from "@nostrify/react"

import type { EventRecord } from "@/types/admin"

const referenceEventCache = new Map<string, EventRecord | null>()
const pendingQueries = new Map<string, Promise<EventRecord | null>>()

export function useReferencedNostrEvent(eventID: string) {
  const { nostr } = useNostr()
  const [data, setData] = useState<EventRecord | null | undefined>(undefined)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<unknown>(null)

  useEffect(() => {
    if (!eventID) {
      setData(null)
      setIsLoading(false)
      return
    }

    if (referenceEventCache.has(eventID)) {
      setData(referenceEventCache.get(eventID))
      setIsLoading(false)
      return
    }

    let active = true
    setIsLoading(true)

    const fetchEvent = async () => {
      if (pendingQueries.has(eventID)) {
        return pendingQueries.get(eventID)!
      }

      const promise = (async () => {
        try {
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
        } catch (err) {
          throw err
        }
      })()

      pendingQueries.set(eventID, promise)
      try {
        const result = await promise
        referenceEventCache.set(eventID, result)
        return result
      } finally {
        pendingQueries.delete(eventID)
      }
    }

    fetchEvent()
      .then((record) => {
        if (active) {
          setData(record)
          setIsLoading(false)
        }
      })
      .catch((err) => {
        if (active) {
          setError(err)
          setIsLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [eventID, nostr])

  return {
    data,
    isLoading,
    isError: error != null,
    error,
  }
}
