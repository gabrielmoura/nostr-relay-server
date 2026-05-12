import type { ReportedEventsFilters } from "@/types/admin"

export type ReportedEventsRouteSearch = {
  q?: string
  type?: string
  targetEventId?: string
  targetPubkey?: string
  since?: number
  until?: number
}

export function normalizeReportedEventsSearch(search: Record<string, unknown>): ReportedEventsRouteSearch {
  return {
    q: typeof search.q === "string" ? search.q : undefined,
    type: typeof search.type === "string" ? search.type : undefined,
    targetEventId: typeof search.targetEventId === "string" ? search.targetEventId : undefined,
    targetPubkey: typeof search.targetPubkey === "string" ? search.targetPubkey : undefined,
    since: typeof search.since === "number" ? search.since : undefined,
    until: typeof search.until === "number" ? search.until : undefined,
  }
}

export function reportedEventsSearchToFilters(search: ReportedEventsRouteSearch): ReportedEventsFilters {
  return {
    query: search.q ?? "",
    type: search.type ?? "all",
    target_event_id: search.targetEventId || undefined,
    target_pubkey: search.targetPubkey || undefined,
    since: typeof search.since === "number" ? search.since : undefined,
    until: typeof search.until === "number" ? search.until : undefined,
  }
}

export function reportedEventsFiltersToSearch(filters: ReportedEventsFilters): ReportedEventsRouteSearch {
  return {
    q: filters.query || undefined,
    type: filters.type && filters.type !== "all" ? filters.type : undefined,
    targetEventId: filters.target_event_id || undefined,
    targetPubkey: filters.target_pubkey || undefined,
    since: filters.since || undefined,
    until: filters.until || undefined,
  }
}
