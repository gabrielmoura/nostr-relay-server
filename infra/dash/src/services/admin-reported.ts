import { adminApolloClient } from "@/graphql/client"
import { EventReportsDocument, ReportedEventsDocument, ReportedEventsSummaryDocument } from "@/graphql/generated/operations"
import { GraphQLApiError } from "@/graphql/helpers"
import type { AdminPage, EventReportsResponse, EventRecord, ReportedEventItem, ReportedEventsSummary } from "@/types/admin"

import { ApiError } from "./admin"

type PageParams = {
  limit: number
  offset: number
}

type ReportedEventsQueryFilters = {
  query: string
  type: string
  target_pubkey?: string
  target_event_id?: string
  since?: number
  until?: number
}

function toGraphApiError(error: unknown): never {
  if (error instanceof GraphQLApiError) {
    throw new ApiError(error.message, error.status, error.details, error.requestId)
  }
  throw error
}

function mapEventRecord(item: any): EventRecord {
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

function mapReportedEventItem(item: any): ReportedEventItem {
  const targetAuthor = item.targetAuthor
  const targetPubkey = item.targetPubkey ? String(item.targetPubkey) : targetAuthor?.pubkey ? String(targetAuthor.pubkey) : ""

  return {
    target_event_id: item.targetEventId,
    target_pubkey: targetPubkey || undefined,
    target_nevent: item.targetNevent ?? undefined,
    target_created_at: item.targetCreatedAt ?? undefined,
    target_created_at_iso: item.targetCreatedAtIso ?? undefined,
    target_author: targetAuthor ? {
      pubkey: targetAuthor.pubkey ? String(targetAuthor.pubkey) : targetPubkey,
      display_name: targetAuthor.displayName ? String(targetAuthor.displayName) : targetPubkey || "autor desconhecido",
      picture: targetAuthor.picture ? String(targetAuthor.picture) : undefined,
      nip05: targetAuthor.nip05 ? String(targetAuthor.nip05) : undefined,
    } : undefined,
    report_count: item.reportCount,
    last_reported: item.lastReported,
    last_reported_at: item.lastReportedAt ?? undefined,
    report_types: item.reportTypes ?? [],
    target_event: item.targetEvent ? mapEventRecord(item.targetEvent) : undefined,
  }
}

function buildReportedEventsFilter(filters: ReportedEventsQueryFilters) {
  return {
    q: filters.query || null,
    types: filters.type && filters.type !== "all" ? [filters.type] : null,
  }
}

export async function getEventReports(eventID: string, params: PageParams) {
  try {
    const result = await adminApolloClient.query({
      query: EventReportsDocument,
      variables: { id: eventID, page: { limit: params.limit, offset: params.offset } },
    })
    const data = result.data
    if (!data) {
      throw new ApiError("GraphQL query returned no data")
    }

    return {
      items: data.eventReports.items.map((item: any) => ({
        report_event_id: item.reportEventId,
        reporter_pubkey: item.reporterPubkey,
        reporter_npub: item.reporterNpub ?? undefined,
        reporter_display_name: item.reporterDisplayName,
        reporter_picture: item.reporterPicture ?? undefined,
        reported_event_id: item.reportedEventId ?? undefined,
        reported_pubkey: item.reportedPubkey ?? undefined,
        report_type: item.reportType ?? undefined,
        content: item.content ?? undefined,
        created_at: item.createdAt,
      })),
      total: data.eventReports.pageInfo.total,
    } satisfies EventReportsResponse
  } catch (error) {
    toGraphApiError(error)
  }
}

export async function getReportedEventsPage(filters: ReportedEventsQueryFilters, params: PageParams) {
  try {
    const result = await adminApolloClient.query({
      query: ReportedEventsDocument,
      variables: {
        filter: buildReportedEventsFilter(filters),
        page: { limit: params.limit, offset: params.offset },
      },
    })
    const data = result.data
    if (!data) {
      throw new ApiError("GraphQL query returned no data")
    }

    return {
      items: data.reportedEvents.items.map(mapReportedEventItem),
      total: data.reportedEvents.pageInfo.total,
      limit: data.reportedEvents.pageInfo.limit,
      offset: data.reportedEvents.pageInfo.offset,
      has_more: data.reportedEvents.pageInfo.hasMore,
    } satisfies AdminPage<ReportedEventItem>
  } catch (error) {
    toGraphApiError(error)
  }
}

export async function getReportedEventsSummary(filters: ReportedEventsQueryFilters) {
  try {
    const result = await adminApolloClient.query({
      query: ReportedEventsSummaryDocument,
      variables: {
        filter: buildReportedEventsFilter(filters),
      },
    })
    const data = result.data
    if (!data) {
      throw new ApiError("GraphQL query returned no data")
    }

    return {
      total_events: data.reportedEventsSummary.totalEvents,
      total_reports: data.reportedEventsSummary.totalReports,
      unique_target_authors: data.reportedEventsSummary.uniqueTargetAuthors,
      timeline: data.reportedEventsSummary.timeline.map((item: any) => ({ bucket: item.name, count: item.count })),
      report_types: data.reportedEventsSummary.reportTypes.map((item: any) => ({ name: item.name, count: item.count })),
      top_authors: data.reportedEventsSummary.topAuthors.map((item: any) => ({ pubkey: item.pubkey, display_name: item.displayName ?? item.pubkey, count: item.count })),
      top_targets: data.reportedEventsSummary.topTargets.map((item: any) => ({ target_event_id: item.targetEventId, count: item.count })),
    } satisfies ReportedEventsSummary
  } catch (error) {
    toGraphApiError(error)
  }
}
