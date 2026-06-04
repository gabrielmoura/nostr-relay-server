import { adminApolloClient } from "@/graphql/client"
import { GraphQLApiError } from "@/graphql/helpers"
import { EventAggregatesDocument, EventsDocument, EventTimelineDocument } from "@/graphql/generated/operations"
import type { EventAggregates, EventSearchFilters, EventSearchResponse, EventTimeline } from "@/types/admin"
import { env } from "@/lib/env"
import { mockEvents } from "@/mocks/admin"

import { ApiError, type PageParams } from "./admin"

function toGraphApiError(error: unknown): never {
  if (error instanceof GraphQLApiError) {
    throw new ApiError(error.message, error.status, error.details, error.requestId)
  }
  throw error
}

function isMockEnabled() {
  return env.mockOnFailure
}

function fallbackPage<T>(items: T[], limit: number, offset: number) {
  const sliced = items.slice(offset, offset + limit)
  return {
    items: sliced,
    total: items.length,
    limit,
    offset,
    has_more: offset + sliced.length < items.length,
  }
}

function buildEventFilter(filters: EventSearchFilters) {
  const tags = filters.tags.map((tag) => {
    const [name = "", ...rest] = tag.split(":")
    return { name: name.replace(/^#/, ""), value: rest.join(":") }
  }).filter((item) => item.name && item.value)

  return {
    q: filters.query || null,
    authors: filters.authors,
    kinds: filters.kinds,
    tags: tags.length > 0 ? tags : null,
  }
}

function mapEventRecord(item: any) {
  return {
    id: item.id,
    pubkey: item.pubkey,
    created_at: item.createdAt,
    kind: item.kind,
    content: item.content,
    sig: item.sig,
    tags: (item.tags ?? []).map((tag: any) => tag.values),
  }
}

export async function searchEventsPage(filters: EventSearchFilters, params: PageParams) {
  try {
    const result = await adminApolloClient.query({
      query: EventsDocument,
      variables: {
        filter: buildEventFilter(filters),
        page: { limit: params.limit || filters.limit, offset: params.offset },
      },
    })
    const data = result.data
    if (!data) {
      throw new ApiError("GraphQL query returned no data")
    }

    return {
      items: data.events.items.map(mapEventRecord),
      total: data.events.pageInfo.total,
      limit: data.events.pageInfo.limit,
      offset: data.events.pageInfo.offset,
      has_more: data.events.pageInfo.hasMore,
    } satisfies EventSearchResponse
  } catch (error) {
    if (!isMockEnabled()) {
      toGraphApiError(error)
    }

    const source = mockEvents.filter((event) => {
      const q = !filters.query || event.content.toLowerCase().includes(filters.query.toLowerCase())
      const a = filters.authors.length === 0 || filters.authors.includes(event.pubkey)
      const k = filters.kinds.length === 0 || filters.kinds.includes(event.kind)
      const t =
        filters.tags.length === 0 ||
        filters.tags.every((tag) => {
          const [key, value] = tag.split(":")
          if (!key || !value) {
            return false
          }
          const normalized = key.replace(/^#/, "")
          return event.tags.some((entry) => entry[0] === normalized && entry[1] === value)
        })
      return q && a && k && t
    })
    return fallbackPage(source, params.limit, params.offset)
  }
}

export async function getEventSearchAggregates(filters: EventSearchFilters) {
  try {
    const result = await adminApolloClient.query({
      query: EventAggregatesDocument,
      variables: { filter: buildEventFilter(filters) },
    })
    const data = result.data
    if (!data) {
      throw new ApiError("GraphQL query returned no data")
    }

    return {
      total: data.eventAggregates.total,
      kinds: data.eventAggregates.kinds.map((item: any) => ({ kind: item.kind, count: item.count })),
      top_authors: data.eventAggregates.topAuthors.map((item: any) => ({ pubkey: item.pubkey, display_name: item.displayName ?? undefined, count: item.count })),
      top_tags: data.eventAggregates.topTags.map((item: any) => ({ tag: item.tag, count: item.count })),
      trends: data.eventAggregates.trends,
    } satisfies EventAggregates
  } catch (error) {
    toGraphApiError(error)
  }
}

export async function getEventSearchTimeline(filters: EventSearchFilters, bucket: "hour" | "day") {
  try {
    const result = await adminApolloClient.query({
      query: EventTimelineDocument,
      variables: {
        filter: buildEventFilter(filters),
        bucket: bucket === "day" ? "DAY" : "HOUR",
      },
    })
    const data = result.data
    if (!data) {
      throw new ApiError("GraphQL query returned no data")
    }

    return {
      bucket: String(data.eventTimeline.bucket).toLowerCase() as EventTimeline["bucket"],
      points: data.eventTimeline.points.map((item: any) => ({ ts: item.ts, count: item.count })),
    } satisfies EventTimeline
  } catch (error) {
    toGraphApiError(error)
  }
}
