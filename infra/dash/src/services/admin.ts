import type {
  AdminPage,
  BanPayload,
  BannedUser,
  ConnectionRecord,
  EventRecord,
  EventSearchFilters,
  EventSearchResponse,
  EventAggregates,
  EventDetail,
  FetchEventFromRelaysResponse,
  ImportEventsResponse,
  LoggedUser,
  NIP86EventRecord,
  NIP86IPRecord,
  NIP86PubKeyRecord,
  NIP86ReasonPayload,
  NIP86RelayMetadata,
  NIP86RelayMetadataPayload,
  NIP05Identity,
  NIP05IdentityPayload,
  EventReportsResponse,
  EventTimeline,
  ReportedEventItem,
  RelayOverview,
  StreamStatus,
  UserNIP05Association,
  UserProfile,
  NegentropySyncRequest,
  DownloadEventsRequest,
  DownloadJobsResponse,
  DownloadJob,
  AdminGroupResponse,
  AdminWoTSummaryResponse,
} from "@/types/admin"
import { env } from "@/lib/env"
import { buildNpubLike, formatCount, toTitleCase } from "@/lib/utils"
import { toNevent, toNote, toNprofile, toNpub } from "@/lib/nostr"
import { mockConnections, mockEvents, mockUsers, seedBannedUsers } from "@/mocks/admin"

export class ApiError extends Error {
  status?: number
  details?: unknown

  constructor(message: string, status?: number, details?: unknown) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.details = details
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)

  if (env.adminToken) {
    headers.set("X-Admin-Token", env.adminToken)
  }

  if (init?.body && !(init.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }

  const response = await fetch(`${env.adminBaseUrl}${path}`, {
    ...init,
    headers,
  })

  if (!response.ok) {
    let message = `Falha na requisicao (${response.status})`
    let details: unknown
    try {
      const body = (await response.json()) as { error?: string }
      details = body
      if (body.error) {
        message = body.error
      }
    } catch {
      // ignore
    }
    throw new ApiError(message, response.status, details)
  }

  return response.json() as Promise<T>
}

function fallbackPage<T>(items: T[], limit: number, offset: number): AdminPage<T> {
  const sliced = items.slice(offset, offset + limit)
  return {
    items: sliced,
    total: items.length,
    limit,
    offset,
    has_more: offset + sliced.length < items.length,
  }
}

function isMockEnabled() {
  return env.mockOnFailure
}

function normalizeProfile(input: Record<string, unknown>): UserProfile {
  const pubkey = String(input.pubkey ?? input.public_key ?? "")
  const displayName = String(input.displayName ?? input.display_name ?? input.name ?? pubkey)
  const handleValue = input.handle ?? input.name ?? displayName

  return {
    pubkey,
    npub: String(input.npub ?? (pubkey ? buildNpubLike(pubkey) : "")),
    displayName,
    handle: String(handleValue ? `@${String(handleValue).replace(/^@/, "")}` : `@${toTitleCase(pubkey.slice(0, 8))}`),
    picture: input.picture ? String(input.picture) : undefined,
    nip05: input.nip05 ? String(input.nip05) : undefined,
    metadata: input.metadata ? String(input.metadata) : input.about ? String(input.about) : undefined,
    status: (input.status as UserProfile["status"]) ?? undefined,
    reason: input.reason ? String(input.reason) : undefined,
    related_ids: Array.isArray(input.related_ids) ? input.related_ids.map((item) => String(item)) : undefined,
    created_at: input.created_at ? String(input.created_at) : undefined,
    trustScore: typeof input.trustScore === "number" ? input.trustScore : typeof input.trust_score === "number" ? input.trust_score : undefined,
    relayCount: typeof input.relayCount === "number" ? input.relayCount : typeof input.relay_count === "number" ? input.relay_count : undefined,
    followers: typeof input.followers === "number" ? input.followers : undefined,
  }
}

type PageParams = {
  limit: number
  offset: number
}

export async function getRelayOverview(): Promise<RelayOverview> {
  try {
    const payload = await request<{
      active_connections: number
      authed_connections: number
      logged_users: number
      banned_users: number
      indexed_events: number
      events_per_minute: number
      relay_status: string
    }>("/overview")

    return {
      activeConnections: payload.active_connections,
      authedConnections: payload.authed_connections,
      bannedUsers: payload.banned_users,
      status: payload.relay_status === "operational" ? "operational" : "degraded",
      cards: [
        { label: "Conexoes ativas", value: formatCount(payload.active_connections) },
        { label: "Conexoes logadas", value: formatCount(payload.authed_connections), tone: "success" },
        { label: "Usuarios banidos", value: formatCount(payload.banned_users), tone: "danger" },
        { label: "Eventos indexados", value: formatCount(payload.indexed_events) },
        { label: "Eventos / min", value: formatCount(payload.events_per_minute) },
        { label: "Status do relay", value: payload.relay_status, tone: payload.relay_status === "operational" ? "success" : "warning" },
      ],
    }
  } catch (error) {
    if (!isMockEnabled()) {
      throw error
    }

    return {
      activeConnections: mockConnections.length,
      authedConnections: mockConnections.filter((connection) => Boolean(connection.authed)).length,
      bannedUsers: seedBannedUsers.length,
      status: "operational",
      cards: [
        { label: "Conexoes ativas", value: formatCount(mockConnections.length) },
        { label: "Conexoes logadas", value: formatCount(mockConnections.filter((connection) => Boolean(connection.authed)).length), tone: "success" },
        { label: "Usuarios banidos", value: formatCount(seedBannedUsers.length), tone: "danger" },
        { label: "Eventos indexados", value: "42,3M" },
        { label: "Eventos / min", value: "2.418" },
        { label: "Status do relay", value: "operational", tone: "success" },
      ],
    }
  }
}

export async function getStreamStatus(): Promise<StreamStatus> {
  try {
    return await request<StreamStatus>("/stream/status")
  } catch (error) {
    if (!isMockEnabled()) {
      throw error
    }

    return {
      config: {
        stream_up: true,
        stream_down: false,
        relays: ["wss://relay.damus.io", "wss://nos.lol"],
      },
      dispatcher: {
        started: true,
        worker_count: 2,
        event_queue_len: 0,
        event_queue_cap: 1024,
        request_queue_len: 1,
        request_queue_cap: 256,
        dropped_event_jobs: 0,
        dropped_request_jobs: 0,
      },
      pool: {
        initialized: true,
        connected_relays: 1,
        total_relays: 2,
        relays: [
          { url: "wss://relay.damus.io", connected: true, failure_count: 0 },
          { url: "wss://nos.lol", connected: false, failure_count: 2, last_error: "dial timeout" },
        ],
      },
      counters: {
        forwarded_events: 124,
        forwarded_requests: 38,
        forward_failures: 3,
      },
    }
  }
}

export async function getConnectionsPage(mode: "active" | "authed", params: PageParams) {
  try {
    return await request<AdminPage<ConnectionRecord>>(`/${mode === "active" ? "connections/active" : "connections/authed"}?limit=${params.limit}&offset=${params.offset}`)
  } catch (error) {
    if (!isMockEnabled()) {
      throw error
    }

    const source = mode === "active" ? mockConnections : mockConnections.filter((connection) => Boolean(connection.authed))
    return fallbackPage(source, params.limit, params.offset)
  }
}

export async function disconnectConnection(wsid: string) {
  return request<{ ws_id: string; disconnected: boolean }>(`/connections/${wsid}/disconnect`, {
    method: "POST",
    body: JSON.stringify({ reason: "manual moderation" }),
  })
}

export async function getLoggedUsersPage(params: PageParams) {
  try {
    const page = await request<AdminPage<LoggedUser & { connection_count?: number; last_seen_at?: string }>>(`/users/logged?limit=${params.limit}&offset=${params.offset}`)
    return {
      ...page,
      items: page.items.map((item) => ({
        ...item,
        connectionCount: item.connectionCount ?? item.connection_count,
        lastSeenAt: item.lastSeenAt ?? item.last_seen_at,
      })),
    }
  } catch (error) {
    if (!isMockEnabled()) {
      throw error
    }

    const users = mockUsers.slice(0, 3).map<LoggedUser>((user, index) => ({
      ...user,
      status: "online",
      connectionCount: Math.max(1, 4 - index),
      lastSeenAt: new Date(Date.now() - index * 120000).toISOString(),
      connectionState: index === 0 ? "stable" : "attention",
    }))
    return fallbackPage(users, params.limit, params.offset)
  }
}

export async function getBannedUsersPage(query: string, params: PageParams) {
  try {
    const search = new URLSearchParams({ limit: String(params.limit), offset: String(params.offset) })
    if (query) {
      search.set("q", query)
    }
    const page = await request<AdminPage<Record<string, unknown>>>(`/users/banned?${search.toString()}`)
    return {
      ...page,
      items: page.items.map((item) => normalizeProfile(item) as BannedUser),
    }
  } catch (error) {
    if (!isMockEnabled()) {
      throw error
    }

    const source = seedBannedUsers.filter((item) => {
      if (!query) {
        return true
      }
      const needle = query.toLowerCase()
      return [item.displayName, item.handle, item.reason, item.npub].some((value) => value?.toLowerCase().includes(needle))
    })
    return fallbackPage(source, params.limit, params.offset)
  }
}

export async function getBanStatus(pubkey: string) {
  return request<{ pubkey: string; banned: boolean; reason?: string; related_ids?: string[]; created_at?: string }>(`/users/${pubkey}/ban`)
}

export async function banUser(payload: BanPayload) {
  return request(`/users/${payload.pubkey}/ban`, {
    method: "POST",
    body: JSON.stringify({
      reason: payload.reason,
      related_ids: payload.related_ids ?? [],
      mode: payload.mode,
      period_value: payload.period_value,
      period_unit: payload.period_unit,
    }),
  })
}

export async function unbanUser(pubkey: string) {
  return request(`/users/${pubkey}/ban`, { method: "DELETE" })
}

export async function searchEventsPage(filters: EventSearchFilters, params: PageParams) {
  const search = new URLSearchParams()
  if (filters.query) {
    search.set("q", filters.query)
  }
  for (const author of filters.authors) {
    search.append("author", author)
  }
  for (const kind of filters.kinds) {
    search.append("kind", String(kind))
  }
  for (const tag of filters.tags) {
    search.append("tag", tag)
  }
  search.set("limit", String(params.limit || filters.limit))
  search.set("offset", String(params.offset))

  try {
    return await request<EventSearchResponse>(`/events/search?${search.toString()}`)
  } catch (error) {
    if (!isMockEnabled()) {
      throw error
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
  const search = new URLSearchParams()
  if (filters.query) {
    search.set("q", filters.query)
  }
  for (const author of filters.authors) {
    search.append("author", author)
  }
  for (const kind of filters.kinds) {
    search.append("kind", String(kind))
  }
  for (const tag of filters.tags) {
    search.append("tag", tag)
  }

  return request<EventAggregates>(`/events/search/aggregates?${search.toString()}`)
}

export async function getEventSearchTimeline(filters: EventSearchFilters, bucket: "hour" | "day") {
  const search = new URLSearchParams()
  if (filters.query) {
    search.set("q", filters.query)
  }
  for (const author of filters.authors) {
    search.append("author", author)
  }
  for (const kind of filters.kinds) {
    search.append("kind", String(kind))
  }
  for (const tag of filters.tags) {
    search.append("tag", tag)
  }
  search.set("bucket", bucket)

  return request<EventTimeline>(`/events/search/timeline?${search.toString()}`)
}

export async function getEventDetail(eventID: string) {
  const response = await request<EventDetail>(`/events/${eventID}`)
  const pubkey = response.event?.pubkey ?? response.author?.pubkey ?? ""

  return {
    ...response,
    identifiers: {
      note: response.identifiers?.note ?? toNote(response.event.id),
      nevent: response.identifiers?.nevent ?? toNevent(response.event.id, pubkey),
      npub: response.identifiers?.npub ?? toNpub(pubkey),
      nprofile: response.identifiers?.nprofile ?? toNprofile(pubkey),
    },
  }
}

export async function fetchEventFromRelays(eventID: string, relays: string[]) {
  return request<FetchEventFromRelaysResponse>(`/events/${eventID}/fetch`, {
    method: "POST",
    body: JSON.stringify({ relays }),
  })
}

export async function importEventsFiles(files: File[]) {
  const formData = new FormData()
  for (const file of files) {
    formData.append("files", file)
  }
  return request<ImportEventsResponse>("/events/import", {
    method: "POST",
    body: formData,
  })
}

export async function getEventReports(eventID: string, params: PageParams) {
  return request<EventReportsResponse>(`/events/${eventID}/reports?limit=${params.limit}&offset=${params.offset}`)
}

export async function getReportedEventsPage(query: string, type: string, params: PageParams) {
  const search = new URLSearchParams({ limit: String(params.limit), offset: String(params.offset) })
  if (query) {
    search.set("q", query)
  }
  if (type) {
    search.set("type", type)
  }

  return request<AdminPage<ReportedEventItem>>(`/events/reported?${search.toString()}`)
}

export async function searchUsersPage(query: string, params: PageParams) {
  try {
    const search = new URLSearchParams({ limit: String(params.limit), offset: String(params.offset) })
    if (query) {
      search.set("q", query)
    }
    const page = await request<AdminPage<Record<string, unknown>>>(`/users/search?${search.toString()}`)
    return {
      ...page,
      items: page.items.map((item) => normalizeProfile(item)),
    }
  } catch (error) {
    if (!isMockEnabled()) {
      throw error
    }

    const filtered = mockUsers.filter((user) => {
      if (!query) {
        return true
      }
      const needle = query.toLowerCase()
      return [user.displayName, user.handle, user.npub, user.nip05, user.metadata]
        .filter(Boolean)
        .some((value) => value!.toLowerCase().includes(needle))
    })
    return fallbackPage(filtered, params.limit, params.offset)
  }
}

export async function getUser(pubkey: string) {
  try {
    const response = await request<Record<string, unknown>>(`/users/${pubkey}/profile`)
    return normalizeProfile(response)
  } catch (error) {
    if (!isMockEnabled()) {
      throw error
    }
    const fallback = mockUsers.find((user) => user.pubkey === pubkey)
    if (!fallback) {
      throw error
    }
    return fallback
  }
}

export async function getNIP05Page(query: string, params: PageParams) {
  const search = new URLSearchParams({ limit: String(params.limit), offset: String(params.offset) })
  if (query) {
    search.set("q", query)
  }
  return request<AdminPage<NIP05Identity>>(`/nip05?${search.toString()}`)
}

export async function upsertNIP05Identity(payload: NIP05IdentityPayload) {
  return request<NIP05Identity>("/nip05", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export async function deleteNIP05Identity(name: string) {
  return request<{ name: string; deleted: boolean }>(`/nip05/${encodeURIComponent(name)}`, {
    method: "DELETE",
  })
}

export async function getUserNIP05(pubkey: string) {
  return request<UserNIP05Association>(`/users/${pubkey}/nip05`)
}

export async function getNIP86AllowedPubKeysPage(query: string, params: PageParams) {
  const search = new URLSearchParams({ limit: String(params.limit), offset: String(params.offset) })
  if (query) {
    search.set("q", query)
  }
  return request<AdminPage<NIP86PubKeyRecord>>(`/nip86/allowed-pubkeys?${search.toString()}`)
}

export async function allowNIP86PubKey(pubkey: string, payload: NIP86ReasonPayload) {
  return request<{ ok: boolean }>(`/nip86/allowed-pubkeys/${encodeURIComponent(pubkey)}`, {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export async function unallowNIP86PubKey(pubkey: string) {
  return request<{ ok: boolean }>(`/nip86/allowed-pubkeys/${encodeURIComponent(pubkey)}`, { method: "DELETE" })
}

export async function getNIP86BlockedIPsPage(query: string, params: PageParams) {
  const search = new URLSearchParams({ limit: String(params.limit), offset: String(params.offset) })
  if (query) {
    search.set("q", query)
  }
  return request<AdminPage<NIP86IPRecord>>(`/nip86/blocked-ips?${search.toString()}`)
}

export async function blockNIP86IP(ip: string, payload: NIP86ReasonPayload) {
  return request<{ ok: boolean }>(`/nip86/blocked-ips/${encodeURIComponent(ip)}`, {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export async function unblockNIP86IP(ip: string) {
  return request<{ ok: boolean }>(`/nip86/blocked-ips/${encodeURIComponent(ip)}`, { method: "DELETE" })
}

export async function getNIP86BannedEventsPage(query: string, params: PageParams) {
  const search = new URLSearchParams({ limit: String(params.limit), offset: String(params.offset) })
  if (query) {
    search.set("q", query)
  }
  return request<AdminPage<NIP86EventRecord>>(`/nip86/banned-events?${search.toString()}`)
}

export async function banNIP86Event(eventID: string, payload: NIP86ReasonPayload) {
  return request<{ ok: boolean }>(`/nip86/banned-events/${encodeURIComponent(eventID)}`, {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export async function unbanNIP86Event(eventID: string) {
  return request<{ ok: boolean }>(`/nip86/banned-events/${encodeURIComponent(eventID)}`, { method: "DELETE" })
}

export async function getNIP86RelayMetadata() {
  return request<NIP86RelayMetadata>("/nip86/relay-metadata")
}

export async function updateNIP86RelayMetadata(payload: NIP86RelayMetadataPayload) {
  return request<{ updated: boolean; name: string; description: string }>("/nip86/relay-metadata", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export function getEventTags(event: EventRecord) {
  return event.tags.filter((entry) => entry[0] === "t").map((entry) => entry[1])
}

export async function startNegentropySync(payload: NegentropySyncRequest) {
  return request<{ status: string; remote: string; message: string }>("/sync/negentropy", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export async function startDownloadEvents(payload: DownloadEventsRequest) {
  return request<{ status: string; job_id: string; relays: string[]; message: string }>("/events/download", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export async function getDownloadJobs() {
  return request<DownloadJobsResponse>("/events/download/jobs")
}

export async function getDownloadJob(jobID: string) {
  return request<DownloadJob>(`/events/download/jobs/${encodeURIComponent(jobID)}`)
}

export async function getGroupsPage(params: PageParams) {
  return request<AdminPage<AdminGroupResponse>>(`/groups?limit=${params.limit}&offset=${params.offset}`)
}

export async function getWoTSummary() {
  return request<AdminWoTSummaryResponse>("/wot/summary")
}

export async function addTrustedPubkey(pubkey: string) {
  return request<{ pubkey: string; added: boolean }>("/wot/trusted", {
    method: "POST",
    body: JSON.stringify({ pubkey }),
  })
}

export async function removeTrustedPubkey(pubkey: string) {
  return request<{ pubkey: string; removed: boolean }>(`/wot/trusted/${pubkey}`, {
    method: "DELETE",
  })
}
