import type { EventRecord, EventSearchFilters } from "@/types/admin"

export const defaultFilters: EventSearchFilters = {
  query: "",
  authors: [],
  kinds: [],
  tags: [],
  limit: 100,
}

export type EventSearchRouteSearch = {
  q?: string
  authors?: string
  kinds?: string
  tags?: string
  limit?: number
}

export function parseCSV(value?: string): string[] {
  if (!value) {
    return []
  }
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
}

export function parseSearchToFilters(search: EventSearchRouteSearch): EventSearchFilters {
  const parsedKinds = parseCSV(search.kinds)
    .map((value) => Number(value))
    .filter((value) => !Number.isNaN(value))

  const parsedTags = parseCSV(search.tags).map((value) => {
    if (value.includes(":")) {
      return value
    }
    return `t:${value}`
  })

  return {
    ...defaultFilters,
    query: search.q ?? "",
    authors: parseCSV(search.authors),
    kinds: parsedKinds,
    tags: parsedTags,
    limit: typeof search.limit === "number" && search.limit > 0 ? search.limit : defaultFilters.limit,
  }
}

export function filtersToSearch(filters: EventSearchFilters): EventSearchRouteSearch {
  return {
    q: filters.query || undefined,
    authors: filters.authors.length > 0 ? filters.authors.join(",") : undefined,
    kinds: filters.kinds.length > 0 ? filters.kinds.join(",") : undefined,
    tags: filters.tags.length > 0 ? filters.tags.join(",") : undefined,
    limit: filters.limit !== defaultFilters.limit ? filters.limit : undefined,
  }
}

export type EventRef = { id: string; relay?: string }

export function parseEventRefs(eventItem: EventRecord): EventRef[] {
  const refs: EventRef[] = []
  for (const tag of eventItem.tags) {
    if (tag[0] === "e" && tag[1]) {
      refs.push({ id: tag[1], relay: tag[2] })
    }
  }
  return refs
}

export function parseServers(eventItem: EventRecord): string[] {
  const servers: string[] = []
  for (const tag of eventItem.tags) {
    if (tag[0] === "server" && tag[1]) {
      servers.push(tag[1])
    }
  }
  return servers
}

export type ProfileData = {
  name?: string
  display_name?: string
  about?: string
  picture?: string
  nip05?: string
} | null

export function parseProfileContent(content: string): ProfileData {
  try {
    const parsed = JSON.parse(content) as ProfileData
    return parsed
  } catch {
    return null
  }
}

export function tagValue(eventItem: EventRecord, key: string): string {
  const tag = eventItem.tags.find((entry) => entry[0] === key && entry[1])
  return tag?.[1] ?? ""
}

export function eventHeadline(eventItem: EventRecord): string {
  if (eventItem.kind === 30003) {
    const title = tagValue(eventItem, "title")
    const dTag = tagValue(eventItem, "d")
    return title || dTag || "(lista sem titulo)"
  }
  return eventItem.content || "(sem conteudo textual)"
}

export interface NostrFilter {
  ids?: string[]
  authors?: string[]
  kinds?: number[]
  since?: number
  until?: number
  limit?: number
  search?: string
  [key: `#${string}`]: string[]
  [key: string]: any
}

export function nostrFilterToEventSearch(nf: NostrFilter): EventSearchFilters {
  const tags: string[] = []
  
  // Convert standard tag filters (#t, #p, etc) to CSV tags (t:value)
  Object.entries(nf).forEach(([key, values]) => {
    if (key.startsWith("#") && Array.isArray(values)) {
      const tagName = key.slice(1)
      values.forEach(v => tags.push(`${tagName}:${v}`))
    }
  })

  return {
    query: nf.search || "",
    authors: nf.authors || [],
    kinds: nf.kinds || [],
    tags: tags,
    limit: nf.limit || 100,
  }
}

export function eventSearchToNostrFilter(es: EventSearchFilters): NostrFilter {
  const nf: NostrFilter = {
    search: es.query || undefined,
    authors: es.authors.length > 0 ? es.authors : undefined,
    kinds: es.kinds.length > 0 ? es.kinds : undefined,
    limit: es.limit || 100,
  }

  es.tags.forEach(t => {
    const [key, value] = t.split(":")
    if (key && value) {
      const tagKey = `#${key}` as `#${string}`
      nf[tagKey] = [...(nf[tagKey] || []), value]
    }
  })

  return nf
}