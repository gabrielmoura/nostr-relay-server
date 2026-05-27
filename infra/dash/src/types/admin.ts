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

export type NIP86PubKeyRecord = {
  pubkey: string
  reason?: string
  created_by?: string
  created_at?: string
  updated_at?: string
}

export type NIP86EventRecord = {
  event_id: string
  reason?: string
  created_by?: string
  created_at?: string
  updated_at?: string
}

export type NIP86IPRecord = {
  ip: string
  reason?: string
  created_by?: string
  created_at?: string
  updated_at?: string
}

export type NIP86RelayMetadata = {
  relay_url: string
  name?: string
  description?: string
  updated_by?: string
  updated_at?: string
}

export type NIP86ReasonPayload = {
  reason: string
}

export type NIP86RelayMetadataPayload = {
  name: string
  description: string
}

export type DownloadJobStatus = "queued" | "running" | "completed" | "failed"

export type AdminJobStatus = "unknown" | "queued" | "running" | "succeeded" | "failed" | "delayed" | "dead" | "canceled"

export type AdminJob = {
  id: string
  queue: string
  priority: string
  job_name: string
  status: AdminJobStatus
  attempts: number
  max_attempts: number
  created_at: string
  started_at?: string
  finished_at?: string
  run_at?: string
  last_error?: string
  payload?: any
  result?: any
}

export type AdminJobsResponse = AdminPage<AdminJob>

export type AdminJobsFilters = {
  queue?: string
  job_name?: string
  status?: AdminJobStatus | ""
  statuses?: AdminJobStatus[]
  limit?: number
}

export type DownloadJobSummary = {
  events_received: number
  inserted_events: number
  duplicate_events: number
  pages: number
  successful_relays: number
  failed_relays: number
}

export type DownloadJobRelayResult = {
  relay: string
  status: string
  events_received: number
  inserted_events: number
  duplicate_events: number
  pages: number
  error?: string
}

export type DownloadJob = {
  id: string
  status: DownloadJobStatus
  message?: string
  created_at: string
  started_at?: string
  finished_at?: string
  relays: string[]
  public_key?: string
  kinds?: number[]
  timeout: number
  filter: any
  filter_json: string
  summary: DownloadJobSummary
  relay_results: DownloadJobRelayResult[]
  error?: string
}

export type DownloadJobsResponse = {
  items: DownloadJob[]
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
  display_name?: string
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
  trends?: {
    top_tag_month?: string
    top_tag_month_count?: number
    top_tag_year?: string
    top_tag_year_count?: number
    peak_month?: string
    peak_month_count?: number
    peak_year?: string
    peak_year_count?: number
  }
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

export type ReportedEventsTimelinePoint = {
  bucket: string
  count: number
}

export type ReportedEventsTypeCount = {
  name: string
  count: number
}

export type ReportedEventsAuthorCount = {
  pubkey: string
  display_name?: string
  count: number
}

export type ReportedEventsTargetCount = {
  target_event_id: string
  count: number
}

export type ReportedEventsSummary = {
  total_events: number
  total_reports: number
  unique_target_authors: number
  timeline: ReportedEventsTimelinePoint[]
  report_types: ReportedEventsTypeCount[]
  top_authors: ReportedEventsAuthorCount[]
  top_targets: ReportedEventsTargetCount[]
}

export type ReportedEventsFilters = {
  query: string
  type: string
  target_pubkey?: string
  target_event_id?: string
  since?: number
  until?: number
}

export type AdminLabelTargetType = "event" | "pubkey" | "address" | "reference" | "topic"

export type AdminLabelTarget = {
  type: AdminLabelTargetType
  value: string
  relay_hint?: string
}

export type AdminLabelEvent = {
  id: string
  pubkey: string
  author_npub?: string
  created_at: number
  kind: number
  content: string
  namespace: string
  labels: string[]
  target: AdminLabelTarget
  tags: string[][]
}

export type AdminLabelCount = {
  count: number
  namespace?: string
  label?: string
  target_type?: AdminLabelTargetType
}

export type AdminLabelsSummary = {
  total_events: number
  total_targets: number
  namespaces: AdminLabelCount[]
  labels: AdminLabelCount[]
  target_types: AdminLabelCount[]
}

export type AdminLabelsFilters = {
  namespace?: string
  labels?: string[]
  target_type?: AdminLabelTargetType | ""
  target?: string
  author?: string
  q?: string
  limit?: number
}

export type CreateAdminLabelPayload = {
  namespace: string
  labels: string[]
  comment?: string
  target: AdminLabelTarget
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

export type NIP05Identity = {
  name: string
  pubkey: string
  npub?: string
  display_name?: string
  picture?: string
  relay_hints?: string[]
  created_at?: string
  updated_at?: string
}

export type NIP05IdentityPayload = {
  name: string
  pubkey: string
}

export type UserNIP05Association = {
  pubkey: string
  exists: boolean
  name?: string
  display_name?: string
  picture?: string
  relay_hints?: string[]
  created_at?: string
  updated_at?: string
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
  found: boolean
  persisted: boolean
  relays_tried: number
  relay_results: RelayFetchStatus[]
  message?: string
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

export type NegentropySyncRequest = {
  remote: string
  direction: "up" | "down" | "both"
  filter?: any[]
  timeout?: number
}

export type NegentropySyncResponse = {
  status: string
  remote: string
  message: string
  job_id?: string
}

export type DownloadEventsRequest = {
  relays: string[]
  public_key?: string
  kinds?: number[]
  filter?: any
  timeout?: number
}

export type AdminGroupResponse = {
  group_id: string
  name: string
  description: string
  private: boolean
  closed: boolean
  hidden: boolean
  member_count: number
}

export type AdminWoTSummaryResponse = {
  total_nodes: number
  total_edges: number
  trusted_pubkeys: string[]
  last_computed_at?: string
}

export type BlossomTab = "overview" | "library" | "users" | "workers"

export type BlossomLibraryView = "table" | "grid"

export type BlossomReviewState = "ready" | "flagged" | "pending_review" | "approved" | "deleted"

export type BlossomExifStatus = "pending" | "clean" | "stripped" | "rejected"

export type BlossomAlert = {
  level: "info" | "warning" | "danger"
  code: string
  message: string
}

export type BlossomMetricSummary = {
  used_bytes: number
  free_bytes: number
  used_percent: number
}

export type BlossomObjectsSummary = {
  total: number
  flagged: number
  pending_review: number
}

export type BlossomTrafficSummary = {
  monthly_ingress_bytes: number
  monthly_egress_bytes: number
}

export type BlossomUsersSummary = {
  active: number
  whitelisted: number
}

export type BlossomWorkersSummary = {
  running: number
  failed: number
}

export type BlossomOverview = {
  storage: BlossomMetricSummary
  objects: BlossomObjectsSummary
  traffic: BlossomTrafficSummary
  users: BlossomUsersSummary
  workers: BlossomWorkersSummary
  policy?: BlossomPolicy
  alerts: BlossomAlert[]
}

export type BlossomPolicyMode = "mandatory_review" | "enabled_users" | "free"

export const POLICY_MODE_LABELS: Record<BlossomPolicyMode, string> = {
  mandatory_review: "Revisao obrigatoria",
  enabled_users: "Somente habilitados",
  free: "Livre",
}

export type BlossomPolicy = {
  mode: BlossomPolicyMode
  default_storage_quota_bytes?: number
  default_egress_quota_bytes?: number
  enabled_user_default_storage_quota_bytes?: number
  enabled_user_default_egress_quota_bytes?: number
  updated_at?: string
}

export type BlossomPlanScope = "free" | "enabled_users"

export const BLOSSOM_PLAN_SCOPE_LABELS: Record<BlossomPlanScope, string> = {
  free: "Livre",
  enabled_users: "Somente habilitados",
}

export type BlossomPlan = {
  id: string
  name: string
  scope: BlossomPlanScope
  storage_quota_bytes?: number
  egress_quota_bytes?: number
  description?: string
  is_default: boolean
  updated_at?: string
}

export type BlossomPlansResponse = {
  items: BlossomPlan[]
}

export type BlossomPlanAssignPayload = {
  pubkey: string
}

export type BlossomPlanAssignResponse = {
  ok: boolean
  plan_id: string
  pubkey: string
}

export type BlossomPlanAssignment = {
  plan_id: string
  pubkey: string
  display_name?: string
  picture?: string
  npub?: string
  assigned_by: string
  assigned_at: string
}

export type BlossomObjectRecord = {
  hash: string
  uploader_pubkey: string
  mime_type: string
  extension: string
  size: number
  created_at: string
  width?: number
  height?: number
  duration_ms?: number
  bitrate_kbps?: number
  blurhash?: string
  thumbnail_url?: string
  direct_url: string
  optimized_url?: string
  review_state: BlossomReviewState
  exif_status: BlossomExifStatus
  gps_detected: boolean
  download_count: number
  last_downloaded_at?: string
}

export type BlossomObjectDetail = BlossomObjectRecord & {
  ingress_bytes: number
  egress_bytes: number
  mirrors: string[]
  flag_reason?: string
  nip94_tags: Record<string, string>
  blossom_id?: string
  report_count?: number
}

export type BlossomObjectsFilters = {
  q?: string
  sha256?: string
  mime_type?: string
  extension?: string
  review_state?: BlossomReviewState | ""
  pubkey?: string
  uploader_q?: string
}

export type BlossomBulkReviewAction = "approve" | "hard_delete" | "requeue_optimization"

export type BlossomBulkReviewPayload = {
  hashes: string[]
  action: BlossomBulkReviewAction
  reason?: string
}

export type BlossomBulkReviewResponse = {
  ok: boolean
  updated: number
}

export type BlossomUserRecord = {
  pubkey: string
  display_name?: string
  picture?: string
  npub?: string
  object_count: number
  storage_used_bytes: number
  storage_quota_bytes?: number
  monthly_egress_bytes: number
  egress_quota_bytes?: number
  enabled: boolean
  last_upload_at?: string
  notes?: string
}

export type BlossomUsersFilters = {
  q?: string
  sort_by?: string
  sort_dir?: "asc" | "desc"
}

export type BlossomUserDetail = BlossomUserRecord & {
  files: BlossomObjectRecord[]
}

export type BlossomWhitelistPayload = {
  pubkey: string
  enabled: boolean
  storage_quota_bytes?: number
  egress_quota_bytes?: number
  notes?: string
}

export type BlossomMirrorPayload = {
  source_url: string
  expected_sha256: string
}

export type BlossomMirrorResponse = {
  ok: boolean
  job_id: string
  status: string
}

export type BlossomWorkerRecord = {
  job_id: string
  job_type: string
  status: AdminJobStatus
  target_hash?: string
  detail: string
  progress_label?: string
  created_at: string
  updated_at: string
}

export type BlossomWorkersFilters = {
  status?: AdminJobStatus | ""
  job_type?: string
  target_hash?: string
}

export type BlossomReportStatus = "open" | "dismissed" | "actioned"

export type BlossomReportRecord = {
  id: string
  event_id: string
  object_hash: string
  reporter_pubkey: string
  reporter_npub?: string
  target_event_id?: string
  target_pubkey?: string
  report_type?: string
  reason?: string
  status: BlossomReportStatus
  resolved_by?: string
  resolved_note?: string
  created_at?: string
  resolved_at?: string
}

export type BlossomReportsFilters = {
  q?: string
  report_type?: string
  status?: BlossomReportStatus | ""
  object_hash?: string
}

export type BlossomResolveReportPayload = {
  id: string
  status: Exclude<BlossomReportStatus, "open">
  note?: string
}

export type BlossomCountByValue = {
  name: string
  count: number
}

export type BlossomAnalytics = {
  reports: {
    total: number
    open: number
    resolved: number
    by_type: BlossomCountByValue[]
    by_status: BlossomCountByValue[]
  }
  objects: {
    by_mime: BlossomCountByValue[]
    by_review_state: BlossomCountByValue[]
  }
  workers: {
    by_status: BlossomCountByValue[]
    by_type: BlossomCountByValue[]
  }
}

export type BlossomAuditRecord = {
  id: string
  actor_pubkey: string
  action: string
  target_type: string
  target_id: string
  created_at: string
  request_id?: string
  nostr_event_id?: string
  payload?: Record<string, string>
}

export type BlossomAuditFilters = {
  q?: string
}
