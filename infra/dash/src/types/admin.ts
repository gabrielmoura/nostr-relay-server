export type ConnectionRecord = {
  ws_id: string
  ip: string
  authed?: string
  subscription_count: number
  connected_at?: string
  last_seen_at?: string
  user_agent?: string
}

export type AdminPage<T> = {
  items: T[]
  total: number
  limit: number
  offset: number
  has_more: boolean
}

export type RelayMetricCard = {
  label: string
  value: string
  tone?: "default" | "success" | "danger" | "warning"
  helper?: string
}

export type RelayOverview = {
  cards: RelayMetricCard[]
  status: "operational" | "degraded"
  activeConnections: number
  authedConnections: number
  bannedUsers: number
}

export type StreamRelayStatus = {
  url: string
  connected: boolean
  failure_count: number
  last_error?: string
}

export type StreamPoolStatus = {
  initialized: boolean
  connected_relays: number
  total_relays: number
  relays: StreamRelayStatus[]
}

export type StreamDispatcherStatus = {
  started: boolean
  worker_count: number
  event_queue_len: number
  event_queue_cap: number
  request_queue_len: number
  request_queue_cap: number
  dropped_event_jobs: number
  dropped_request_jobs: number
}

export type StreamStatus = {
  config: {
    stream_up: boolean
    stream_down: boolean
    relays: string[]
  }
  dispatcher: StreamDispatcherStatus
  pool: StreamPoolStatus
  counters: {
    forwarded_events: number
    forwarded_requests: number
    forward_failures: number
  }
}

export type UserProfile = {
  pubkey: string
  npub: string
  displayName: string
  handle: string
  picture?: string
  nip05?: string
  trustScore?: number
  relayCount?: number
  followers?: number
  metadata?: string
  status?: "online" | "monitor" | "suspect" | "banned"
  reason?: string
  related_ids?: string[]
  created_at?: string
}

export type LoggedUser = UserProfile & {
  connectionCount: number
  lastSeenAt?: string
  connectionState: "stable" | "attention"
}

export type BannedUser = UserProfile & {
  reason: string
  source: "manual" | "rule"
  bannedAt: string
  durationLabel: string
  related_ids?: string[]
}

export type EventRecord = {
  id: string
  pubkey: string
  kind: number
  created_at: number
  content: string
  tags: string[][]
  sig?: string
}

export type EventIdentifiers = {
  note?: string
  nevent?: string
  npub?: string
  nprofile?: string
}

export type EventAuthor = {
  pubkey: string
  display_name: string
  picture?: string
  nip05?: string
}

export type RelayFetchStatus = {
  relay: string
  status: "found" | "not_found" | "connect_error" | "query_error"
  error?: string
}

export type EventDetail = {
  event: EventRecord
  identifiers: EventIdentifiers
  author: EventAuthor
  hashtags: string[]
  image_urls: string[]
}

export type EventAggregateKind = {
  kind: number
  count: number
}

export type EventAggregateAuthor = {
  pubkey: string
  count: number
}

export type EventAggregateTag = {
  tag: string
  count: number
}

export type EventAggregates = {
  total: number
  kinds: EventAggregateKind[]
  top_authors: EventAggregateAuthor[]
  top_tags: EventAggregateTag[]
}

export type EventTimelinePoint = {
  ts: number
  count: number
}

export type EventTimeline = {
  bucket: "hour" | "day"
  points: EventTimelinePoint[]
}

export type EventReport = {
  report_event_id: string
  reporter_pubkey: string
  reporter_npub?: string
  reporter_display_name: string
  reporter_picture?: string
  reported_event_id?: string
  reported_pubkey?: string
  report_type?: string
  content?: string
  created_at: number
}

export type EventReportsResponse = {
  items: EventReport[]
  total: number
}

export type ReportedEventItem = {
  target_event_id: string
  target_pubkey?: string
  target_nevent?: string
  target_created_at?: number
  target_created_at_iso?: string
  target_author?: EventAuthor
  report_count: number
  last_reported: number
  last_reported_at?: string
  report_types: string[]
  target_event?: EventRecord
}

export type EventSearchFilters = {
  query: string
  authors: string[]
  kinds: number[]
  tags: string[]
  limit: number
}

export type UserSearchFilters = {
  query: string
  mode: "cards" | "list" | "suspects"
}

export type BanPayload = {
  pubkey: string
  reason: string
  related_ids?: string[]
  mode?: "permanent" | "temporary"
  period_value?: number
  period_unit?: "hours" | "days"
}

export type FetchEventFromRelaysPayload = {
  event_id: string
  relays: string[]
}

export type FetchEventFromRelaysResponse = {
  event_id: string
  source_relay: string
  persisted: boolean
  relays_tried: number
  relay_results: RelayFetchStatus[]
}

export type ImportEventsFileResult = {
  filename: string
  total: number
  inserted: number
  duplicates: number
  invalid: number
  error?: string
}

export type ImportEventsResponse = {
  files: ImportEventsFileResult[]
}

export type EventSearchResponse = AdminPage<EventRecord>
