import type { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  Int64: { input: number; output: number; }
  JSON: { input: Record<string, unknown>; output: Record<string, unknown>; }
  Upload: { input: File; output: File; }
};

export type AdminAsyncJob = {
  __typename: 'AdminAsyncJob';
  jobId?: Maybe<Scalars['ID']['output']>;
  message?: Maybe<Scalars['String']['output']>;
  relays: Array<Scalars['String']['output']>;
  remote?: Maybe<Scalars['String']['output']>;
  status: Scalars['String']['output'];
};

export type AdminBanStatus = {
  __typename: 'AdminBanStatus';
  banned: Scalars['Boolean']['output'];
  createdAt?: Maybe<Scalars['String']['output']>;
  pubkey: Scalars['String']['output'];
  reason?: Maybe<Scalars['String']['output']>;
  relatedIds: Array<Scalars['String']['output']>;
};

export type AdminBanUserInput = {
  reason?: InputMaybe<Scalars['String']['input']>;
  relatedIds?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type AdminBlossomAuditPage = {
  __typename: 'AdminBlossomAuditPage';
  items: Array<Scalars['JSON']['output']>;
  pageInfo: PageInfo;
};

export type AdminBlossomBulkReviewInput = {
  action: BlossomReviewAction;
  hashes: Array<Scalars['ID']['input']>;
  reason?: InputMaybe<Scalars['String']['input']>;
};

export type AdminBlossomMirrorInput = {
  expectedSHA256: Scalars['String']['input'];
  sourceURL: Scalars['String']['input'];
};

export type AdminBlossomObject = {
  __typename: 'AdminBlossomObject';
  bitrateKbps?: Maybe<Scalars['Int']['output']>;
  blossomID?: Maybe<Scalars['String']['output']>;
  createdAt: Scalars['String']['output'];
  directURL: Scalars['String']['output'];
  downloadCount: Scalars['Int64']['output'];
  durationMS?: Maybe<Scalars['Int64']['output']>;
  egressBytes: Scalars['Int64']['output'];
  exifStatus: Scalars['String']['output'];
  extension: Scalars['String']['output'];
  flagReason?: Maybe<Scalars['String']['output']>;
  gpsDetected: Scalars['Boolean']['output'];
  hash: Scalars['ID']['output'];
  height?: Maybe<Scalars['Int']['output']>;
  ingressBytes: Scalars['Int64']['output'];
  mimeType: Scalars['String']['output'];
  mirrors: Array<Scalars['String']['output']>;
  nip94Tags: Array<NostrTag>;
  optimizedURL?: Maybe<Scalars['String']['output']>;
  reportCount: Scalars['Int64']['output'];
  reviewState: Scalars['String']['output'];
  size: Scalars['Int64']['output'];
  thumbnailURL?: Maybe<Scalars['String']['output']>;
  uploaderPubkey: Scalars['String']['output'];
  width?: Maybe<Scalars['Int']['output']>;
};

export type AdminBlossomObjectFilterInput = {
  extension?: InputMaybe<Scalars['String']['input']>;
  mimeType?: InputMaybe<Scalars['String']['input']>;
  pubkey?: InputMaybe<Scalars['String']['input']>;
  reviewState?: InputMaybe<Scalars['String']['input']>;
  sha256?: InputMaybe<Scalars['String']['input']>;
  uploaderQuery?: InputMaybe<Scalars['String']['input']>;
};

export type AdminBlossomObjectPage = {
  __typename: 'AdminBlossomObjectPage';
  items: Array<AdminBlossomObject>;
  pageInfo: PageInfo;
};

export type AdminBlossomOverview = {
  __typename: 'AdminBlossomOverview';
  alerts: Array<Scalars['JSON']['output']>;
  objects: Scalars['JSON']['output'];
  policy: AdminBlossomPolicy;
  storage: Scalars['JSON']['output'];
  traffic: Scalars['JSON']['output'];
  users: Scalars['JSON']['output'];
  workers: Scalars['JSON']['output'];
};

export type AdminBlossomPlan = {
  __typename: 'AdminBlossomPlan';
  description?: Maybe<Scalars['String']['output']>;
  egressQuotaBytes?: Maybe<Scalars['Int64']['output']>;
  id: Scalars['ID']['output'];
  isDefault: Scalars['Boolean']['output'];
  name: Scalars['String']['output'];
  scope: BlossomPlanScope;
  storageQuotaBytes?: Maybe<Scalars['Int64']['output']>;
  updatedAt?: Maybe<Scalars['String']['output']>;
};

export type AdminBlossomPlanAssignment = {
  __typename: 'AdminBlossomPlanAssignment';
  assignedAt: Scalars['String']['output'];
  assignedBy: Scalars['String']['output'];
  displayName?: Maybe<Scalars['String']['output']>;
  npub?: Maybe<Scalars['String']['output']>;
  picture?: Maybe<Scalars['String']['output']>;
  planId: Scalars['ID']['output'];
  pubkey: Scalars['String']['output'];
};

export type AdminBlossomPlanAssignmentPage = {
  __typename: 'AdminBlossomPlanAssignmentPage';
  items: Array<AdminBlossomPlanAssignment>;
  pageInfo: PageInfo;
};

export type AdminBlossomPlanInput = {
  description?: InputMaybe<Scalars['String']['input']>;
  egressQuotaBytes?: InputMaybe<Scalars['Int64']['input']>;
  id: Scalars['ID']['input'];
  isDefault: Scalars['Boolean']['input'];
  name: Scalars['String']['input'];
  scope: BlossomPlanScope;
  storageQuotaBytes?: InputMaybe<Scalars['Int64']['input']>;
};

export type AdminBlossomPolicy = {
  __typename: 'AdminBlossomPolicy';
  defaultEgressQuotaBytes?: Maybe<Scalars['Int64']['output']>;
  defaultStorageQuotaBytes?: Maybe<Scalars['Int64']['output']>;
  enabledUserDefaultEgressQuotaBytes?: Maybe<Scalars['Int64']['output']>;
  enabledUserDefaultStorageQuotaBytes?: Maybe<Scalars['Int64']['output']>;
  mode: Scalars['String']['output'];
  updatedAt?: Maybe<Scalars['String']['output']>;
};

export type AdminBlossomPolicyInput = {
  mode: Scalars['String']['input'];
};

export type AdminBlossomReport = {
  __typename: 'AdminBlossomReport';
  createdAt?: Maybe<Scalars['String']['output']>;
  eventID: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  objectHash: Scalars['String']['output'];
  reason?: Maybe<Scalars['String']['output']>;
  reportType?: Maybe<Scalars['String']['output']>;
  reporterNpub?: Maybe<Scalars['String']['output']>;
  reporterPubkey: Scalars['String']['output'];
  resolvedAt?: Maybe<Scalars['String']['output']>;
  resolvedBy?: Maybe<Scalars['String']['output']>;
  resolvedNote?: Maybe<Scalars['String']['output']>;
  status: BlossomReportStatus;
  targetEventID?: Maybe<Scalars['String']['output']>;
  targetPubkey?: Maybe<Scalars['String']['output']>;
};

export type AdminBlossomReportFilterInput = {
  objectHash?: InputMaybe<Scalars['String']['input']>;
  q?: InputMaybe<Scalars['String']['input']>;
  reportType?: InputMaybe<Scalars['String']['input']>;
  status?: InputMaybe<BlossomReportStatus>;
};

export type AdminBlossomReportPage = {
  __typename: 'AdminBlossomReportPage';
  items: Array<AdminBlossomReport>;
  pageInfo: PageInfo;
};

export type AdminBlossomResolveReportInput = {
  note?: InputMaybe<Scalars['String']['input']>;
  status: BlossomReportStatus;
};

export type AdminBlossomUser = {
  __typename: 'AdminBlossomUser';
  displayName?: Maybe<Scalars['String']['output']>;
  egressQuotaBytes?: Maybe<Scalars['Int64']['output']>;
  enabled: Scalars['Boolean']['output'];
  lastUploadAt?: Maybe<Scalars['String']['output']>;
  monthlyEgressBytes: Scalars['Int64']['output'];
  notes?: Maybe<Scalars['String']['output']>;
  npub?: Maybe<Scalars['String']['output']>;
  objectCount: Scalars['Int64']['output'];
  picture?: Maybe<Scalars['String']['output']>;
  pubkey: Scalars['String']['output'];
  storageQuotaBytes?: Maybe<Scalars['Int64']['output']>;
  storageUsedBytes: Scalars['Int64']['output'];
};

export type AdminBlossomUserDetail = {
  __typename: 'AdminBlossomUserDetail';
  files: Array<AdminBlossomObject>;
  user: AdminBlossomUser;
};

export type AdminBlossomUserFilterInput = {
  q?: InputMaybe<Scalars['String']['input']>;
  sortBy?: InputMaybe<Scalars['String']['input']>;
  sortDir?: InputMaybe<SortDirection>;
};

export type AdminBlossomUserPage = {
  __typename: 'AdminBlossomUserPage';
  items: Array<AdminBlossomUser>;
  pageInfo: PageInfo;
};

export type AdminBlossomWhitelistInput = {
  egressQuotaBytes?: InputMaybe<Scalars['Int64']['input']>;
  enabled: Scalars['Boolean']['input'];
  notes?: InputMaybe<Scalars['String']['input']>;
  pubkey: Scalars['String']['input'];
  storageQuotaBytes?: InputMaybe<Scalars['Int64']['input']>;
};

export type AdminBlossomWorkerPage = {
  __typename: 'AdminBlossomWorkerPage';
  items: Array<Scalars['JSON']['output']>;
  pageInfo: PageInfo;
};

export type AdminConnection = {
  __typename: 'AdminConnection';
  authed?: Maybe<Scalars['String']['output']>;
  connectedAt?: Maybe<Scalars['String']['output']>;
  ip: Scalars['String']['output'];
  lastSeenAt?: Maybe<Scalars['String']['output']>;
  subscriptionCount: Scalars['Int']['output'];
  userAgent?: Maybe<Scalars['String']['output']>;
  wsid: Scalars['ID']['output'];
};

export type AdminConnectionPage = {
  __typename: 'AdminConnectionPage';
  items: Array<AdminConnection>;
  pageInfo: PageInfo;
};

export type AdminCreateLabelInput = {
  comment?: InputMaybe<Scalars['String']['input']>;
  labels: Array<Scalars['String']['input']>;
  namespace: Scalars['String']['input'];
  target: AdminLabelTargetInput;
};

export type AdminCreateLabelPayload = {
  __typename: 'AdminCreateLabelPayload';
  event: NostrEvent;
  stored: Scalars['Boolean']['output'];
};

export type AdminDeleteJobsHistoryInput = {
  jobName: Scalars['String']['input'];
  statuses?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type AdminDownloadEventsInput = {
  filter?: InputMaybe<Scalars['JSON']['input']>;
  kinds?: InputMaybe<Array<Scalars['Int']['input']>>;
  publicKey?: InputMaybe<Scalars['String']['input']>;
  relays: Array<Scalars['String']['input']>;
  timeoutSeconds?: InputMaybe<Scalars['Int']['input']>;
};

export type AdminEventAggregates = {
  __typename: 'AdminEventAggregates';
  kinds: Array<KindCount>;
  topAuthors: Array<PubkeyCount>;
  topTags: Array<TagCount>;
  total: Scalars['Int64']['output'];
  trends: Scalars['JSON']['output'];
};

export type AdminEventDetail = {
  __typename: 'AdminEventDetail';
  author: Scalars['JSON']['output'];
  event: NostrEvent;
  hashtags: Array<Scalars['String']['output']>;
  identifiers: Scalars['JSON']['output'];
  imageUrls: Array<Scalars['String']['output']>;
};

export type AdminEventPage = {
  __typename: 'AdminEventPage';
  items: Array<NostrEvent>;
  pageInfo: PageInfo;
};

export type AdminEventReport = {
  __typename: 'AdminEventReport';
  content?: Maybe<Scalars['String']['output']>;
  createdAt: Scalars['Int64']['output'];
  reportEventId: Scalars['ID']['output'];
  reportType?: Maybe<Scalars['String']['output']>;
  reportedEventId?: Maybe<Scalars['String']['output']>;
  reportedPubkey?: Maybe<Scalars['String']['output']>;
  reporterDisplayName: Scalars['String']['output'];
  reporterNpub?: Maybe<Scalars['String']['output']>;
  reporterPicture?: Maybe<Scalars['String']['output']>;
  reporterPubkey: Scalars['String']['output'];
};

export type AdminEventReportPage = {
  __typename: 'AdminEventReportPage';
  items: Array<AdminEventReport>;
  pageInfo: PageInfo;
};

export type AdminEventSearchInput = {
  authors?: InputMaybe<Array<Scalars['String']['input']>>;
  kinds?: InputMaybe<Array<Scalars['Int']['input']>>;
  q?: InputMaybe<Scalars['String']['input']>;
  since?: InputMaybe<Scalars['Int64']['input']>;
  tags?: InputMaybe<Array<AdminTagFilterInput>>;
  until?: InputMaybe<Scalars['Int64']['input']>;
};

export type AdminFetchEventInput = {
  relays?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type AdminFetchEventPayload = {
  __typename: 'AdminFetchEventPayload';
  eventId?: Maybe<Scalars['ID']['output']>;
  found: Scalars['Boolean']['output'];
  message?: Maybe<Scalars['String']['output']>;
  persisted: Scalars['Boolean']['output'];
  relayResults: Array<AdminFetchRelayResult>;
  relaysTried: Scalars['Int']['output'];
  sourceRelay?: Maybe<Scalars['String']['output']>;
};

export type AdminFetchRelayResult = {
  __typename: 'AdminFetchRelayResult';
  error?: Maybe<Scalars['String']['output']>;
  relay: Scalars['String']['output'];
  status: Scalars['String']['output'];
};

export type AdminGroup = {
  __typename: 'AdminGroup';
  closed: Scalars['Boolean']['output'];
  description: Scalars['String']['output'];
  groupId: Scalars['ID']['output'];
  hidden: Scalars['Boolean']['output'];
  memberCount: Scalars['Int']['output'];
  name: Scalars['String']['output'];
  private: Scalars['Boolean']['output'];
};

export type AdminImportEventsPayload = {
  __typename: 'AdminImportEventsPayload';
  files: Array<AdminImportFileResult>;
};

export type AdminImportFileResult = {
  __typename: 'AdminImportFileResult';
  duplicates: Scalars['Int']['output'];
  error?: Maybe<Scalars['String']['output']>;
  filename: Scalars['String']['output'];
  inserted: Scalars['Int']['output'];
  invalid: Scalars['Int']['output'];
  total: Scalars['Int']['output'];
};

export type AdminJob = {
  __typename: 'AdminJob';
  attempts: Scalars['Int']['output'];
  createdAt: Scalars['String']['output'];
  finishedAt?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  jobName: Scalars['String']['output'];
  lastError?: Maybe<Scalars['String']['output']>;
  maxAttempts: Scalars['Int']['output'];
  payload?: Maybe<Scalars['JSON']['output']>;
  priority: Scalars['String']['output'];
  queue: Scalars['String']['output'];
  result?: Maybe<Scalars['JSON']['output']>;
  runAt?: Maybe<Scalars['String']['output']>;
  startedAt?: Maybe<Scalars['String']['output']>;
  status: Scalars['String']['output'];
};

export type AdminJobFilterInput = {
  jobName?: InputMaybe<Scalars['String']['input']>;
  queue?: InputMaybe<Scalars['String']['input']>;
  statuses?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type AdminJobPage = {
  __typename: 'AdminJobPage';
  items: Array<AdminJob>;
  pageInfo: PageInfo;
};

export type AdminLabelEvent = {
  __typename: 'AdminLabelEvent';
  authorNpub?: Maybe<Scalars['String']['output']>;
  content: Scalars['String']['output'];
  createdAt: Scalars['Int64']['output'];
  id: Scalars['ID']['output'];
  kind: Scalars['Int']['output'];
  labels: Array<Scalars['String']['output']>;
  namespace: Scalars['String']['output'];
  pubkey: Scalars['String']['output'];
  tags: Array<NostrTag>;
  target: AdminLabelTarget;
};

export type AdminLabelFilterInput = {
  author?: InputMaybe<Scalars['String']['input']>;
  labels?: InputMaybe<Array<Scalars['String']['input']>>;
  namespace?: InputMaybe<Scalars['String']['input']>;
  q?: InputMaybe<Scalars['String']['input']>;
  target?: InputMaybe<Scalars['String']['input']>;
  targetType?: InputMaybe<LabelTargetType>;
};

export type AdminLabelPage = {
  __typename: 'AdminLabelPage';
  items: Array<AdminLabelEvent>;
  pageInfo: PageInfo;
};

export type AdminLabelTarget = {
  __typename: 'AdminLabelTarget';
  relayHint?: Maybe<Scalars['String']['output']>;
  type: LabelTargetType;
  value: Scalars['String']['output'];
};

export type AdminLabelTargetInput = {
  relayHint?: InputMaybe<Scalars['String']['input']>;
  type: LabelTargetType;
  value: Scalars['String']['input'];
};

export type AdminLabelsSummary = {
  __typename: 'AdminLabelsSummary';
  labels: Array<NameCount>;
  namespaces: Array<NameCount>;
  targetTypes: Array<NameCount>;
  totalEvents: Scalars['Int64']['output'];
  totalTargets: Scalars['Int64']['output'];
};

export type AdminLoggedUser = {
  __typename: 'AdminLoggedUser';
  connectionCount: Scalars['Int']['output'];
  connectionState: Scalars['String']['output'];
  lastSeenAt?: Maybe<Scalars['String']['output']>;
  profile: AdminProfile;
};

export type AdminLoggedUserPage = {
  __typename: 'AdminLoggedUserPage';
  items: Array<AdminLoggedUser>;
  pageInfo: PageInfo;
};

export type AdminNegentropySyncInput = {
  direction?: InputMaybe<Scalars['String']['input']>;
  filter?: InputMaybe<Scalars['JSON']['input']>;
  remote: Scalars['String']['input'];
  timeoutSeconds?: InputMaybe<Scalars['Int']['input']>;
};

export type AdminNip05Identity = {
  __typename: 'AdminNip05Identity';
  createdAt?: Maybe<Scalars['String']['output']>;
  displayName?: Maybe<Scalars['String']['output']>;
  name: Scalars['String']['output'];
  npub?: Maybe<Scalars['String']['output']>;
  picture?: Maybe<Scalars['String']['output']>;
  pubkey: Scalars['String']['output'];
  relayHints: Array<Scalars['String']['output']>;
  updatedAt?: Maybe<Scalars['String']['output']>;
};

export type AdminNip05IdentityPage = {
  __typename: 'AdminNip05IdentityPage';
  items: Array<AdminNip05Identity>;
  pageInfo: PageInfo;
};

export type AdminNip05Lookup = {
  __typename: 'AdminNip05Lookup';
  exists: Scalars['Boolean']['output'];
  identity?: Maybe<AdminNip05Identity>;
  pubkey: Scalars['String']['output'];
};

export type AdminNip05UpsertInput = {
  name: Scalars['String']['input'];
  pubkey: Scalars['String']['input'];
};

export type AdminNip86EventPage = {
  __typename: 'AdminNip86EventPage';
  items: Array<AdminNip86EventRecord>;
  pageInfo: PageInfo;
};

export type AdminNip86EventRecord = {
  __typename: 'AdminNip86EventRecord';
  createdAt?: Maybe<Scalars['String']['output']>;
  createdBy?: Maybe<Scalars['String']['output']>;
  eventId: Scalars['ID']['output'];
  reason?: Maybe<Scalars['String']['output']>;
  updatedAt?: Maybe<Scalars['String']['output']>;
};

export type AdminNip86IpPage = {
  __typename: 'AdminNip86IPPage';
  items: Array<AdminNip86IpRecord>;
  pageInfo: PageInfo;
};

export type AdminNip86IpRecord = {
  __typename: 'AdminNip86IPRecord';
  createdAt?: Maybe<Scalars['String']['output']>;
  createdBy?: Maybe<Scalars['String']['output']>;
  ip: Scalars['String']['output'];
  reason?: Maybe<Scalars['String']['output']>;
  updatedAt?: Maybe<Scalars['String']['output']>;
};

export type AdminNip86PubkeyPage = {
  __typename: 'AdminNip86PubkeyPage';
  items: Array<AdminNip86PubkeyRecord>;
  pageInfo: PageInfo;
};

export type AdminNip86PubkeyRecord = {
  __typename: 'AdminNip86PubkeyRecord';
  createdAt?: Maybe<Scalars['String']['output']>;
  createdBy?: Maybe<Scalars['String']['output']>;
  pubkey: Scalars['String']['output'];
  reason?: Maybe<Scalars['String']['output']>;
  updatedAt?: Maybe<Scalars['String']['output']>;
};

export type AdminNip86ReasonInput = {
  reason?: InputMaybe<Scalars['String']['input']>;
};

export type AdminNip86RelayMetadata = {
  __typename: 'AdminNip86RelayMetadata';
  description: Scalars['String']['output'];
  name: Scalars['String']['output'];
  relayUrl: Scalars['String']['output'];
};

export type AdminNip86RelayMetadataInput = {
  description: Scalars['String']['input'];
  name: Scalars['String']['input'];
};

export type AdminOverview = {
  __typename: 'AdminOverview';
  activeConnections: Scalars['Int']['output'];
  authedConnections: Scalars['Int']['output'];
  bannedUsers: Scalars['Int']['output'];
  eventsPerMinute: Scalars['Int64']['output'];
  indexedEvents: Scalars['Int64']['output'];
  loggedUsers: Scalars['Int']['output'];
  relayStatus: Scalars['String']['output'];
};

export type AdminProfile = {
  __typename: 'AdminProfile';
  about?: Maybe<Scalars['String']['output']>;
  bot: Scalars['Boolean']['output'];
  createdAt?: Maybe<Scalars['String']['output']>;
  displayName: Scalars['String']['output'];
  handle?: Maybe<Scalars['String']['output']>;
  nip05?: Maybe<Scalars['String']['output']>;
  npub?: Maybe<Scalars['String']['output']>;
  picture?: Maybe<Scalars['String']['output']>;
  pubkey: Scalars['String']['output'];
  reason?: Maybe<Scalars['String']['output']>;
  relatedIds: Array<Scalars['String']['output']>;
  status?: Maybe<Scalars['String']['output']>;
  website?: Maybe<Scalars['String']['output']>;
};

export type AdminProfilePage = {
  __typename: 'AdminProfilePage';
  items: Array<AdminProfile>;
  pageInfo: PageInfo;
};

export type AdminReportedEvent = {
  __typename: 'AdminReportedEvent';
  lastReported: Scalars['Int64']['output'];
  lastReportedAt?: Maybe<Scalars['String']['output']>;
  reportCount: Scalars['Int64']['output'];
  reportTypes: Array<Scalars['String']['output']>;
  targetAuthor: Scalars['JSON']['output'];
  targetCreatedAt?: Maybe<Scalars['Int64']['output']>;
  targetCreatedAtIso?: Maybe<Scalars['String']['output']>;
  targetEvent?: Maybe<NostrEvent>;
  targetEventId: Scalars['ID']['output'];
  targetNevent?: Maybe<Scalars['String']['output']>;
  targetPubkey?: Maybe<Scalars['String']['output']>;
};

export type AdminReportedEventFilterInput = {
  q?: InputMaybe<Scalars['String']['input']>;
  types?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type AdminReportedEventPage = {
  __typename: 'AdminReportedEventPage';
  items: Array<AdminReportedEvent>;
  pageInfo: PageInfo;
};

export type AdminReportedEventsSummary = {
  __typename: 'AdminReportedEventsSummary';
  reportTypes: Array<NameCount>;
  timeline: Array<NameCount>;
  topAuthors: Array<PubkeyCount>;
  topTargets: Array<TargetCount>;
  totalEvents: Scalars['Int64']['output'];
  totalReports: Scalars['Int64']['output'];
  uniqueTargetAuthors: Scalars['Int64']['output'];
};

export type AdminStreamStatus = {
  __typename: 'AdminStreamStatus';
  config: Scalars['JSON']['output'];
  counters: Scalars['JSON']['output'];
  dispatcher: Scalars['JSON']['output'];
  pool: Scalars['JSON']['output'];
};

export type AdminTagFilterInput = {
  name: Scalars['String']['input'];
  value: Scalars['String']['input'];
};

export type AdminTimeline = {
  __typename: 'AdminTimeline';
  bucket: TimelineBucket;
  points: Array<AdminTimelinePoint>;
};

export type AdminTimelinePoint = {
  __typename: 'AdminTimelinePoint';
  count: Scalars['Int64']['output'];
  ts: Scalars['Int64']['output'];
};

export type AdminWoTSummary = {
  __typename: 'AdminWoTSummary';
  lastComputedAt?: Maybe<Scalars['String']['output']>;
  totalEdges: Scalars['Int']['output'];
  totalNodes: Scalars['Int']['output'];
  trustedPubkeys: Array<Scalars['String']['output']>;
};

export type BlossomPlanScope =
  | 'ENABLED_USERS'
  | 'FREE';

export type BlossomReportStatus =
  | 'ACTIONED'
  | 'DISMISSED'
  | 'OPEN';

export type BlossomReviewAction =
  | 'APPROVE'
  | 'HARD_DELETE'
  | 'REQUEUE_OPTIMIZATION';

export type KindCount = {
  __typename: 'KindCount';
  count: Scalars['Int64']['output'];
  kind: Scalars['Int']['output'];
};

export type LabelTargetType =
  | 'ADDRESS'
  | 'EVENT'
  | 'PUBKEY'
  | 'REFERENCE'
  | 'TOPIC';

export type Mutation = {
  __typename: 'Mutation';
  addTrustedPubkey: AdminWoTSummary;
  assignBlossomPlan: MutationAck;
  banUser: AdminBanStatus;
  cancelJob: MutationAck;
  createLabel: AdminCreateLabelPayload;
  createNip86AllowedPubkey: MutationAck;
  createNip86BannedEvent: MutationAck;
  createNip86BlockedIp: MutationAck;
  deleteBlossomPlan: MutationAck;
  deleteJobsHistory: MutationAck;
  deleteNip05: MutationAck;
  deleteNip86AllowedPubkey: MutationAck;
  deleteNip86BannedEvent: MutationAck;
  deleteNip86BlockedIp: MutationAck;
  disconnectConnection: MutationAck;
  downloadEvents: AdminAsyncJob;
  fetchEventFromRelays: AdminFetchEventPayload;
  importEvents: AdminImportEventsPayload;
  mirrorBlossomObject: AdminAsyncJob;
  purgeBlossomUser: MutationAck;
  removeTrustedPubkey: AdminWoTSummary;
  resolveBlossomReport: AdminBlossomReport;
  resumeJob: MutationAck;
  retryJob: MutationAck;
  reviewBlossomObjects: MutationAck;
  startNegentropySync: AdminAsyncJob;
  unassignBlossomPlan: MutationAck;
  unbanUser: AdminBanStatus;
  updateNip86RelayMetadata: AdminNip86RelayMetadata;
  upsertBlossomPlan: AdminBlossomPlan;
  upsertBlossomPolicy: AdminBlossomPolicy;
  upsertBlossomWhitelist: AdminBlossomUser;
  upsertNip05: AdminNip05Identity;
};


export type MutationAddTrustedPubkeyArgs = {
  pubkey: Scalars['String']['input'];
};


export type MutationAssignBlossomPlanArgs = {
  planId: Scalars['ID']['input'];
  pubkey: Scalars['String']['input'];
};


export type MutationBanUserArgs = {
  input: AdminBanUserInput;
  pubkey: Scalars['String']['input'];
};


export type MutationCancelJobArgs = {
  id: Scalars['ID']['input'];
  queue?: InputMaybe<Scalars['String']['input']>;
};


export type MutationCreateLabelArgs = {
  input: AdminCreateLabelInput;
};


export type MutationCreateNip86AllowedPubkeyArgs = {
  input?: InputMaybe<AdminNip86ReasonInput>;
  pubkey: Scalars['String']['input'];
};


export type MutationCreateNip86BannedEventArgs = {
  eventId: Scalars['ID']['input'];
  input?: InputMaybe<AdminNip86ReasonInput>;
};


export type MutationCreateNip86BlockedIpArgs = {
  input?: InputMaybe<AdminNip86ReasonInput>;
  ip: Scalars['String']['input'];
};


export type MutationDeleteBlossomPlanArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteJobsHistoryArgs = {
  input: AdminDeleteJobsHistoryInput;
};


export type MutationDeleteNip05Args = {
  name: Scalars['String']['input'];
};


export type MutationDeleteNip86AllowedPubkeyArgs = {
  pubkey: Scalars['String']['input'];
};


export type MutationDeleteNip86BannedEventArgs = {
  eventId: Scalars['ID']['input'];
};


export type MutationDeleteNip86BlockedIpArgs = {
  ip: Scalars['String']['input'];
};


export type MutationDisconnectConnectionArgs = {
  reason?: InputMaybe<Scalars['String']['input']>;
  wsid: Scalars['ID']['input'];
};


export type MutationDownloadEventsArgs = {
  input: AdminDownloadEventsInput;
};


export type MutationFetchEventFromRelaysArgs = {
  id: Scalars['ID']['input'];
  input?: InputMaybe<AdminFetchEventInput>;
};


export type MutationImportEventsArgs = {
  files: Array<Scalars['Upload']['input']>;
};


export type MutationMirrorBlossomObjectArgs = {
  input: AdminBlossomMirrorInput;
};


export type MutationPurgeBlossomUserArgs = {
  pubkey: Scalars['String']['input'];
};


export type MutationRemoveTrustedPubkeyArgs = {
  pubkey: Scalars['String']['input'];
};


export type MutationResolveBlossomReportArgs = {
  id: Scalars['ID']['input'];
  input: AdminBlossomResolveReportInput;
};


export type MutationResumeJobArgs = {
  id: Scalars['ID']['input'];
  queue?: InputMaybe<Scalars['String']['input']>;
};


export type MutationRetryJobArgs = {
  id: Scalars['ID']['input'];
  queue?: InputMaybe<Scalars['String']['input']>;
};


export type MutationReviewBlossomObjectsArgs = {
  input: AdminBlossomBulkReviewInput;
};


export type MutationStartNegentropySyncArgs = {
  input: AdminNegentropySyncInput;
};


export type MutationUnassignBlossomPlanArgs = {
  planId: Scalars['ID']['input'];
  pubkey: Scalars['String']['input'];
};


export type MutationUnbanUserArgs = {
  pubkey: Scalars['String']['input'];
};


export type MutationUpdateNip86RelayMetadataArgs = {
  input: AdminNip86RelayMetadataInput;
};


export type MutationUpsertBlossomPlanArgs = {
  input: AdminBlossomPlanInput;
};


export type MutationUpsertBlossomPolicyArgs = {
  input: AdminBlossomPolicyInput;
};


export type MutationUpsertBlossomWhitelistArgs = {
  input: AdminBlossomWhitelistInput;
};


export type MutationUpsertNip05Args = {
  input: AdminNip05UpsertInput;
};

export type MutationAck = {
  __typename: 'MutationAck';
  entityId?: Maybe<Scalars['ID']['output']>;
  message?: Maybe<Scalars['String']['output']>;
  ok: Scalars['Boolean']['output'];
};

export type NameCount = {
  __typename: 'NameCount';
  count: Scalars['Int64']['output'];
  name: Scalars['String']['output'];
};

export type NostrEvent = {
  __typename: 'NostrEvent';
  content: Scalars['String']['output'];
  createdAt: Scalars['Int64']['output'];
  id: Scalars['ID']['output'];
  kind: Scalars['Int']['output'];
  pubkey: Scalars['String']['output'];
  sig: Scalars['String']['output'];
  tags: Array<NostrTag>;
};

export type NostrTag = {
  __typename: 'NostrTag';
  values: Array<Scalars['String']['output']>;
};

export type OffsetPageInput = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
};

export type PageInfo = {
  __typename: 'PageInfo';
  hasMore: Scalars['Boolean']['output'];
  limit: Scalars['Int']['output'];
  offset: Scalars['Int']['output'];
  total: Scalars['Int']['output'];
};

export type PrivacyNetwork = {
  __typename: 'PrivacyNetwork';
  addresses: Array<Scalars['String']['output']>;
  enabled: Scalars['Boolean']['output'];
  error?: Maybe<Scalars['String']['output']>;
  id: PrivacyNetworkId;
  metrics?: Maybe<PrivacyNetworkMetrics>;
  mode: Scalars['String']['output'];
  name: Scalars['String']['output'];
  started: Scalars['Boolean']['output'];
  status: PrivacyNetworkStatusEnum;
  uptimeMs: Scalars['Int64']['output'];
};

export type PrivacyNetworkId =
  | 'i2p'
  | 'tor'
  | 'yggdrasil';

export type PrivacyNetworkMetrics = {
  __typename: 'PrivacyNetworkMetrics';
  connections?: Maybe<Scalars['Int']['output']>;
  peers?: Maybe<Scalars['Int']['output']>;
  rxBytes: Scalars['Int64']['output'];
  txBytes: Scalars['Int64']['output'];
};

export type PrivacyNetworkStatusEnum =
  | 'disabled'
  | 'error'
  | 'operational'
  | 'starting';

export type PrivacyStatus = {
  __typename: 'PrivacyStatus';
  enabled: Scalars['Boolean']['output'];
  networks: Array<PrivacyNetwork>;
  persistence: Scalars['Boolean']['output'];
  stateDir?: Maybe<Scalars['String']['output']>;
};

export type PubkeyCount = {
  __typename: 'PubkeyCount';
  count: Scalars['Int64']['output'];
  displayName?: Maybe<Scalars['String']['output']>;
  pubkey: Scalars['String']['output'];
};

export type Query = {
  __typename: 'Query';
  activeConnections: AdminConnectionPage;
  adminOverview: AdminOverview;
  adminStreamStatus: AdminStreamStatus;
  authedConnections: AdminConnectionPage;
  bannedUsers: AdminProfilePage;
  blossomAnalytics: Scalars['JSON']['output'];
  blossomAudit: AdminBlossomAuditPage;
  blossomObject: AdminBlossomObject;
  blossomObjects: AdminBlossomObjectPage;
  blossomOverview: AdminBlossomOverview;
  blossomPlanAssignments: AdminBlossomPlanAssignmentPage;
  blossomPlans: Array<AdminBlossomPlan>;
  blossomPolicy: AdminBlossomPolicy;
  blossomReports: AdminBlossomReportPage;
  blossomUser: AdminBlossomUserDetail;
  blossomUsers: AdminBlossomUserPage;
  blossomWorkers: AdminBlossomWorkerPage;
  eventAggregates: AdminEventAggregates;
  eventDetail: AdminEventDetail;
  eventReports: AdminEventReportPage;
  eventTimeline: AdminTimeline;
  events: AdminEventPage;
  groups: Array<AdminGroup>;
  job: AdminJob;
  jobs: AdminJobPage;
  labels: AdminLabelPage;
  labelsSummary: AdminLabelsSummary;
  loggedUsers: AdminLoggedUserPage;
  nip05Identities: AdminNip05IdentityPage;
  nip86AllowedPubkeys: AdminNip86PubkeyPage;
  nip86BannedEvents: AdminNip86EventPage;
  nip86BlockedIps: AdminNip86IpPage;
  nip86RelayMetadata: AdminNip86RelayMetadata;
  privacyStatus: PrivacyStatus;
  reportedEvents: AdminReportedEventPage;
  reportedEventsSummary: AdminReportedEventsSummary;
  searchUsers: AdminProfilePage;
  userBanStatus: AdminBanStatus;
  userNip05: AdminNip05Lookup;
  userProfile: AdminProfile;
  wotSummary: AdminWoTSummary;
};


export type QueryActiveConnectionsArgs = {
  page?: InputMaybe<OffsetPageInput>;
};


export type QueryAuthedConnectionsArgs = {
  page?: InputMaybe<OffsetPageInput>;
};


export type QueryBannedUsersArgs = {
  page?: InputMaybe<OffsetPageInput>;
  q?: InputMaybe<Scalars['String']['input']>;
};


export type QueryBlossomAuditArgs = {
  page?: InputMaybe<OffsetPageInput>;
};


export type QueryBlossomObjectArgs = {
  hash: Scalars['ID']['input'];
};


export type QueryBlossomObjectsArgs = {
  filter?: InputMaybe<AdminBlossomObjectFilterInput>;
  page?: InputMaybe<OffsetPageInput>;
};


export type QueryBlossomPlanAssignmentsArgs = {
  page?: InputMaybe<OffsetPageInput>;
  planId: Scalars['ID']['input'];
};


export type QueryBlossomReportsArgs = {
  filter?: InputMaybe<AdminBlossomReportFilterInput>;
  page?: InputMaybe<OffsetPageInput>;
};


export type QueryBlossomUserArgs = {
  pubkey: Scalars['String']['input'];
};


export type QueryBlossomUsersArgs = {
  filter?: InputMaybe<AdminBlossomUserFilterInput>;
  page?: InputMaybe<OffsetPageInput>;
};


export type QueryBlossomWorkersArgs = {
  jobType?: InputMaybe<Scalars['String']['input']>;
  page?: InputMaybe<OffsetPageInput>;
  status?: InputMaybe<Scalars['String']['input']>;
  targetHash?: InputMaybe<Scalars['String']['input']>;
};


export type QueryEventAggregatesArgs = {
  filter?: InputMaybe<AdminEventSearchInput>;
};


export type QueryEventDetailArgs = {
  id: Scalars['ID']['input'];
};


export type QueryEventReportsArgs = {
  id: Scalars['ID']['input'];
  page?: InputMaybe<OffsetPageInput>;
};


export type QueryEventTimelineArgs = {
  bucket?: InputMaybe<TimelineBucket>;
  filter?: InputMaybe<AdminEventSearchInput>;
};


export type QueryEventsArgs = {
  filter?: InputMaybe<AdminEventSearchInput>;
  page?: InputMaybe<OffsetPageInput>;
};


export type QueryJobArgs = {
  id: Scalars['ID']['input'];
  queue?: InputMaybe<Scalars['String']['input']>;
};


export type QueryJobsArgs = {
  filter?: InputMaybe<AdminJobFilterInput>;
  page?: InputMaybe<OffsetPageInput>;
};


export type QueryLabelsArgs = {
  filter?: InputMaybe<AdminLabelFilterInput>;
  page?: InputMaybe<OffsetPageInput>;
};


export type QueryLabelsSummaryArgs = {
  filter?: InputMaybe<AdminLabelFilterInput>;
};


export type QueryLoggedUsersArgs = {
  page?: InputMaybe<OffsetPageInput>;
};


export type QueryNip05IdentitiesArgs = {
  page?: InputMaybe<OffsetPageInput>;
  q?: InputMaybe<Scalars['String']['input']>;
};


export type QueryNip86AllowedPubkeysArgs = {
  page?: InputMaybe<OffsetPageInput>;
  q?: InputMaybe<Scalars['String']['input']>;
};


export type QueryNip86BannedEventsArgs = {
  page?: InputMaybe<OffsetPageInput>;
  q?: InputMaybe<Scalars['String']['input']>;
};


export type QueryNip86BlockedIpsArgs = {
  page?: InputMaybe<OffsetPageInput>;
  q?: InputMaybe<Scalars['String']['input']>;
};


export type QueryReportedEventsArgs = {
  filter?: InputMaybe<AdminReportedEventFilterInput>;
  page?: InputMaybe<OffsetPageInput>;
};


export type QueryReportedEventsSummaryArgs = {
  filter?: InputMaybe<AdminReportedEventFilterInput>;
};


export type QuerySearchUsersArgs = {
  page?: InputMaybe<OffsetPageInput>;
  q?: InputMaybe<Scalars['String']['input']>;
};


export type QueryUserBanStatusArgs = {
  pubkey: Scalars['String']['input'];
};


export type QueryUserNip05Args = {
  pubkey: Scalars['String']['input'];
};


export type QueryUserProfileArgs = {
  pubkey: Scalars['String']['input'];
};

export type SortDirection =
  | 'ASC'
  | 'DESC';

export type TagCount = {
  __typename: 'TagCount';
  count: Scalars['Int64']['output'];
  tag: Scalars['String']['output'];
};

export type TargetCount = {
  __typename: 'TargetCount';
  count: Scalars['Int64']['output'];
  targetEventId: Scalars['ID']['output'];
};

export type TimelineBucket =
  | 'DAY'
  | 'HOUR';

export type AdminOverviewQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminOverviewQuery = { __typename: 'Query', adminOverview: { __typename: 'AdminOverview', activeConnections: number, authedConnections: number, loggedUsers: number, bannedUsers: number, indexedEvents: number, eventsPerMinute: number, relayStatus: string } };

export type AdminStreamStatusQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminStreamStatusQuery = { __typename: 'Query', adminStreamStatus: { __typename: 'AdminStreamStatus', config: Record<string, unknown>, dispatcher: Record<string, unknown>, pool: Record<string, unknown>, counters: Record<string, unknown> } };

export type PrivacyStatusQueryVariables = Exact<{ [key: string]: never; }>;


export type PrivacyStatusQuery = { __typename: 'Query', privacyStatus: { __typename: 'PrivacyStatus', enabled: boolean, persistence: boolean, stateDir?: string | null, networks: Array<{ __typename: 'PrivacyNetwork', id: PrivacyNetworkId, name: string, mode: string, enabled: boolean, started: boolean, status: PrivacyNetworkStatusEnum, addresses: Array<string>, error?: string | null, uptimeMs: number, metrics?: { __typename: 'PrivacyNetworkMetrics', txBytes: number, rxBytes: number, peers?: number | null, connections?: number | null } | null }> } };

export type ActiveConnectionsQueryVariables = Exact<{
  page?: InputMaybe<OffsetPageInput>;
}>;


export type ActiveConnectionsQuery = { __typename: 'Query', activeConnections: { __typename: 'AdminConnectionPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminConnection', wsid: string, ip: string, authed?: string | null, subscriptionCount: number, connectedAt?: string | null, lastSeenAt?: string | null, userAgent?: string | null }> } };

export type AuthedConnectionsQueryVariables = Exact<{
  page?: InputMaybe<OffsetPageInput>;
}>;


export type AuthedConnectionsQuery = { __typename: 'Query', authedConnections: { __typename: 'AdminConnectionPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminConnection', wsid: string, ip: string, authed?: string | null, subscriptionCount: number, connectedAt?: string | null, lastSeenAt?: string | null, userAgent?: string | null }> } };

export type DisconnectConnectionMutationVariables = Exact<{
  wsid: Scalars['ID']['input'];
  reason?: InputMaybe<Scalars['String']['input']>;
}>;


export type DisconnectConnectionMutation = { __typename: 'Mutation', disconnectConnection: { __typename: 'MutationAck', ok: boolean, entityId?: string | null } };

export type LoggedUsersQueryVariables = Exact<{
  page?: InputMaybe<OffsetPageInput>;
}>;


export type LoggedUsersQuery = { __typename: 'Query', loggedUsers: { __typename: 'AdminLoggedUserPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminLoggedUser', connectionCount: number, lastSeenAt?: string | null, connectionState: string, profile: { __typename: 'AdminProfile', pubkey: string, npub?: string | null, displayName: string, handle?: string | null, picture?: string | null, nip05?: string | null, about?: string | null, website?: string | null, bot: boolean, status?: string | null, reason?: string | null, relatedIds: Array<string>, createdAt?: string | null } }> } };

export type BannedUsersQueryVariables = Exact<{
  q?: InputMaybe<Scalars['String']['input']>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type BannedUsersQuery = { __typename: 'Query', bannedUsers: { __typename: 'AdminProfilePage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminProfile', pubkey: string, npub?: string | null, displayName: string, handle?: string | null, picture?: string | null, nip05?: string | null, about?: string | null, website?: string | null, bot: boolean, status?: string | null, reason?: string | null, relatedIds: Array<string>, createdAt?: string | null }> } };

export type SearchUsersQueryVariables = Exact<{
  q?: InputMaybe<Scalars['String']['input']>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type SearchUsersQuery = { __typename: 'Query', searchUsers: { __typename: 'AdminProfilePage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminProfile', pubkey: string, npub?: string | null, displayName: string, handle?: string | null, picture?: string | null, nip05?: string | null, about?: string | null, website?: string | null, bot: boolean, status?: string | null, reason?: string | null, relatedIds: Array<string>, createdAt?: string | null }> } };

export type UserProfileQueryVariables = Exact<{
  pubkey: Scalars['String']['input'];
}>;


export type UserProfileQuery = { __typename: 'Query', userProfile: { __typename: 'AdminProfile', pubkey: string, npub?: string | null, displayName: string, handle?: string | null, picture?: string | null, nip05?: string | null, about?: string | null, website?: string | null, bot: boolean, status?: string | null, reason?: string | null, relatedIds: Array<string>, createdAt?: string | null } };

export type UserBanStatusQueryVariables = Exact<{
  pubkey: Scalars['String']['input'];
}>;


export type UserBanStatusQuery = { __typename: 'Query', userBanStatus: { __typename: 'AdminBanStatus', pubkey: string, banned: boolean, reason?: string | null, relatedIds: Array<string>, createdAt?: string | null } };

export type BanUserMutationVariables = Exact<{
  pubkey: Scalars['String']['input'];
  input: AdminBanUserInput;
}>;


export type BanUserMutation = { __typename: 'Mutation', banUser: { __typename: 'AdminBanStatus', pubkey: string, banned: boolean, reason?: string | null, relatedIds: Array<string>, createdAt?: string | null } };

export type UnbanUserMutationVariables = Exact<{
  pubkey: Scalars['String']['input'];
}>;


export type UnbanUserMutation = { __typename: 'Mutation', unbanUser: { __typename: 'AdminBanStatus', pubkey: string, banned: boolean, reason?: string | null, relatedIds: Array<string>, createdAt?: string | null } };

export type EventsQueryVariables = Exact<{
  filter?: InputMaybe<AdminEventSearchInput>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type EventsQuery = { __typename: 'Query', events: { __typename: 'AdminEventPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'NostrEvent', id: string, pubkey: string, createdAt: number, kind: number, content: string, sig: string, tags: Array<{ __typename: 'NostrTag', values: Array<string> }> }> } };

export type EventAggregatesQueryVariables = Exact<{
  filter?: InputMaybe<AdminEventSearchInput>;
}>;


export type EventAggregatesQuery = { __typename: 'Query', eventAggregates: { __typename: 'AdminEventAggregates', total: number, trends: Record<string, unknown>, kinds: Array<{ __typename: 'KindCount', kind: number, count: number }>, topAuthors: Array<{ __typename: 'PubkeyCount', pubkey: string, displayName?: string | null, count: number }>, topTags: Array<{ __typename: 'TagCount', tag: string, count: number }> } };

export type EventTimelineQueryVariables = Exact<{
  filter?: InputMaybe<AdminEventSearchInput>;
  bucket?: InputMaybe<TimelineBucket>;
}>;


export type EventTimelineQuery = { __typename: 'Query', eventTimeline: { __typename: 'AdminTimeline', bucket: TimelineBucket, points: Array<{ __typename: 'AdminTimelinePoint', ts: number, count: number }> } };

export type EventDetailQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type EventDetailQuery = { __typename: 'Query', eventDetail: { __typename: 'AdminEventDetail', identifiers: Record<string, unknown>, author: Record<string, unknown>, hashtags: Array<string>, imageUrls: Array<string>, event: { __typename: 'NostrEvent', id: string, pubkey: string, createdAt: number, kind: number, content: string, sig: string, tags: Array<{ __typename: 'NostrTag', values: Array<string> }> } } };

export type FetchEventFromRelaysMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input?: InputMaybe<AdminFetchEventInput>;
}>;


export type FetchEventFromRelaysMutation = { __typename: 'Mutation', fetchEventFromRelays: { __typename: 'AdminFetchEventPayload', eventId?: string | null, sourceRelay?: string | null, found: boolean, persisted: boolean, relaysTried: number, message?: string | null, relayResults: Array<{ __typename: 'AdminFetchRelayResult', relay: string, status: string, error?: string | null }> } };

export type EventReportsQueryVariables = Exact<{
  id: Scalars['ID']['input'];
  page?: InputMaybe<OffsetPageInput>;
}>;


export type EventReportsQuery = { __typename: 'Query', eventReports: { __typename: 'AdminEventReportPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminEventReport', reportEventId: string, reporterPubkey: string, reporterNpub?: string | null, reporterDisplayName: string, reporterPicture?: string | null, reportedEventId?: string | null, reportedPubkey?: string | null, reportType?: string | null, content?: string | null, createdAt: number }> } };

export type ReportedEventsQueryVariables = Exact<{
  filter?: InputMaybe<AdminReportedEventFilterInput>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type ReportedEventsQuery = { __typename: 'Query', reportedEvents: { __typename: 'AdminReportedEventPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminReportedEvent', targetEventId: string, targetPubkey?: string | null, targetNevent?: string | null, targetCreatedAt?: number | null, targetCreatedAtIso?: string | null, targetAuthor: Record<string, unknown>, reportCount: number, lastReported: number, lastReportedAt?: string | null, reportTypes: Array<string>, targetEvent?: { __typename: 'NostrEvent', id: string, pubkey: string, createdAt: number, kind: number, content: string, sig: string, tags: Array<{ __typename: 'NostrTag', values: Array<string> }> } | null }> } };

export type ReportedEventsSummaryQueryVariables = Exact<{
  filter?: InputMaybe<AdminReportedEventFilterInput>;
}>;


export type ReportedEventsSummaryQuery = { __typename: 'Query', reportedEventsSummary: { __typename: 'AdminReportedEventsSummary', totalEvents: number, totalReports: number, uniqueTargetAuthors: number, timeline: Array<{ __typename: 'NameCount', name: string, count: number }>, reportTypes: Array<{ __typename: 'NameCount', name: string, count: number }>, topAuthors: Array<{ __typename: 'PubkeyCount', pubkey: string, displayName?: string | null, count: number }>, topTargets: Array<{ __typename: 'TargetCount', targetEventId: string, count: number }> } };

export type LabelsQueryVariables = Exact<{
  filter?: InputMaybe<AdminLabelFilterInput>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type LabelsQuery = { __typename: 'Query', labels: { __typename: 'AdminLabelPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminLabelEvent', id: string, pubkey: string, authorNpub?: string | null, createdAt: number, kind: number, content: string, namespace: string, labels: Array<string>, target: { __typename: 'AdminLabelTarget', type: LabelTargetType, value: string, relayHint?: string | null }, tags: Array<{ __typename: 'NostrTag', values: Array<string> }> }> } };

export type LabelsSummaryQueryVariables = Exact<{
  filter?: InputMaybe<AdminLabelFilterInput>;
}>;


export type LabelsSummaryQuery = { __typename: 'Query', labelsSummary: { __typename: 'AdminLabelsSummary', totalEvents: number, totalTargets: number, namespaces: Array<{ __typename: 'NameCount', name: string, count: number }>, labels: Array<{ __typename: 'NameCount', name: string, count: number }>, targetTypes: Array<{ __typename: 'NameCount', name: string, count: number }> } };

export type CreateLabelMutationVariables = Exact<{
  input: AdminCreateLabelInput;
}>;


export type CreateLabelMutation = { __typename: 'Mutation', createLabel: { __typename: 'AdminCreateLabelPayload', stored: boolean, event: { __typename: 'NostrEvent', id: string, pubkey: string, createdAt: number, kind: number, content: string, sig: string, tags: Array<{ __typename: 'NostrTag', values: Array<string> }> } } };

export type Nip05IdentitiesQueryVariables = Exact<{
  q?: InputMaybe<Scalars['String']['input']>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type Nip05IdentitiesQuery = { __typename: 'Query', nip05Identities: { __typename: 'AdminNip05IdentityPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminNip05Identity', name: string, pubkey: string, npub?: string | null, displayName?: string | null, picture?: string | null, relayHints: Array<string>, createdAt?: string | null, updatedAt?: string | null }> } };

export type UpsertNip05MutationVariables = Exact<{
  input: AdminNip05UpsertInput;
}>;


export type UpsertNip05Mutation = { __typename: 'Mutation', upsertNip05: { __typename: 'AdminNip05Identity', name: string, pubkey: string, npub?: string | null, displayName?: string | null, picture?: string | null, relayHints: Array<string>, createdAt?: string | null, updatedAt?: string | null } };

export type DeleteNip05MutationVariables = Exact<{
  name: Scalars['String']['input'];
}>;


export type DeleteNip05Mutation = { __typename: 'Mutation', deleteNip05: { __typename: 'MutationAck', ok: boolean, entityId?: string | null } };

export type UserNip05QueryVariables = Exact<{
  pubkey: Scalars['String']['input'];
}>;


export type UserNip05Query = { __typename: 'Query', userNip05: { __typename: 'AdminNip05Lookup', pubkey: string, exists: boolean, identity?: { __typename: 'AdminNip05Identity', name: string, pubkey: string, npub?: string | null, displayName?: string | null, picture?: string | null, relayHints: Array<string>, createdAt?: string | null, updatedAt?: string | null } | null } };

export type Nip86AllowedPubkeysQueryVariables = Exact<{
  q?: InputMaybe<Scalars['String']['input']>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type Nip86AllowedPubkeysQuery = { __typename: 'Query', nip86AllowedPubkeys: { __typename: 'AdminNip86PubkeyPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminNip86PubkeyRecord', pubkey: string, reason?: string | null, createdBy?: string | null, createdAt?: string | null, updatedAt?: string | null }> } };

export type CreateNip86AllowedPubkeyMutationVariables = Exact<{
  pubkey: Scalars['String']['input'];
  input?: InputMaybe<AdminNip86ReasonInput>;
}>;


export type CreateNip86AllowedPubkeyMutation = { __typename: 'Mutation', createNip86AllowedPubkey: { __typename: 'MutationAck', ok: boolean } };

export type DeleteNip86AllowedPubkeyMutationVariables = Exact<{
  pubkey: Scalars['String']['input'];
}>;


export type DeleteNip86AllowedPubkeyMutation = { __typename: 'Mutation', deleteNip86AllowedPubkey: { __typename: 'MutationAck', ok: boolean } };

export type Nip86BlockedIpsQueryVariables = Exact<{
  q?: InputMaybe<Scalars['String']['input']>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type Nip86BlockedIpsQuery = { __typename: 'Query', nip86BlockedIps: { __typename: 'AdminNip86IPPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminNip86IPRecord', ip: string, reason?: string | null, createdBy?: string | null, createdAt?: string | null, updatedAt?: string | null }> } };

export type CreateNip86BlockedIpMutationVariables = Exact<{
  ip: Scalars['String']['input'];
  input?: InputMaybe<AdminNip86ReasonInput>;
}>;


export type CreateNip86BlockedIpMutation = { __typename: 'Mutation', createNip86BlockedIp: { __typename: 'MutationAck', ok: boolean } };

export type DeleteNip86BlockedIpMutationVariables = Exact<{
  ip: Scalars['String']['input'];
}>;


export type DeleteNip86BlockedIpMutation = { __typename: 'Mutation', deleteNip86BlockedIp: { __typename: 'MutationAck', ok: boolean } };

export type Nip86BannedEventsQueryVariables = Exact<{
  q?: InputMaybe<Scalars['String']['input']>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type Nip86BannedEventsQuery = { __typename: 'Query', nip86BannedEvents: { __typename: 'AdminNip86EventPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminNip86EventRecord', eventId: string, reason?: string | null, createdBy?: string | null, createdAt?: string | null, updatedAt?: string | null }> } };

export type CreateNip86BannedEventMutationVariables = Exact<{
  eventId: Scalars['ID']['input'];
  input?: InputMaybe<AdminNip86ReasonInput>;
}>;


export type CreateNip86BannedEventMutation = { __typename: 'Mutation', createNip86BannedEvent: { __typename: 'MutationAck', ok: boolean } };

export type DeleteNip86BannedEventMutationVariables = Exact<{
  eventId: Scalars['ID']['input'];
}>;


export type DeleteNip86BannedEventMutation = { __typename: 'Mutation', deleteNip86BannedEvent: { __typename: 'MutationAck', ok: boolean } };

export type Nip86RelayMetadataQueryVariables = Exact<{ [key: string]: never; }>;


export type Nip86RelayMetadataQuery = { __typename: 'Query', nip86RelayMetadata: { __typename: 'AdminNip86RelayMetadata', relayUrl: string, name: string, description: string } };

export type UpdateNip86RelayMetadataMutationVariables = Exact<{
  input: AdminNip86RelayMetadataInput;
}>;


export type UpdateNip86RelayMetadataMutation = { __typename: 'Mutation', updateNip86RelayMetadata: { __typename: 'AdminNip86RelayMetadata', relayUrl: string, name: string, description: string } };

export type StartNegentropySyncMutationVariables = Exact<{
  input: AdminNegentropySyncInput;
}>;


export type StartNegentropySyncMutation = { __typename: 'Mutation', startNegentropySync: { __typename: 'AdminAsyncJob', status: string, jobId?: string | null, remote?: string | null, relays: Array<string>, message?: string | null } };

export type DownloadEventsMutationVariables = Exact<{
  input: AdminDownloadEventsInput;
}>;


export type DownloadEventsMutation = { __typename: 'Mutation', downloadEvents: { __typename: 'AdminAsyncJob', status: string, jobId?: string | null, relays: Array<string>, message?: string | null } };

export type DownloadJobsQueryVariables = Exact<{ [key: string]: never; }>;


export type DownloadJobsQuery = { __typename: 'Query', jobs: { __typename: 'AdminJobPage', items: Array<{ __typename: 'AdminJob', id: string, status: string, createdAt: string, startedAt?: string | null, finishedAt?: string | null, payload?: Record<string, unknown> | null, result?: Record<string, unknown> | null, lastError?: string | null }> } };

export type DownloadJobQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DownloadJobQuery = { __typename: 'Query', job: { __typename: 'AdminJob', id: string, status: string, createdAt: string, startedAt?: string | null, finishedAt?: string | null, payload?: Record<string, unknown> | null, result?: Record<string, unknown> | null, lastError?: string | null } };

export type JobsQueryVariables = Exact<{
  filter?: InputMaybe<AdminJobFilterInput>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type JobsQuery = { __typename: 'Query', jobs: { __typename: 'AdminJobPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminJob', id: string, queue: string, priority: string, jobName: string, status: string, attempts: number, maxAttempts: number, createdAt: string, startedAt?: string | null, finishedAt?: string | null, runAt?: string | null, lastError?: string | null, payload?: Record<string, unknown> | null, result?: Record<string, unknown> | null }> } };

export type JobQueryVariables = Exact<{
  id: Scalars['ID']['input'];
  queue?: InputMaybe<Scalars['String']['input']>;
}>;


export type JobQuery = { __typename: 'Query', job: { __typename: 'AdminJob', id: string, queue: string, priority: string, jobName: string, status: string, attempts: number, maxAttempts: number, createdAt: string, startedAt?: string | null, finishedAt?: string | null, runAt?: string | null, lastError?: string | null, payload?: Record<string, unknown> | null, result?: Record<string, unknown> | null } };

export type RetryJobMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  queue?: InputMaybe<Scalars['String']['input']>;
}>;


export type RetryJobMutation = { __typename: 'Mutation', retryJob: { __typename: 'MutationAck', ok: boolean } };

export type ResumeJobMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  queue?: InputMaybe<Scalars['String']['input']>;
}>;


export type ResumeJobMutation = { __typename: 'Mutation', resumeJob: { __typename: 'MutationAck', ok: boolean } };

export type CancelJobMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  queue?: InputMaybe<Scalars['String']['input']>;
}>;


export type CancelJobMutation = { __typename: 'Mutation', cancelJob: { __typename: 'MutationAck', ok: boolean } };

export type DeleteJobsHistoryMutationVariables = Exact<{
  input: AdminDeleteJobsHistoryInput;
}>;


export type DeleteJobsHistoryMutation = { __typename: 'Mutation', deleteJobsHistory: { __typename: 'MutationAck', ok: boolean, message?: string | null } };

export type GroupsQueryVariables = Exact<{ [key: string]: never; }>;


export type GroupsQuery = { __typename: 'Query', groups: Array<{ __typename: 'AdminGroup', groupId: string, name: string, description: string, private: boolean, closed: boolean, hidden: boolean, memberCount: number }> };

export type WotSummaryQueryVariables = Exact<{ [key: string]: never; }>;


export type WotSummaryQuery = { __typename: 'Query', wotSummary: { __typename: 'AdminWoTSummary', totalNodes: number, totalEdges: number, trustedPubkeys: Array<string>, lastComputedAt?: string | null } };

export type AddTrustedPubkeyMutationVariables = Exact<{
  pubkey: Scalars['String']['input'];
}>;


export type AddTrustedPubkeyMutation = { __typename: 'Mutation', addTrustedPubkey: { __typename: 'AdminWoTSummary', trustedPubkeys: Array<string> } };

export type RemoveTrustedPubkeyMutationVariables = Exact<{
  pubkey: Scalars['String']['input'];
}>;


export type RemoveTrustedPubkeyMutation = { __typename: 'Mutation', removeTrustedPubkey: { __typename: 'AdminWoTSummary', trustedPubkeys: Array<string> } };

export type BlossomOverviewQueryVariables = Exact<{ [key: string]: never; }>;


export type BlossomOverviewQuery = { __typename: 'Query', blossomOverview: { __typename: 'AdminBlossomOverview', storage: Record<string, unknown>, objects: Record<string, unknown>, traffic: Record<string, unknown>, users: Record<string, unknown>, workers: Record<string, unknown>, alerts: Array<Record<string, unknown>>, policy: { __typename: 'AdminBlossomPolicy', mode: string, defaultStorageQuotaBytes?: number | null, defaultEgressQuotaBytes?: number | null, enabledUserDefaultStorageQuotaBytes?: number | null, enabledUserDefaultEgressQuotaBytes?: number | null, updatedAt?: string | null } } };

export type BlossomPolicyQueryVariables = Exact<{ [key: string]: never; }>;


export type BlossomPolicyQuery = { __typename: 'Query', blossomPolicy: { __typename: 'AdminBlossomPolicy', mode: string, defaultStorageQuotaBytes?: number | null, defaultEgressQuotaBytes?: number | null, enabledUserDefaultStorageQuotaBytes?: number | null, enabledUserDefaultEgressQuotaBytes?: number | null, updatedAt?: string | null } };

export type UpsertBlossomPolicyMutationVariables = Exact<{
  input: AdminBlossomPolicyInput;
}>;


export type UpsertBlossomPolicyMutation = { __typename: 'Mutation', upsertBlossomPolicy: { __typename: 'AdminBlossomPolicy', mode: string, defaultStorageQuotaBytes?: number | null, defaultEgressQuotaBytes?: number | null, enabledUserDefaultStorageQuotaBytes?: number | null, enabledUserDefaultEgressQuotaBytes?: number | null, updatedAt?: string | null } };

export type BlossomPlansQueryVariables = Exact<{ [key: string]: never; }>;


export type BlossomPlansQuery = { __typename: 'Query', blossomPlans: Array<{ __typename: 'AdminBlossomPlan', id: string, name: string, scope: BlossomPlanScope, storageQuotaBytes?: number | null, egressQuotaBytes?: number | null, description?: string | null, isDefault: boolean, updatedAt?: string | null }> };

export type UpsertBlossomPlanMutationVariables = Exact<{
  input: AdminBlossomPlanInput;
}>;


export type UpsertBlossomPlanMutation = { __typename: 'Mutation', upsertBlossomPlan: { __typename: 'AdminBlossomPlan', id: string, name: string, scope: BlossomPlanScope, storageQuotaBytes?: number | null, egressQuotaBytes?: number | null, description?: string | null, isDefault: boolean, updatedAt?: string | null } };

export type DeleteBlossomPlanMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteBlossomPlanMutation = { __typename: 'Mutation', deleteBlossomPlan: { __typename: 'MutationAck', ok: boolean } };

export type AssignBlossomPlanMutationVariables = Exact<{
  planId: Scalars['ID']['input'];
  pubkey: Scalars['String']['input'];
}>;


export type AssignBlossomPlanMutation = { __typename: 'Mutation', assignBlossomPlan: { __typename: 'MutationAck', ok: boolean } };

export type UnassignBlossomPlanMutationVariables = Exact<{
  planId: Scalars['ID']['input'];
  pubkey: Scalars['String']['input'];
}>;


export type UnassignBlossomPlanMutation = { __typename: 'Mutation', unassignBlossomPlan: { __typename: 'MutationAck', ok: boolean } };

export type BlossomPlanAssignmentsQueryVariables = Exact<{
  planId: Scalars['ID']['input'];
}>;


export type BlossomPlanAssignmentsQuery = { __typename: 'Query', blossomPlanAssignments: { __typename: 'AdminBlossomPlanAssignmentPage', items: Array<{ __typename: 'AdminBlossomPlanAssignment', planId: string, pubkey: string, displayName?: string | null, picture?: string | null, npub?: string | null, assignedBy: string, assignedAt: string }> } };

export type BlossomObjectsQueryVariables = Exact<{
  filter?: InputMaybe<AdminBlossomObjectFilterInput>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type BlossomObjectsQuery = { __typename: 'Query', blossomObjects: { __typename: 'AdminBlossomObjectPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminBlossomObject', hash: string, uploaderPubkey: string, mimeType: string, extension: string, size: number, createdAt: string, width?: number | null, height?: number | null, durationMS?: number | null, bitrateKbps?: number | null, thumbnailURL?: string | null, directURL: string, optimizedURL?: string | null, reviewState: string, exifStatus: string, gpsDetected: boolean, downloadCount: number, ingressBytes: number, egressBytes: number, flagReason?: string | null, mirrors: Array<string>, blossomID?: string | null, reportCount: number, nip94Tags: Array<{ __typename: 'NostrTag', values: Array<string> }> }> } };

export type BlossomObjectQueryVariables = Exact<{
  hash: Scalars['ID']['input'];
}>;


export type BlossomObjectQuery = { __typename: 'Query', blossomObject: { __typename: 'AdminBlossomObject', hash: string, uploaderPubkey: string, mimeType: string, extension: string, size: number, createdAt: string, width?: number | null, height?: number | null, durationMS?: number | null, bitrateKbps?: number | null, thumbnailURL?: string | null, directURL: string, optimizedURL?: string | null, reviewState: string, exifStatus: string, gpsDetected: boolean, downloadCount: number, ingressBytes: number, egressBytes: number, flagReason?: string | null, mirrors: Array<string>, blossomID?: string | null, reportCount: number, nip94Tags: Array<{ __typename: 'NostrTag', values: Array<string> }> } };

export type ReviewBlossomObjectsMutationVariables = Exact<{
  input: AdminBlossomBulkReviewInput;
}>;


export type ReviewBlossomObjectsMutation = { __typename: 'Mutation', reviewBlossomObjects: { __typename: 'MutationAck', ok: boolean } };

export type BlossomUsersQueryVariables = Exact<{
  filter?: InputMaybe<AdminBlossomUserFilterInput>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type BlossomUsersQuery = { __typename: 'Query', blossomUsers: { __typename: 'AdminBlossomUserPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminBlossomUser', pubkey: string, displayName?: string | null, picture?: string | null, npub?: string | null, objectCount: number, storageUsedBytes: number, storageQuotaBytes?: number | null, monthlyEgressBytes: number, egressQuotaBytes?: number | null, enabled: boolean, lastUploadAt?: string | null, notes?: string | null }> } };

export type BlossomUserQueryVariables = Exact<{
  pubkey: Scalars['String']['input'];
}>;


export type BlossomUserQuery = { __typename: 'Query', blossomUser: { __typename: 'AdminBlossomUserDetail', user: { __typename: 'AdminBlossomUser', pubkey: string, displayName?: string | null, picture?: string | null, npub?: string | null, objectCount: number, storageUsedBytes: number, storageQuotaBytes?: number | null, monthlyEgressBytes: number, egressQuotaBytes?: number | null, enabled: boolean, lastUploadAt?: string | null, notes?: string | null }, files: Array<{ __typename: 'AdminBlossomObject', hash: string, uploaderPubkey: string, mimeType: string, extension: string, size: number, createdAt: string, width?: number | null, height?: number | null, durationMS?: number | null, bitrateKbps?: number | null, thumbnailURL?: string | null, directURL: string, optimizedURL?: string | null, reviewState: string, exifStatus: string, gpsDetected: boolean, downloadCount: number }> } };

export type UpsertBlossomWhitelistMutationVariables = Exact<{
  input: AdminBlossomWhitelistInput;
}>;


export type UpsertBlossomWhitelistMutation = { __typename: 'Mutation', upsertBlossomWhitelist: { __typename: 'AdminBlossomUser', pubkey: string, displayName?: string | null, picture?: string | null, npub?: string | null, objectCount: number, storageUsedBytes: number, storageQuotaBytes?: number | null, monthlyEgressBytes: number, egressQuotaBytes?: number | null, enabled: boolean, lastUploadAt?: string | null, notes?: string | null } };

export type PurgeBlossomUserMutationVariables = Exact<{
  pubkey: Scalars['String']['input'];
}>;


export type PurgeBlossomUserMutation = { __typename: 'Mutation', purgeBlossomUser: { __typename: 'MutationAck', ok: boolean } };

export type MirrorBlossomObjectMutationVariables = Exact<{
  input: AdminBlossomMirrorInput;
}>;


export type MirrorBlossomObjectMutation = { __typename: 'Mutation', mirrorBlossomObject: { __typename: 'AdminAsyncJob', status: string, jobId?: string | null } };

export type BlossomWorkersQueryVariables = Exact<{
  status?: InputMaybe<Scalars['String']['input']>;
  jobType?: InputMaybe<Scalars['String']['input']>;
  targetHash?: InputMaybe<Scalars['String']['input']>;
}>;


export type BlossomWorkersQuery = { __typename: 'Query', blossomWorkers: { __typename: 'AdminBlossomWorkerPage', items: Array<Record<string, unknown>> } };

export type BlossomReportsQueryVariables = Exact<{
  filter?: InputMaybe<AdminBlossomReportFilterInput>;
  page?: InputMaybe<OffsetPageInput>;
}>;


export type BlossomReportsQuery = { __typename: 'Query', blossomReports: { __typename: 'AdminBlossomReportPage', pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean }, items: Array<{ __typename: 'AdminBlossomReport', id: string, eventID: string, objectHash: string, reporterPubkey: string, reporterNpub?: string | null, targetEventID?: string | null, targetPubkey?: string | null, reportType?: string | null, reason?: string | null, status: BlossomReportStatus, resolvedBy?: string | null, resolvedNote?: string | null, createdAt?: string | null, resolvedAt?: string | null }> } };

export type ResolveBlossomReportMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: AdminBlossomResolveReportInput;
}>;


export type ResolveBlossomReportMutation = { __typename: 'Mutation', resolveBlossomReport: { __typename: 'AdminBlossomReport', id: string, status: BlossomReportStatus } };

export type BlossomAnalyticsQueryVariables = Exact<{ [key: string]: never; }>;


export type BlossomAnalyticsQuery = { __typename: 'Query', blossomAnalytics: Record<string, unknown> };

export type BlossomAuditQueryVariables = Exact<{
  page?: InputMaybe<OffsetPageInput>;
}>;


export type BlossomAuditQuery = { __typename: 'Query', blossomAudit: { __typename: 'AdminBlossomAuditPage', items: Array<Record<string, unknown>>, pageInfo: { __typename: 'PageInfo', total: number, limit: number, offset: number, hasMore: boolean } } };


export const AdminOverviewDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AdminOverview"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"adminOverview"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"activeConnections"}},{"kind":"Field","name":{"kind":"Name","value":"authedConnections"}},{"kind":"Field","name":{"kind":"Name","value":"loggedUsers"}},{"kind":"Field","name":{"kind":"Name","value":"bannedUsers"}},{"kind":"Field","name":{"kind":"Name","value":"indexedEvents"}},{"kind":"Field","name":{"kind":"Name","value":"eventsPerMinute"}},{"kind":"Field","name":{"kind":"Name","value":"relayStatus"}}]}}]}}]} as unknown as DocumentNode<AdminOverviewQuery, AdminOverviewQueryVariables>;
export const AdminStreamStatusDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AdminStreamStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"adminStreamStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"config"}},{"kind":"Field","name":{"kind":"Name","value":"dispatcher"}},{"kind":"Field","name":{"kind":"Name","value":"pool"}},{"kind":"Field","name":{"kind":"Name","value":"counters"}}]}}]}}]} as unknown as DocumentNode<AdminStreamStatusQuery, AdminStreamStatusQueryVariables>;
export const PrivacyStatusDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"PrivacyStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"privacyStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"persistence"}},{"kind":"Field","name":{"kind":"Name","value":"stateDir"}},{"kind":"Field","name":{"kind":"Name","value":"networks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"mode"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"started"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"addresses"}},{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"txBytes"}},{"kind":"Field","name":{"kind":"Name","value":"rxBytes"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}},{"kind":"Field","name":{"kind":"Name","value":"connections"}}]}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"uptimeMs"}}]}}]}}]}}]} as unknown as DocumentNode<PrivacyStatusQuery, PrivacyStatusQueryVariables>;
export const ActiveConnectionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ActiveConnections"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"activeConnections"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"wsid"}},{"kind":"Field","name":{"kind":"Name","value":"ip"}},{"kind":"Field","name":{"kind":"Name","value":"authed"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionCount"}},{"kind":"Field","name":{"kind":"Name","value":"connectedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastSeenAt"}},{"kind":"Field","name":{"kind":"Name","value":"userAgent"}}]}}]}}]}}]} as unknown as DocumentNode<ActiveConnectionsQuery, ActiveConnectionsQueryVariables>;
export const AuthedConnectionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AuthedConnections"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"authedConnections"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"wsid"}},{"kind":"Field","name":{"kind":"Name","value":"ip"}},{"kind":"Field","name":{"kind":"Name","value":"authed"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionCount"}},{"kind":"Field","name":{"kind":"Name","value":"connectedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastSeenAt"}},{"kind":"Field","name":{"kind":"Name","value":"userAgent"}}]}}]}}]}}]} as unknown as DocumentNode<AuthedConnectionsQuery, AuthedConnectionsQueryVariables>;
export const DisconnectConnectionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DisconnectConnection"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"wsid"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"reason"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"disconnectConnection"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"wsid"},"value":{"kind":"Variable","name":{"kind":"Name","value":"wsid"}}},{"kind":"Argument","name":{"kind":"Name","value":"reason"},"value":{"kind":"Variable","name":{"kind":"Name","value":"reason"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}},{"kind":"Field","name":{"kind":"Name","value":"entityId"}}]}}]}}]} as unknown as DocumentNode<DisconnectConnectionMutation, DisconnectConnectionMutationVariables>;
export const LoggedUsersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"LoggedUsers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"loggedUsers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"profile"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"npub"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"handle"}},{"kind":"Field","name":{"kind":"Name","value":"picture"}},{"kind":"Field","name":{"kind":"Name","value":"nip05"}},{"kind":"Field","name":{"kind":"Name","value":"about"}},{"kind":"Field","name":{"kind":"Name","value":"website"}},{"kind":"Field","name":{"kind":"Name","value":"bot"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"relatedIds"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectionCount"}},{"kind":"Field","name":{"kind":"Name","value":"lastSeenAt"}},{"kind":"Field","name":{"kind":"Name","value":"connectionState"}}]}}]}}]}}]} as unknown as DocumentNode<LoggedUsersQuery, LoggedUsersQueryVariables>;
export const BannedUsersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BannedUsers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"q"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bannedUsers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"q"},"value":{"kind":"Variable","name":{"kind":"Name","value":"q"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"npub"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"handle"}},{"kind":"Field","name":{"kind":"Name","value":"picture"}},{"kind":"Field","name":{"kind":"Name","value":"nip05"}},{"kind":"Field","name":{"kind":"Name","value":"about"}},{"kind":"Field","name":{"kind":"Name","value":"website"}},{"kind":"Field","name":{"kind":"Name","value":"bot"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"relatedIds"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]}}]} as unknown as DocumentNode<BannedUsersQuery, BannedUsersQueryVariables>;
export const SearchUsersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SearchUsers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"q"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"searchUsers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"q"},"value":{"kind":"Variable","name":{"kind":"Name","value":"q"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"npub"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"handle"}},{"kind":"Field","name":{"kind":"Name","value":"picture"}},{"kind":"Field","name":{"kind":"Name","value":"nip05"}},{"kind":"Field","name":{"kind":"Name","value":"about"}},{"kind":"Field","name":{"kind":"Name","value":"website"}},{"kind":"Field","name":{"kind":"Name","value":"bot"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"relatedIds"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]}}]} as unknown as DocumentNode<SearchUsersQuery, SearchUsersQueryVariables>;
export const UserProfileDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"UserProfile"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"userProfile"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"npub"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"handle"}},{"kind":"Field","name":{"kind":"Name","value":"picture"}},{"kind":"Field","name":{"kind":"Name","value":"nip05"}},{"kind":"Field","name":{"kind":"Name","value":"about"}},{"kind":"Field","name":{"kind":"Name","value":"website"}},{"kind":"Field","name":{"kind":"Name","value":"bot"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"relatedIds"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<UserProfileQuery, UserProfileQueryVariables>;
export const UserBanStatusDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"UserBanStatus"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"userBanStatus"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"banned"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"relatedIds"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<UserBanStatusQuery, UserBanStatusQueryVariables>;
export const BanUserDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"BanUser"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminBanUserInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"banUser"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"banned"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"relatedIds"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<BanUserMutation, BanUserMutationVariables>;
export const UnbanUserDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnbanUser"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unbanUser"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"banned"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"relatedIds"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<UnbanUserMutation, UnbanUserMutationVariables>;
export const EventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Events"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminEventSearchInput"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"tags"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"values"}}]}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"sig"}}]}}]}}]}}]} as unknown as DocumentNode<EventsQuery, EventsQueryVariables>;
export const EventAggregatesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"EventAggregates"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminEventSearchInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventAggregates"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"kinds"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}},{"kind":"Field","name":{"kind":"Name","value":"topAuthors"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}},{"kind":"Field","name":{"kind":"Name","value":"topTags"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"tag"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}},{"kind":"Field","name":{"kind":"Name","value":"trends"}}]}}]}}]} as unknown as DocumentNode<EventAggregatesQuery, EventAggregatesQueryVariables>;
export const EventTimelineDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"EventTimeline"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminEventSearchInput"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"bucket"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"TimelineBucket"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventTimeline"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}},{"kind":"Argument","name":{"kind":"Name","value":"bucket"},"value":{"kind":"Variable","name":{"kind":"Name","value":"bucket"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"points"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ts"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]} as unknown as DocumentNode<EventTimelineQuery, EventTimelineQueryVariables>;
export const EventDetailDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"EventDetail"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventDetail"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"event"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"tags"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"values"}}]}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"sig"}}]}},{"kind":"Field","name":{"kind":"Name","value":"identifiers"}},{"kind":"Field","name":{"kind":"Name","value":"author"}},{"kind":"Field","name":{"kind":"Name","value":"hashtags"}},{"kind":"Field","name":{"kind":"Name","value":"imageUrls"}}]}}]}}]} as unknown as DocumentNode<EventDetailQuery, EventDetailQueryVariables>;
export const FetchEventFromRelaysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"FetchEventFromRelays"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminFetchEventInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"fetchEventFromRelays"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventId"}},{"kind":"Field","name":{"kind":"Name","value":"sourceRelay"}},{"kind":"Field","name":{"kind":"Name","value":"found"}},{"kind":"Field","name":{"kind":"Name","value":"persisted"}},{"kind":"Field","name":{"kind":"Name","value":"relaysTried"}},{"kind":"Field","name":{"kind":"Name","value":"relayResults"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"relay"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"error"}}]}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<FetchEventFromRelaysMutation, FetchEventFromRelaysMutationVariables>;
export const EventReportsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"EventReports"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventReports"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"reportEventId"}},{"kind":"Field","name":{"kind":"Name","value":"reporterPubkey"}},{"kind":"Field","name":{"kind":"Name","value":"reporterNpub"}},{"kind":"Field","name":{"kind":"Name","value":"reporterDisplayName"}},{"kind":"Field","name":{"kind":"Name","value":"reporterPicture"}},{"kind":"Field","name":{"kind":"Name","value":"reportedEventId"}},{"kind":"Field","name":{"kind":"Name","value":"reportedPubkey"}},{"kind":"Field","name":{"kind":"Name","value":"reportType"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]}}]} as unknown as DocumentNode<EventReportsQuery, EventReportsQueryVariables>;
export const ReportedEventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ReportedEvents"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminReportedEventFilterInput"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"reportedEvents"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"targetEventId"}},{"kind":"Field","name":{"kind":"Name","value":"targetPubkey"}},{"kind":"Field","name":{"kind":"Name","value":"targetNevent"}},{"kind":"Field","name":{"kind":"Name","value":"targetCreatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"targetCreatedAtIso"}},{"kind":"Field","name":{"kind":"Name","value":"targetAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"reportCount"}},{"kind":"Field","name":{"kind":"Name","value":"lastReported"}},{"kind":"Field","name":{"kind":"Name","value":"lastReportedAt"}},{"kind":"Field","name":{"kind":"Name","value":"reportTypes"}},{"kind":"Field","name":{"kind":"Name","value":"targetEvent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"tags"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"values"}}]}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"sig"}}]}}]}}]}}]}}]} as unknown as DocumentNode<ReportedEventsQuery, ReportedEventsQueryVariables>;
export const ReportedEventsSummaryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ReportedEventsSummary"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminReportedEventFilterInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"reportedEventsSummary"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalEvents"}},{"kind":"Field","name":{"kind":"Name","value":"totalReports"}},{"kind":"Field","name":{"kind":"Name","value":"uniqueTargetAuthors"}},{"kind":"Field","name":{"kind":"Name","value":"timeline"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}},{"kind":"Field","name":{"kind":"Name","value":"reportTypes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}},{"kind":"Field","name":{"kind":"Name","value":"topAuthors"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}},{"kind":"Field","name":{"kind":"Name","value":"topTargets"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"targetEventId"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]} as unknown as DocumentNode<ReportedEventsSummaryQuery, ReportedEventsSummaryQueryVariables>;
export const LabelsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Labels"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminLabelFilterInput"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"labels"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"authorNpub"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"namespace"}},{"kind":"Field","name":{"kind":"Name","value":"labels"}},{"kind":"Field","name":{"kind":"Name","value":"target"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"relayHint"}}]}},{"kind":"Field","name":{"kind":"Name","value":"tags"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"values"}}]}}]}}]}}]}}]} as unknown as DocumentNode<LabelsQuery, LabelsQueryVariables>;
export const LabelsSummaryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"LabelsSummary"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminLabelFilterInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"labelsSummary"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalEvents"}},{"kind":"Field","name":{"kind":"Name","value":"totalTargets"}},{"kind":"Field","name":{"kind":"Name","value":"namespaces"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}},{"kind":"Field","name":{"kind":"Name","value":"labels"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}},{"kind":"Field","name":{"kind":"Name","value":"targetTypes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]} as unknown as DocumentNode<LabelsSummaryQuery, LabelsSummaryQueryVariables>;
export const CreateLabelDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateLabel"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminCreateLabelInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createLabel"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"event"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"tags"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"values"}}]}},{"kind":"Field","name":{"kind":"Name","value":"sig"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stored"}}]}}]}}]} as unknown as DocumentNode<CreateLabelMutation, CreateLabelMutationVariables>;
export const Nip05IdentitiesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Nip05Identities"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"q"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nip05Identities"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"q"},"value":{"kind":"Variable","name":{"kind":"Name","value":"q"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"npub"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"picture"}},{"kind":"Field","name":{"kind":"Name","value":"relayHints"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]}}]} as unknown as DocumentNode<Nip05IdentitiesQuery, Nip05IdentitiesQueryVariables>;
export const UpsertNip05Document = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpsertNip05"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminNip05UpsertInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"upsertNip05"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"npub"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"picture"}},{"kind":"Field","name":{"kind":"Name","value":"relayHints"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<UpsertNip05Mutation, UpsertNip05MutationVariables>;
export const DeleteNip05Document = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteNip05"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteNip05"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}},{"kind":"Field","name":{"kind":"Name","value":"entityId"}}]}}]}}]} as unknown as DocumentNode<DeleteNip05Mutation, DeleteNip05MutationVariables>;
export const UserNip05Document = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"UserNip05"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"userNip05"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"exists"}},{"kind":"Field","name":{"kind":"Name","value":"identity"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"npub"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"picture"}},{"kind":"Field","name":{"kind":"Name","value":"relayHints"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]}}]} as unknown as DocumentNode<UserNip05Query, UserNip05QueryVariables>;
export const Nip86AllowedPubkeysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Nip86AllowedPubkeys"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"q"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nip86AllowedPubkeys"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"q"},"value":{"kind":"Variable","name":{"kind":"Name","value":"q"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"createdBy"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]}}]} as unknown as DocumentNode<Nip86AllowedPubkeysQuery, Nip86AllowedPubkeysQueryVariables>;
export const CreateNip86AllowedPubkeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateNip86AllowedPubkey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminNip86ReasonInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createNip86AllowedPubkey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<CreateNip86AllowedPubkeyMutation, CreateNip86AllowedPubkeyMutationVariables>;
export const DeleteNip86AllowedPubkeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteNip86AllowedPubkey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteNip86AllowedPubkey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<DeleteNip86AllowedPubkeyMutation, DeleteNip86AllowedPubkeyMutationVariables>;
export const Nip86BlockedIpsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Nip86BlockedIps"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"q"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nip86BlockedIps"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"q"},"value":{"kind":"Variable","name":{"kind":"Name","value":"q"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ip"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"createdBy"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]}}]} as unknown as DocumentNode<Nip86BlockedIpsQuery, Nip86BlockedIpsQueryVariables>;
export const CreateNip86BlockedIpDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateNip86BlockedIp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ip"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminNip86ReasonInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createNip86BlockedIp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ip"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ip"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<CreateNip86BlockedIpMutation, CreateNip86BlockedIpMutationVariables>;
export const DeleteNip86BlockedIpDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteNip86BlockedIp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ip"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteNip86BlockedIp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ip"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ip"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<DeleteNip86BlockedIpMutation, DeleteNip86BlockedIpMutationVariables>;
export const Nip86BannedEventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Nip86BannedEvents"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"q"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nip86BannedEvents"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"q"},"value":{"kind":"Variable","name":{"kind":"Name","value":"q"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventId"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"createdBy"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]}}]} as unknown as DocumentNode<Nip86BannedEventsQuery, Nip86BannedEventsQueryVariables>;
export const CreateNip86BannedEventDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateNip86BannedEvent"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminNip86ReasonInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createNip86BannedEvent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"eventId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventId"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<CreateNip86BannedEventMutation, CreateNip86BannedEventMutationVariables>;
export const DeleteNip86BannedEventDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteNip86BannedEvent"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteNip86BannedEvent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"eventId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<DeleteNip86BannedEventMutation, DeleteNip86BannedEventMutationVariables>;
export const Nip86RelayMetadataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Nip86RelayMetadata"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nip86RelayMetadata"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"relayUrl"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"description"}}]}}]}}]} as unknown as DocumentNode<Nip86RelayMetadataQuery, Nip86RelayMetadataQueryVariables>;
export const UpdateNip86RelayMetadataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateNip86RelayMetadata"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminNip86RelayMetadataInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateNip86RelayMetadata"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"relayUrl"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"description"}}]}}]}}]} as unknown as DocumentNode<UpdateNip86RelayMetadataMutation, UpdateNip86RelayMetadataMutationVariables>;
export const StartNegentropySyncDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"StartNegentropySync"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminNegentropySyncInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startNegentropySync"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"jobId"}},{"kind":"Field","name":{"kind":"Name","value":"remote"}},{"kind":"Field","name":{"kind":"Name","value":"relays"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<StartNegentropySyncMutation, StartNegentropySyncMutationVariables>;
export const DownloadEventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DownloadEvents"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminDownloadEventsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"downloadEvents"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"jobId"}},{"kind":"Field","name":{"kind":"Name","value":"relays"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<DownloadEventsMutation, DownloadEventsMutationVariables>;
export const DownloadJobsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DownloadJobs"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jobs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"jobName"},"value":{"kind":"StringValue","value":"download.events","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"limit"},"value":{"kind":"IntValue","value":"100"}},{"kind":"ObjectField","name":{"kind":"Name","value":"offset"},"value":{"kind":"IntValue","value":"0"}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"payload"}},{"kind":"Field","name":{"kind":"Name","value":"result"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}}]}}]}}]}}]} as unknown as DocumentNode<DownloadJobsQuery, DownloadJobsQueryVariables>;
export const DownloadJobDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DownloadJob"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"job"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"payload"}},{"kind":"Field","name":{"kind":"Name","value":"result"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}}]}}]}}]} as unknown as DocumentNode<DownloadJobQuery, DownloadJobQueryVariables>;
export const JobsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Jobs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminJobFilterInput"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jobs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"queue"}},{"kind":"Field","name":{"kind":"Name","value":"priority"}},{"kind":"Field","name":{"kind":"Name","value":"jobName"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"attempts"}},{"kind":"Field","name":{"kind":"Name","value":"maxAttempts"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"runAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"payload"}},{"kind":"Field","name":{"kind":"Name","value":"result"}}]}}]}}]}}]} as unknown as DocumentNode<JobsQuery, JobsQueryVariables>;
export const JobDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Job"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"queue"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"job"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"queue"},"value":{"kind":"Variable","name":{"kind":"Name","value":"queue"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"queue"}},{"kind":"Field","name":{"kind":"Name","value":"priority"}},{"kind":"Field","name":{"kind":"Name","value":"jobName"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"attempts"}},{"kind":"Field","name":{"kind":"Name","value":"maxAttempts"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"runAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"payload"}},{"kind":"Field","name":{"kind":"Name","value":"result"}}]}}]}}]} as unknown as DocumentNode<JobQuery, JobQueryVariables>;
export const RetryJobDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RetryJob"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"queue"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"retryJob"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"queue"},"value":{"kind":"Variable","name":{"kind":"Name","value":"queue"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<RetryJobMutation, RetryJobMutationVariables>;
export const ResumeJobDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ResumeJob"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"queue"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"resumeJob"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"queue"},"value":{"kind":"Variable","name":{"kind":"Name","value":"queue"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<ResumeJobMutation, ResumeJobMutationVariables>;
export const CancelJobDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CancelJob"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"queue"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancelJob"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"queue"},"value":{"kind":"Variable","name":{"kind":"Name","value":"queue"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<CancelJobMutation, CancelJobMutationVariables>;
export const DeleteJobsHistoryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteJobsHistory"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminDeleteJobsHistoryInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteJobsHistory"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<DeleteJobsHistoryMutation, DeleteJobsHistoryMutationVariables>;
export const GroupsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Groups"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"groups"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"groupId"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"private"}},{"kind":"Field","name":{"kind":"Name","value":"closed"}},{"kind":"Field","name":{"kind":"Name","value":"hidden"}},{"kind":"Field","name":{"kind":"Name","value":"memberCount"}}]}}]}}]} as unknown as DocumentNode<GroupsQuery, GroupsQueryVariables>;
export const WotSummaryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"WotSummary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"wotSummary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalNodes"}},{"kind":"Field","name":{"kind":"Name","value":"totalEdges"}},{"kind":"Field","name":{"kind":"Name","value":"trustedPubkeys"}},{"kind":"Field","name":{"kind":"Name","value":"lastComputedAt"}}]}}]}}]} as unknown as DocumentNode<WotSummaryQuery, WotSummaryQueryVariables>;
export const AddTrustedPubkeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddTrustedPubkey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"addTrustedPubkey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"trustedPubkeys"}}]}}]}}]} as unknown as DocumentNode<AddTrustedPubkeyMutation, AddTrustedPubkeyMutationVariables>;
export const RemoveTrustedPubkeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RemoveTrustedPubkey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"removeTrustedPubkey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"trustedPubkeys"}}]}}]}}]} as unknown as DocumentNode<RemoveTrustedPubkeyMutation, RemoveTrustedPubkeyMutationVariables>;
export const BlossomOverviewDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BlossomOverview"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"blossomOverview"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"storage"}},{"kind":"Field","name":{"kind":"Name","value":"objects"}},{"kind":"Field","name":{"kind":"Name","value":"traffic"}},{"kind":"Field","name":{"kind":"Name","value":"users"}},{"kind":"Field","name":{"kind":"Name","value":"workers"}},{"kind":"Field","name":{"kind":"Name","value":"policy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"mode"}},{"kind":"Field","name":{"kind":"Name","value":"defaultStorageQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"defaultEgressQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"enabledUserDefaultStorageQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"enabledUserDefaultEgressQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"alerts"}}]}}]}}]} as unknown as DocumentNode<BlossomOverviewQuery, BlossomOverviewQueryVariables>;
export const BlossomPolicyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BlossomPolicy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"blossomPolicy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"mode"}},{"kind":"Field","name":{"kind":"Name","value":"defaultStorageQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"defaultEgressQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"enabledUserDefaultStorageQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"enabledUserDefaultEgressQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<BlossomPolicyQuery, BlossomPolicyQueryVariables>;
export const UpsertBlossomPolicyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpsertBlossomPolicy"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminBlossomPolicyInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"upsertBlossomPolicy"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"mode"}},{"kind":"Field","name":{"kind":"Name","value":"defaultStorageQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"defaultEgressQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"enabledUserDefaultStorageQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"enabledUserDefaultEgressQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<UpsertBlossomPolicyMutation, UpsertBlossomPolicyMutationVariables>;
export const BlossomPlansDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BlossomPlans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"blossomPlans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"scope"}},{"kind":"Field","name":{"kind":"Name","value":"storageQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"egressQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"isDefault"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<BlossomPlansQuery, BlossomPlansQueryVariables>;
export const UpsertBlossomPlanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpsertBlossomPlan"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminBlossomPlanInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"upsertBlossomPlan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"scope"}},{"kind":"Field","name":{"kind":"Name","value":"storageQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"egressQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"isDefault"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<UpsertBlossomPlanMutation, UpsertBlossomPlanMutationVariables>;
export const DeleteBlossomPlanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteBlossomPlan"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteBlossomPlan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<DeleteBlossomPlanMutation, DeleteBlossomPlanMutationVariables>;
export const AssignBlossomPlanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AssignBlossomPlan"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"planId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"assignBlossomPlan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"planId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"planId"}}},{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<AssignBlossomPlanMutation, AssignBlossomPlanMutationVariables>;
export const UnassignBlossomPlanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnassignBlossomPlan"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"planId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unassignBlossomPlan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"planId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"planId"}}},{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<UnassignBlossomPlanMutation, UnassignBlossomPlanMutationVariables>;
export const BlossomPlanAssignmentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BlossomPlanAssignments"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"planId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"blossomPlanAssignments"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"planId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"planId"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"limit"},"value":{"kind":"IntValue","value":"250"}},{"kind":"ObjectField","name":{"kind":"Name","value":"offset"},"value":{"kind":"IntValue","value":"0"}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"planId"}},{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"picture"}},{"kind":"Field","name":{"kind":"Name","value":"npub"}},{"kind":"Field","name":{"kind":"Name","value":"assignedBy"}},{"kind":"Field","name":{"kind":"Name","value":"assignedAt"}}]}}]}}]}}]} as unknown as DocumentNode<BlossomPlanAssignmentsQuery, BlossomPlanAssignmentsQueryVariables>;
export const BlossomObjectsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BlossomObjects"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminBlossomObjectFilterInput"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"blossomObjects"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hash"}},{"kind":"Field","name":{"kind":"Name","value":"uploaderPubkey"}},{"kind":"Field","name":{"kind":"Name","value":"mimeType"}},{"kind":"Field","name":{"kind":"Name","value":"extension"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"width"}},{"kind":"Field","name":{"kind":"Name","value":"height"}},{"kind":"Field","name":{"kind":"Name","value":"durationMS"}},{"kind":"Field","name":{"kind":"Name","value":"bitrateKbps"}},{"kind":"Field","name":{"kind":"Name","value":"thumbnailURL"}},{"kind":"Field","name":{"kind":"Name","value":"directURL"}},{"kind":"Field","name":{"kind":"Name","value":"optimizedURL"}},{"kind":"Field","name":{"kind":"Name","value":"reviewState"}},{"kind":"Field","name":{"kind":"Name","value":"exifStatus"}},{"kind":"Field","name":{"kind":"Name","value":"gpsDetected"}},{"kind":"Field","name":{"kind":"Name","value":"downloadCount"}},{"kind":"Field","name":{"kind":"Name","value":"ingressBytes"}},{"kind":"Field","name":{"kind":"Name","value":"egressBytes"}},{"kind":"Field","name":{"kind":"Name","value":"flagReason"}},{"kind":"Field","name":{"kind":"Name","value":"mirrors"}},{"kind":"Field","name":{"kind":"Name","value":"blossomID"}},{"kind":"Field","name":{"kind":"Name","value":"reportCount"}},{"kind":"Field","name":{"kind":"Name","value":"nip94Tags"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"values"}}]}}]}}]}}]}}]} as unknown as DocumentNode<BlossomObjectsQuery, BlossomObjectsQueryVariables>;
export const BlossomObjectDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BlossomObject"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"hash"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"blossomObject"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"hash"},"value":{"kind":"Variable","name":{"kind":"Name","value":"hash"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hash"}},{"kind":"Field","name":{"kind":"Name","value":"uploaderPubkey"}},{"kind":"Field","name":{"kind":"Name","value":"mimeType"}},{"kind":"Field","name":{"kind":"Name","value":"extension"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"width"}},{"kind":"Field","name":{"kind":"Name","value":"height"}},{"kind":"Field","name":{"kind":"Name","value":"durationMS"}},{"kind":"Field","name":{"kind":"Name","value":"bitrateKbps"}},{"kind":"Field","name":{"kind":"Name","value":"thumbnailURL"}},{"kind":"Field","name":{"kind":"Name","value":"directURL"}},{"kind":"Field","name":{"kind":"Name","value":"optimizedURL"}},{"kind":"Field","name":{"kind":"Name","value":"reviewState"}},{"kind":"Field","name":{"kind":"Name","value":"exifStatus"}},{"kind":"Field","name":{"kind":"Name","value":"gpsDetected"}},{"kind":"Field","name":{"kind":"Name","value":"downloadCount"}},{"kind":"Field","name":{"kind":"Name","value":"ingressBytes"}},{"kind":"Field","name":{"kind":"Name","value":"egressBytes"}},{"kind":"Field","name":{"kind":"Name","value":"flagReason"}},{"kind":"Field","name":{"kind":"Name","value":"mirrors"}},{"kind":"Field","name":{"kind":"Name","value":"blossomID"}},{"kind":"Field","name":{"kind":"Name","value":"reportCount"}},{"kind":"Field","name":{"kind":"Name","value":"nip94Tags"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"values"}}]}}]}}]}}]} as unknown as DocumentNode<BlossomObjectQuery, BlossomObjectQueryVariables>;
export const ReviewBlossomObjectsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ReviewBlossomObjects"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminBlossomBulkReviewInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"reviewBlossomObjects"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<ReviewBlossomObjectsMutation, ReviewBlossomObjectsMutationVariables>;
export const BlossomUsersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BlossomUsers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminBlossomUserFilterInput"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"blossomUsers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"picture"}},{"kind":"Field","name":{"kind":"Name","value":"npub"}},{"kind":"Field","name":{"kind":"Name","value":"objectCount"}},{"kind":"Field","name":{"kind":"Name","value":"storageUsedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"storageQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"monthlyEgressBytes"}},{"kind":"Field","name":{"kind":"Name","value":"egressQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"lastUploadAt"}},{"kind":"Field","name":{"kind":"Name","value":"notes"}}]}}]}}]}}]} as unknown as DocumentNode<BlossomUsersQuery, BlossomUsersQueryVariables>;
export const BlossomUserDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BlossomUser"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"blossomUser"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"picture"}},{"kind":"Field","name":{"kind":"Name","value":"npub"}},{"kind":"Field","name":{"kind":"Name","value":"objectCount"}},{"kind":"Field","name":{"kind":"Name","value":"storageUsedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"storageQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"monthlyEgressBytes"}},{"kind":"Field","name":{"kind":"Name","value":"egressQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"lastUploadAt"}},{"kind":"Field","name":{"kind":"Name","value":"notes"}}]}},{"kind":"Field","name":{"kind":"Name","value":"files"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hash"}},{"kind":"Field","name":{"kind":"Name","value":"uploaderPubkey"}},{"kind":"Field","name":{"kind":"Name","value":"mimeType"}},{"kind":"Field","name":{"kind":"Name","value":"extension"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"width"}},{"kind":"Field","name":{"kind":"Name","value":"height"}},{"kind":"Field","name":{"kind":"Name","value":"durationMS"}},{"kind":"Field","name":{"kind":"Name","value":"bitrateKbps"}},{"kind":"Field","name":{"kind":"Name","value":"thumbnailURL"}},{"kind":"Field","name":{"kind":"Name","value":"directURL"}},{"kind":"Field","name":{"kind":"Name","value":"optimizedURL"}},{"kind":"Field","name":{"kind":"Name","value":"reviewState"}},{"kind":"Field","name":{"kind":"Name","value":"exifStatus"}},{"kind":"Field","name":{"kind":"Name","value":"gpsDetected"}},{"kind":"Field","name":{"kind":"Name","value":"downloadCount"}}]}}]}}]}}]} as unknown as DocumentNode<BlossomUserQuery, BlossomUserQueryVariables>;
export const UpsertBlossomWhitelistDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpsertBlossomWhitelist"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminBlossomWhitelistInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"upsertBlossomWhitelist"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pubkey"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"picture"}},{"kind":"Field","name":{"kind":"Name","value":"npub"}},{"kind":"Field","name":{"kind":"Name","value":"objectCount"}},{"kind":"Field","name":{"kind":"Name","value":"storageUsedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"storageQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"monthlyEgressBytes"}},{"kind":"Field","name":{"kind":"Name","value":"egressQuotaBytes"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"lastUploadAt"}},{"kind":"Field","name":{"kind":"Name","value":"notes"}}]}}]}}]} as unknown as DocumentNode<UpsertBlossomWhitelistMutation, UpsertBlossomWhitelistMutationVariables>;
export const PurgeBlossomUserDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"PurgeBlossomUser"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"purgeBlossomUser"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"pubkey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pubkey"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}}]}}]}}]} as unknown as DocumentNode<PurgeBlossomUserMutation, PurgeBlossomUserMutationVariables>;
export const MirrorBlossomObjectDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"MirrorBlossomObject"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminBlossomMirrorInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"mirrorBlossomObject"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"jobId"}}]}}]}}]} as unknown as DocumentNode<MirrorBlossomObjectMutation, MirrorBlossomObjectMutationVariables>;
export const BlossomWorkersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BlossomWorkers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"jobType"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"targetHash"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"blossomWorkers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}},{"kind":"Argument","name":{"kind":"Name","value":"jobType"},"value":{"kind":"Variable","name":{"kind":"Name","value":"jobType"}}},{"kind":"Argument","name":{"kind":"Name","value":"targetHash"},"value":{"kind":"Variable","name":{"kind":"Name","value":"targetHash"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"limit"},"value":{"kind":"IntValue","value":"250"}},{"kind":"ObjectField","name":{"kind":"Name","value":"offset"},"value":{"kind":"IntValue","value":"0"}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"}}]}}]}}]} as unknown as DocumentNode<BlossomWorkersQuery, BlossomWorkersQueryVariables>;
export const BlossomReportsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BlossomReports"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminBlossomReportFilterInput"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"blossomReports"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"eventID"}},{"kind":"Field","name":{"kind":"Name","value":"objectHash"}},{"kind":"Field","name":{"kind":"Name","value":"reporterPubkey"}},{"kind":"Field","name":{"kind":"Name","value":"reporterNpub"}},{"kind":"Field","name":{"kind":"Name","value":"targetEventID"}},{"kind":"Field","name":{"kind":"Name","value":"targetPubkey"}},{"kind":"Field","name":{"kind":"Name","value":"reportType"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"resolvedBy"}},{"kind":"Field","name":{"kind":"Name","value":"resolvedNote"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"resolvedAt"}}]}}]}}]}}]} as unknown as DocumentNode<BlossomReportsQuery, BlossomReportsQueryVariables>;
export const ResolveBlossomReportDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ResolveBlossomReport"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AdminBlossomResolveReportInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"resolveBlossomReport"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}}]}}]} as unknown as DocumentNode<ResolveBlossomReportMutation, ResolveBlossomReportMutationVariables>;
export const BlossomAnalyticsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BlossomAnalytics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"blossomAnalytics"}}]}}]} as unknown as DocumentNode<BlossomAnalyticsQuery, BlossomAnalyticsQueryVariables>;
export const BlossomAuditDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BlossomAudit"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"OffsetPageInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"blossomAudit"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"hasMore"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"}}]}}]}}]} as unknown as DocumentNode<BlossomAuditQuery, BlossomAuditQueryVariables>;