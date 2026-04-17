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