import type { AdminJobStatus, BlossomExifStatus, BlossomLibraryView, BlossomReviewState, BlossomTab } from "@/types/admin"

export type BlossomRouteSearch = {
  tab?: BlossomTab
  view?: BlossomLibraryView
  q?: string
  sha256?: string
  mimeType?: string
  extension?: string
  reviewState?: BlossomReviewState
  pubkey?: string
  uploaderQuery?: string
  userQuery?: string
  userSortBy?: string
  userSortDir?: "asc" | "desc"
  reportQuery?: string
  reportType?: string
  reportStatus?: "open" | "dismissed" | "actioned"
  auditQuery?: string
  workerStatus?: AdminJobStatus
}

export function normalizeBlossomSearch(search: Record<string, unknown>): BlossomRouteSearch {
  return {
    tab: normalizeEnum(search.tab, ["overview", "library", "users", "workers"]),
    view: normalizeEnum(search.view, ["table", "grid"]),
    q: normalizeString(search.q),
    sha256: normalizeString(search.sha256),
    mimeType: normalizeString(search.mimeType),
    extension: normalizeString(search.extension),
    reviewState: normalizeEnum(search.reviewState, ["ready", "flagged", "pending_review", "approved", "deleted"]),
    pubkey: normalizeString(search.pubkey),
    uploaderQuery: normalizeString(search.uploaderQuery),
    userQuery: normalizeString(search.userQuery),
    userSortBy: normalizeString(search.userSortBy),
    userSortDir: normalizeEnum(search.userSortDir, ["asc", "desc"]),
    reportQuery: normalizeString(search.reportQuery),
    reportType: normalizeString(search.reportType),
    reportStatus: normalizeEnum(search.reportStatus, ["open", "dismissed", "actioned"]),
    auditQuery: normalizeString(search.auditQuery),
    workerStatus: normalizeEnum(search.workerStatus, ["unknown", "queued", "running", "succeeded", "failed", "delayed", "dead", "canceled"]),
  }
}

export function blossomReviewVariant(value: BlossomReviewState) {
  switch (value) {
    case "approved":
      return "success" as const
    case "flagged":
    case "pending_review":
      return "warning" as const
    case "deleted":
      return "danger" as const
    default:
      return "default" as const
  }
}

export function blossomExifVariant(value: BlossomExifStatus) {
  switch (value) {
    case "clean":
    case "stripped":
      return "success" as const
    case "rejected":
      return "danger" as const
    default:
      return "warning" as const
  }
}

function normalizeString(value: unknown) {
  const text = typeof value === "string" ? value.trim() : ""
  return text || undefined
}

function normalizeEnum<T extends string>(value: unknown, allowed: T[]) {
  const text = normalizeString(value)
  if (!text) {
    return undefined
  }
  return allowed.includes(text as T) ? (text as T) : undefined
}
