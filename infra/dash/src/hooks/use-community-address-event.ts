import { useEffect, useState } from "react"
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

const communityAddressCache = new Map<string, EventRecord | null>()
const pendingCommunityQueries = new Map<string, Promise<EventRecord | null>>()

export function useCommunityAddressEvent(address: string) {
  const { nostr } = useNostr()
  const pointer = parseAddressPointer(address)
  const [data, setData] = useState<EventRecord | null | undefined>(undefined)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<unknown>(null)

  useEffect(() => {
    const isEnabled = Boolean(pointer && pointer.kind === 34550)
    if (!isEnabled || !pointer) {
      setData(null)
      setIsLoading(false)
      return
    }

    if (communityAddressCache.has(address)) {
      setData(communityAddressCache.get(address))
      setIsLoading(false)
      return
    }

    let active = true
    setIsLoading(true)

    const fetchEvent = async () => {
      if (pendingCommunityQueries.has(address)) {
        return pendingCommunityQueries.get(address)!
      }

      const promise = (async () => {
        try {
          const events = await nostr.query([{
            kinds: [pointer.kind],
            authors: [pointer.pubkey],
            "#d": [pointer.identifier],
            limit: 1,
          }])

          const event = events[0]
          return event ? toEventRecord(event) : null
        } catch (err) {
          throw err
        }
      })()

      pendingCommunityQueries.set(address, promise)
      try {
        const result = await promise
        communityAddressCache.set(address, result)
        return result
      } finally {
        pendingCommunityQueries.delete(address)
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
  }, [address, pointer, nostr])

  return {
    data,
    isLoading,
    isError: error != null,
    error,
  }
}
