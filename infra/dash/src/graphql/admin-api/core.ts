import { adminApolloClient } from "@/graphql/client"
import { env } from "@/lib/env"

export type RequestInitLike = RequestInit | undefined

export type RequestContext = {
  method: string
  pathname: string
  search: URLSearchParams
  body: any
  init?: RequestInitLike
}

export class GraphQLApiError extends Error {
  status?: number
  details?: unknown
  requestId?: string

  constructor(message: string, status?: number, details?: unknown, requestId?: string) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.details = details
    this.requestId = requestId
  }
}

export async function buildRequestContext(path: string, init?: RequestInitLike): Promise<RequestContext> {
  const method = (init?.method ?? "GET").toUpperCase()
  const url = new URL(path, "http://admin.local")
  return {
    method,
    pathname: url.pathname,
    search: url.searchParams,
    body: await readBody(init),
    init,
  }
}

export async function graphQuery<TResult>(document: any, variables: Record<string, unknown> | undefined, map: (data: any) => TResult): Promise<TResult> {
	try {
		const result = await adminApolloClient.query({ query: document, variables })
		return map(result.data)
	} catch (error) {
		throw toApiError(error)
	}
}

export async function graphMutation<TResult>(document: any, variables: Record<string, unknown> | undefined, map: (data: any) => TResult): Promise<TResult> {
	try {
		const result = await adminApolloClient.mutate({ mutation: document, variables })
		return map(result.data)
	} catch (error) {
		throw toApiError(error)
	}
}

export async function graphUpload<TResult>(query: string, formData: FormData, map: (data: any) => TResult): Promise<TResult> {
  const operations = { query, variables: { files: Array.from(formData.getAll("files"), () => null) } }
  const mapPayload: Record<string, string[]> = {}
  const fileEntries = formData.getAll("files")
  fileEntries.forEach((_, index) => {
    mapPayload[String(index)] = [`variables.files.${index}`]
  })
  const multipartBody = new FormData()
  multipartBody.set("operations", JSON.stringify(operations))
  multipartBody.set("map", JSON.stringify(mapPayload))
  fileEntries.forEach((entry, index) => {
    multipartBody.set(String(index), entry)
  })
  const headers = new Headers()
  if (env.adminToken) {
    headers.set("X-Admin-Token", env.adminToken)
  }
  const response = await fetch(`${env.adminBaseUrl}/graphql`, { method: "POST", headers, body: multipartBody })
  const requestId = response.headers.get("x-request-id") ?? undefined
  const payload = await response.json()
  if (!response.ok || payload.errors) {
    throw new GraphQLApiError(payload.errors?.[0]?.message ?? `Falha na requisicao (${response.status})`, response.status, payload, requestId)
  }
  return map(payload.data)
}

export async function readBody(init?: RequestInitLike): Promise<any> {
  if (!init?.body || init.body instanceof FormData) {
    return undefined
  }
  if (typeof init.body === "string") {
    return JSON.parse(init.body)
  }
  return init.body
}

export function toApiError(error: any): GraphQLApiError {
  const response = error?.networkError?.response
  const requestId = response?.headers?.get?.("x-request-id") ?? undefined
  const status = error?.networkError?.statusCode ?? error?.statusCode ?? undefined
  const graphMessage = error?.errors?.[0]?.message ?? error?.message ?? "Falha na requisicao GraphQL"
  return new GraphQLApiError(graphMessage, status, error, requestId)
}

export function pageVars(search: URLSearchParams) {
  const limit = Number(search.get("limit") ?? 0)
  const offset = Number(search.get("offset") ?? 0)
  return { limit: limit || 50, offset }
}

export function eventFilterVars(search: URLSearchParams) {
  const tags = search.getAll("tag").map((tag) => {
    const [name = "", ...rest] = tag.split(":")
    return { name: name.replace(/^#/, ""), value: rest.join(":") }
  }).filter((item) => item.name && item.value)
  return {
    q: emptyToNull(search.get("q")),
    authors: search.getAll("author"),
    kinds: search.getAll("kind").map(Number).filter(Number.isFinite),
    tags: tags.length > 0 ? tags : null,
    since: numberOrNull(search.get("since")),
    until: numberOrNull(search.get("until")),
  }
}

export function reportedFilterVars(search: URLSearchParams) {
  const type = search.get("type")
  return {
    q: emptyToNull(search.get("q")),
    types: type && type !== "all" ? [type] : null,
  }
}

export function labelsFilterVars(search: URLSearchParams) {
  const targetType = labelTargetTypeVar(search.get("target_type"))
  return {
    namespace: emptyToNull(search.get("namespace")),
    labels: search.getAll("label"),
    targetType,
    target: emptyToNull(search.get("target")),
    author: emptyToNull(search.get("author")),
    q: emptyToNull(search.get("q")),
  }
}

export function jobsFilterVars(search: URLSearchParams) {
  return {
    queue: emptyToNull(search.get("queue")),
    jobName: emptyToNull(search.get("job_name")),
    statuses: search.getAll("status"),
  }
}

export function blossomObjectFilterVars(search: URLSearchParams) {
  return {
    sha256: emptyToNull(search.get("sha256")),
    mimeType: emptyToNull(search.get("mime_type")),
    extension: emptyToNull(search.get("extension")),
    reviewState: emptyToNull(search.get("review_state")),
    pubkey: emptyToNull(search.get("pubkey")),
    uploaderQuery: emptyToNull(search.get("uploader_q")),
  }
}

export function blossomUserFilterVars(search: URLSearchParams) {
  return {
    q: emptyToNull(search.get("q")),
    sortBy: emptyToNull(search.get("sort_by")),
    sortDir: search.get("sort_dir") ? String(search.get("sort_dir")).toUpperCase() : null,
  }
}

export function blossomReportFilterVars(search: URLSearchParams) {
  return {
    q: emptyToNull(search.get("q")),
    reportType: emptyToNull(search.get("report_type")),
    status: search.get("status") ? String(search.get("status")).toUpperCase() : null,
    objectHash: emptyToNull(search.get("object_hash")),
  }
}

export function bucketVar(value: string | null) {
  const normalized = value?.trim().toLowerCase()
  if (normalized === "hour") return "HOUR"
  if (normalized === "day") return "DAY"
  return undefined
}

export function labelTargetTypeVar(value: string | null) {
  const normalized = value?.trim().toLowerCase()
  if (normalized === "event") return "EVENT"
  if (normalized === "pubkey") return "PUBKEY"
  if (normalized === "address") return "ADDRESS"
  if (normalized === "reference") return "REFERENCE"
  if (normalized === "topic") return "TOPIC"
  return undefined
}

export function numberOrNull(value: string | null) {
  if (!value) return null
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

export function emptyToNull(value: string | null) {
  return value && value !== "" ? value : null
}

export function keysToSnake(value: any): any {
  if (Array.isArray(value)) return value.map(keysToSnake)
  if (!value || typeof value !== "object") return value
  const mapped: Record<string, any> = {}
  for (const [key, nested] of Object.entries(value)) {
    mapped[key.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`)] = keysToSnake(nested)
  }
  return mapped
}

export function pageFromGraph(page: any, mapItem: (item: any) => any) {
  return {
    items: (page.items ?? []).map(mapItem),
    total: page.pageInfo.total,
    limit: page.pageInfo.limit,
    offset: page.pageInfo.offset,
    has_more: page.pageInfo.hasMore,
  }
}

export function listToPage(items: any[], search: URLSearchParams) {
  const limit = Number(search.get("limit") ?? 50) || 50
  const offset = Number(search.get("offset") ?? 0) || 0
  const sliced = items.slice(offset, offset + limit)
  return { items: sliced, total: items.length, limit, offset, has_more: offset + sliced.length < items.length }
}

export function mapEventRecord(item: any) {
  return {
    id: item.id,
    pubkey: item.pubkey,
    kind: item.kind,
    created_at: item.createdAt,
    content: item.content,
    sig: item.sig,
    tags: (item.tags ?? []).map((tag: any) => tag.values),
  }
}

export function mapLabelEvent(item: any) {
  return {
    id: item.id,
    pubkey: item.pubkey,
    author_npub: item.authorNpub,
    created_at: item.createdAt,
    kind: item.kind,
    content: item.content,
    namespace: item.namespace,
    labels: item.labels,
    target: { type: String(item.target.type).toLowerCase(), value: item.target.value, relay_hint: item.target.relayHint },
    tags: (item.tags ?? []).map((tag: any) => tag.values),
  }
}

export function normalizeBlossomOverview(item: any) {
  return {
    storage: keysToSnake(item.storage),
    objects: keysToSnake(item.objects),
    traffic: keysToSnake(item.traffic),
    users: keysToSnake(item.users),
    workers: keysToSnake(item.workers),
    policy: item.policy ? normalizeBlossomPolicy(item.policy) : undefined,
    alerts: keysToSnake(item.alerts ?? []),
  }
}

export function normalizeBlossomPolicy(item: any) {
  return {
    mode: String(item.mode).toLowerCase(),
    default_storage_quota_bytes: item.defaultStorageQuotaBytes,
    default_egress_quota_bytes: item.defaultEgressQuotaBytes,
    enabled_user_default_storage_quota_bytes: item.enabledUserDefaultStorageQuotaBytes,
    enabled_user_default_egress_quota_bytes: item.enabledUserDefaultEgressQuotaBytes,
    updated_at: item.updatedAt,
  }
}

export function normalizeBlossomPlan(item: any) {
  return {
    id: item.id,
    name: item.name,
    scope: String(item.scope).toLowerCase(),
    storage_quota_bytes: item.storageQuotaBytes,
    egress_quota_bytes: item.egressQuotaBytes,
    description: item.description,
    is_default: item.isDefault,
    updated_at: item.updatedAt,
  }
}

export function normalizeBlossomObjectRecord(item: any) {
  return {
    hash: item.hash,
    uploader_pubkey: item.uploaderPubkey,
    mime_type: item.mimeType,
    extension: item.extension,
    size: item.size,
    created_at: item.createdAt,
    width: item.width,
    height: item.height,
    duration_ms: item.durationMS,
    bitrate_kbps: item.bitrateKbps,
    thumbnail_url: item.thumbnailURL,
    direct_url: item.directURL,
    optimized_url: item.optimizedURL,
    review_state: item.reviewState,
    exif_status: item.exifStatus,
    gps_detected: item.gpsDetected,
    download_count: item.downloadCount,
    last_downloaded_at: item.lastDownloadedAt,
  }
}

export function normalizeBlossomObject(item: any) {
  return {
    ...normalizeBlossomObjectRecord(item),
    ingress_bytes: item.ingressBytes,
    egress_bytes: item.egressBytes,
    mirrors: item.mirrors ?? [],
    flag_reason: item.flagReason,
    nip94_tags: Object.fromEntries(((item.nip94Tags ?? []) as any[]).map((tag) => [tag.values[0], tag.values[1] ?? ""])),
    blossom_id: item.blossomID,
    report_count: item.reportCount,
  }
}

export function normalizeBlossomUser(item: any) {
  return {
    pubkey: item.pubkey,
    display_name: item.displayName,
    picture: item.picture,
    npub: item.npub,
    object_count: item.objectCount,
    storage_used_bytes: item.storageUsedBytes,
    storage_quota_bytes: item.storageQuotaBytes,
    monthly_egress_bytes: item.monthlyEgressBytes,
    egress_quota_bytes: item.egressQuotaBytes,
    enabled: item.enabled,
    last_upload_at: item.lastUploadAt,
    notes: item.notes,
  }
}

export function normalizeBlossomReport(item: any) {
  return {
    id: item.id,
    event_id: item.eventID,
    object_hash: item.objectHash,
    reporter_pubkey: item.reporterPubkey,
    reporter_npub: item.reporterNpub,
    target_event_id: item.targetEventID,
    target_pubkey: item.targetPubkey,
    report_type: item.reportType,
    reason: item.reason,
    status: String(item.status).toLowerCase(),
    resolved_by: item.resolvedBy,
    resolved_note: item.resolvedNote,
    created_at: item.createdAt,
    resolved_at: item.resolvedAt,
  }
}

export function mapDownloadJob(item: any) {
  const payload = item?.payload ?? {}
  const result = item?.result ?? {}
  return {
    id: item.id,
    status: downloadStatus(item.status),
    message: result.message ?? item.lastError ?? undefined,
    created_at: item.createdAt,
    started_at: item.startedAt,
    finished_at: item.finishedAt,
    relays: payload.relays ?? [],
    public_key: payload.public_key ?? payload.publicKey,
    kinds: payload.kinds ?? [],
    timeout: payload.timeout ?? payload.timeout_sec ?? 0,
    filter: result.filter ?? payload.filter ?? {},
    filter_json: payload.filter_json ?? payload.filterJSON ?? "",
    summary: result.summary ?? { events_received: 0, inserted_events: 0, duplicate_events: 0, pages: 0, successful_relays: 0, failed_relays: 0 },
    relay_results: result.relay_results ?? result.relayResults ?? [],
    error: result.error ?? item.lastError ?? undefined,
  }
}

export function downloadStatus(status: string) {
  switch (status) {
    case "succeeded":
      return "completed"
    case "failed":
    case "dead":
      return "failed"
    case "running":
      return "running"
    default:
      return "queued"
  }
}

export function extractDeletedCount(message?: string | null) {
  if (!message) return 0
  const match = message.match(/deleted\s+(\d+)/i)
  return match ? Number(match[1]) : 0
}
