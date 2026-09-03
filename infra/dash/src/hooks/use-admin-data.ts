import { useEffect } from "react"

import type {
  AdminJobsFilters,
  AdminLabelsFilters,
  BanPayload,
  BlossomAuditFilters,
  BlossomPlan,
  BlossomPolicy,
  BlossomBulkReviewPayload,
  BlossomObjectsFilters,
  BlossomReportsFilters,
  BlossomResolveReportPayload,
  BlossomUsersFilters,
  BlossomWhitelistPayload,
  BlossomMirrorPayload,
  BlossomWorkersFilters,
  CreateAdminLabelPayload,
  EventSearchFilters,
  NIP86ReasonPayload,
  NIP86RelayMetadataPayload,
} from "@/types/admin"
import {
  getPrivacyStatus,
  getRelayOverview,
  getStreamStatus,
} from "@/services/admin"
import { useInfiniteQuery, useMutation, useQuery, useQueryClient, useSuspenseQuery } from "@/hooks/query-compat"
import { fetchEventFromRelays, getEventDetail, getEventTags, importEventsFiles } from "@/services/admin-event-detail"
import { getEventSearchAggregates, getEventSearchTimeline, searchEventsPage } from "@/services/admin-event-search"
import { createLabel, getLabelsPage, getLabelsSummary } from "@/services/admin-labels"
import { addTrustedPubkey, cancelJob, deleteJobsHistory, getDownloadJob, getDownloadJobs, getGroupsPage, getJob, getJobs, getWoTSummary, removeTrustedPubkey, resumeJob, retryJob, startDownloadEvents, startNegentropySync } from "@/services/admin-jobs-wot"
import { allowNIP86PubKey, banNIP86Event, blockNIP86IP, deleteNIP05Identity, getNIP05Page, getNIP86AllowedPubKeysPage, getNIP86BannedEventsPage, getNIP86BlockedIPsPage, getNIP86RelayMetadata, getUserNIP05, unallowNIP86PubKey, unbanNIP86Event, unblockNIP86IP, updateNIP86RelayMetadata, upsertNIP05Identity } from "@/services/admin-nip05-nip86"
import { getEventReports, getReportedEventsPage, getReportedEventsSummary } from "@/services/admin-reported"
import { assignBlossomPlan, createBlossomMirrorJob, deleteBlossomPlan, getBlossomAnalytics, getBlossomAudit, getBlossomObjectDetail, getBlossomObjects, getBlossomOverview, getBlossomPlanAssignments, getBlossomPlans, getBlossomPolicy, getBlossomReports, getBlossomUserDetail, getBlossomUsers, getBlossomWorkers, purgeBlossomUser, resolveBlossomReport, reviewBlossomObjects, unassignBlossomPlan, updateBlossomPolicy, upsertBlossomPlan, upsertBlossomWhitelistEntry } from "@/services/admin-blossom"
import { banUser, disconnectConnection, getBanStatus, getBannedUsersPage, getConnectionsPage, getLoggedUsersPage, getUser, searchUsersPage, unbanUser } from "@/services/admin-users"

const defaultPageSize = 50

export function useRelayOverview() {
  return useQuery({ queryKey: ["relay-overview"], queryFn: getRelayOverview })
}

export function useBlossomOverview() {
  return useQuery({ queryKey: ["blossom-overview"], queryFn: getBlossomOverview })
}

export function useBlossomPolicy() {
  return useQuery({ queryKey: ["blossom-policy"], queryFn: getBlossomPolicy })
}

export function useBlossomPlans(scope?: string) {
  return useQuery({ queryKey: ["blossom-plans", scope ?? "all"], queryFn: () => getBlossomPlans(scope) })
}

export function useInfiniteBlossomObjects(filters: BlossomObjectsFilters) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["blossom-objects", filters],
    queryFn: ({ pageParam }) => getBlossomObjects(filters, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useBlossomObjectDetail(hash: string, enabled = true) {
  return useQuery({
    queryKey: ["blossom-object", hash],
    queryFn: () => getBlossomObjectDetail(hash),
    enabled: enabled && Boolean(hash),
  })
}

export function useInfiniteBlossomUsers(filters: BlossomUsersFilters) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["blossom-users", filters],
    queryFn: ({ pageParam }) => getBlossomUsers(filters, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useBlossomUserDetail(pubkey: string, enabled = true) {
  return useQuery({
    queryKey: ["blossom-user", pubkey],
    queryFn: () => getBlossomUserDetail(pubkey),
    enabled: enabled && Boolean(pubkey),
  })
}

export function useBlossomWorkers(filters: BlossomWorkersFilters) {
  return useQuery({
    queryKey: ["blossom-workers", filters],
    queryFn: () => getBlossomWorkers(filters),
    refetchInterval: (query) => {
      const items = query.state.data ?? []
      return items.some((item) => item.status === "queued" || item.status === "running" || item.status === "delayed") ? 3000 : 8000
    },
  })
}

export function useInfiniteBlossomAudit(filters: BlossomAuditFilters) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["blossom-audit", filters],
    queryFn: ({ pageParam }) => getBlossomAudit(filters, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useInfiniteBlossomReports(filters: BlossomReportsFilters) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["blossom-reports", filters],
    queryFn: ({ pageParam }) => getBlossomReports(filters, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useBlossomAnalytics(enabled = true) {
  return useQuery({ queryKey: ["blossom-analytics"], queryFn: getBlossomAnalytics, enabled })
}

export function useStreamStatus() {
  return useQuery({ queryKey: ["stream-status"], queryFn: getStreamStatus, refetchInterval: 10000 })
}

export function usePrivacyStatus() {
  return useQuery({ queryKey: ["privacy-status"], queryFn: getPrivacyStatus, refetchInterval: 10000 })
}

export function useInfiniteConnections(mode: "active" | "authed") {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["connections", mode],
    queryFn: ({ pageParam }) => getConnectionsPage(mode, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useInfiniteLoggedUsers() {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["logged-users"],
    queryFn: ({ pageParam }) => getLoggedUsersPage({ limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useInfiniteBannedUsers(query: string) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["banned-users", query],
    queryFn: ({ pageParam }) => getBannedUsersPage(query, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useBanStatus(pubkey: string) {
  return useQuery({ queryKey: ["ban-status", pubkey], queryFn: () => getBanStatus(pubkey), enabled: Boolean(pubkey) })
}

export function useUser(pubkey: string) {
  return useQuery({ queryKey: ["user", pubkey], queryFn: () => getUser(pubkey), enabled: Boolean(pubkey) })
}

export function useUserNIP05(pubkey: string) {
  return useQuery({ queryKey: ["user-nip05", pubkey], queryFn: () => getUserNIP05(pubkey), enabled: Boolean(pubkey) })
}

export function useInfiniteEventSearch(filters: EventSearchFilters) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["events-search", filters],
    queryFn: ({ pageParam }) => searchEventsPage(filters, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useEventSearchAggregates(filters: EventSearchFilters) {
  return useQuery({ queryKey: ["events-search-aggregates", filters], queryFn: () => getEventSearchAggregates(filters) })
}

export function useEventSearchTimeline(filters: EventSearchFilters, bucket: "hour" | "day") {
  return useQuery({ queryKey: ["events-search-timeline", filters, bucket], queryFn: () => getEventSearchTimeline(filters, bucket) })
}

export function useEventDetail(eventID: string) {
  return useQuery({ queryKey: ["event-detail", eventID], queryFn: () => getEventDetail(eventID), enabled: Boolean(eventID) })
}

export function useEventDetailSuspense(eventID: string) {
  return useSuspenseQuery({
    queryKey: ["event-detail", eventID],
    queryFn: () => getEventDetail(eventID),
  })
}

export function useEventReports(eventID: string) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["event-reports", eventID],
    queryFn: ({ pageParam }) => getEventReports(eventID, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.reduce((acc, page) => acc + page.items.length, 0)
      return loaded < lastPage.total ? loaded : undefined
    },
    enabled: Boolean(eventID),
  })
}

export function useInfiniteReportedEvents(filters: { query: string; type: string; target_pubkey?: string; target_event_id?: string; since?: number; until?: number }) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["reported-events", filters],
    queryFn: ({ pageParam }) => getReportedEventsPage(filters, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useReportedEventsSummary(filters: { query: string; type: string; target_pubkey?: string; target_event_id?: string; since?: number; until?: number }) {
  return useQuery({
    queryKey: ["reported-events-summary", filters],
    queryFn: () => getReportedEventsSummary(filters),
  })
}

export function useInfiniteLabels(filters: AdminLabelsFilters) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["labels", filters],
    queryFn: ({ pageParam }) => getLabelsPage(filters, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useLabelsSummary(filters: AdminLabelsFilters) {
  return useQuery({
    queryKey: ["labels-summary", filters],
    queryFn: () => getLabelsSummary(filters),
  })
}

export function useInfiniteUserSearch(query: string, options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["users-search", query],
    queryFn: ({ pageParam }) => searchUsersPage(query, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
    enabled,
  })
}

export function useInfiniteNIP05(query: string) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["nip05", query],
    queryFn: ({ pageParam }) => getNIP05Page(query, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useInfiniteNIP86AllowedPubKeys(query: string) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["nip86", "allowed-pubkeys", query],
    queryFn: ({ pageParam }) => getNIP86AllowedPubKeysPage(query, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useInfiniteNIP86BlockedIPs(query: string) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["nip86", "blocked-ips", query],
    queryFn: ({ pageParam }) => getNIP86BlockedIPsPage(query, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useInfiniteNIP86BannedEvents(query: string) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["nip86", "banned-events", query],
    queryFn: ({ pageParam }) => getNIP86BannedEventsPage(query, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useNIP86RelayMetadata() {
  return useQuery({ queryKey: ["nip86", "relay-metadata"], queryFn: getNIP86RelayMetadata })
}

export function useUpsertNIP05Mutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: upsertNIP05Identity,
    onSuccess: async (result) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["nip05"] }),
        queryClient.invalidateQueries({ queryKey: ["users-search"] }),
        queryClient.invalidateQueries({ queryKey: ["user", result.pubkey] }),
        queryClient.invalidateQueries({ queryKey: ["user-nip05", result.pubkey] }),
      ])
    },
  })
}

export function useDeleteNIP05Mutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteNIP05Identity,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["nip05"] }),
        queryClient.invalidateQueries({ queryKey: ["users-search"] }),
        queryClient.invalidateQueries({ queryKey: ["user"] }),
        queryClient.invalidateQueries({ queryKey: ["user-nip05"] }),
      ])
    },
  })
}

export function useFetchEventFromRelaysMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ eventID, relays }: { eventID: string; relays: string[] }) => fetchEventFromRelays(eventID, relays),
    onSuccess: async (_, payload) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["event-detail", payload.eventID] }),
        queryClient.invalidateQueries({ queryKey: ["events-search"] }),
        queryClient.invalidateQueries({ queryKey: ["reported-events"] }),
      ])
    },
  })
}

export function useImportEventsMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (files: File[]) => importEventsFiles(files),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["events-search"] }),
        queryClient.invalidateQueries({ queryKey: ["reported-events"] }),
      ])
    },
  })
}

export function useBanMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: BanPayload) => banUser(payload),
    onSuccess: async (_, payload) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["banned-users"] }),
        queryClient.invalidateQueries({ queryKey: ["relay-overview"] }),
        queryClient.invalidateQueries({ queryKey: ["ban-status", payload.pubkey] }),
        queryClient.invalidateQueries({ queryKey: ["user", payload.pubkey] }),
        queryClient.invalidateQueries({ queryKey: ["users-search"] }),
      ])
    },
  })
}

export function useUnbanMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (pubkey: string) => unbanUser(pubkey),
    onSuccess: async (_, pubkey) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["banned-users"] }),
        queryClient.invalidateQueries({ queryKey: ["relay-overview"] }),
        queryClient.invalidateQueries({ queryKey: ["ban-status", pubkey] }),
        queryClient.invalidateQueries({ queryKey: ["user", pubkey] }),
        queryClient.invalidateQueries({ queryKey: ["users-search"] }),
      ])
    },
  })
}

export function useCreateLabelMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: CreateAdminLabelPayload) => createLabel(payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["labels"] }),
        queryClient.invalidateQueries({ queryKey: ["labels-summary"] }),
        queryClient.invalidateQueries({ queryKey: ["reported-events"] }),
      ])
    },
  })
}

export function useDisconnectConnectionMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (wsid: string) => disconnectConnection(wsid),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["connections"] }),
        queryClient.invalidateQueries({ queryKey: ["relay-overview"] }),
        queryClient.invalidateQueries({ queryKey: ["logged-users"] }),
      ])
    },
  })
}

export function useAllowNIP86PubKeyMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ pubkey, payload }: { pubkey: string; payload: NIP86ReasonPayload }) => allowNIP86PubKey(pubkey, payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["nip86", "allowed-pubkeys"] }),
        queryClient.invalidateQueries({ queryKey: ["relay-overview"] }),
      ])
    },
  })
}

export function useUnallowNIP86PubKeyMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (pubkey: string) => unallowNIP86PubKey(pubkey),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["nip86", "allowed-pubkeys"] }),
        queryClient.invalidateQueries({ queryKey: ["relay-overview"] }),
      ])
    },
  })
}

export function useBlockNIP86IPMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ ip, payload }: { ip: string; payload: NIP86ReasonPayload }) => blockNIP86IP(ip, payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["nip86", "blocked-ips"] }),
        queryClient.invalidateQueries({ queryKey: ["connections"] }),
        queryClient.invalidateQueries({ queryKey: ["relay-overview"] }),
      ])
    },
  })
}

export function useUnblockNIP86IPMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (ip: string) => unblockNIP86IP(ip),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["nip86", "blocked-ips"] }),
        queryClient.invalidateQueries({ queryKey: ["relay-overview"] }),
      ])
    },
  })
}

export function useBanNIP86EventMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ eventID, payload }: { eventID: string; payload: NIP86ReasonPayload }) => banNIP86Event(eventID, payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["nip86", "banned-events"] }),
        queryClient.invalidateQueries({ queryKey: ["events-search"] }),
        queryClient.invalidateQueries({ queryKey: ["reported-events"] }),
      ])
    },
  })
}

export function useUnbanNIP86EventMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (eventID: string) => unbanNIP86Event(eventID),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["nip86", "banned-events"] }),
        queryClient.invalidateQueries({ queryKey: ["events-search"] }),
        queryClient.invalidateQueries({ queryKey: ["reported-events"] }),
      ])
    },
  })
}

export function useUpdateNIP86RelayMetadataMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: NIP86RelayMetadataPayload) => updateNIP86RelayMetadata(payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["nip86", "relay-metadata"] }),
        queryClient.invalidateQueries({ queryKey: ["relay-overview"] }),
      ])
    },
  })
}

export function useBlossomBulkReviewMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: BlossomBulkReviewPayload) => reviewBlossomObjects(payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["blossom-overview"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-objects"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-object"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-audit"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-workers"] }),
      ])
    },
  })
}

export function useUpsertBlossomWhitelistMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: BlossomWhitelistPayload) => upsertBlossomWhitelistEntry(payload),
    onSuccess: async (_, payload) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["blossom-overview"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-users"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-user", payload.pubkey] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-audit"] }),
      ])
    },
  })
}

export function usePurgeBlossomUserMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (pubkey: string) => purgeBlossomUser(pubkey),
    onSuccess: async (_, pubkey) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["blossom-overview"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-users"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-user", pubkey] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-objects"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-audit"] }),
      ])
    },
  })
}

export function useAssignBlossomPlanMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: { plan_id: string; pubkey: string }) => assignBlossomPlan(payload),
    onSuccess: async (_, payload) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["blossom-overview"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-users"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-user", payload.pubkey] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-plan-assignments"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-audit"] }),
      ])
    },
  })
}

export function useUnassignBlossomPlanMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: { plan_id: string; pubkey: string }) => unassignBlossomPlan(payload),
    onSuccess: async (_, payload) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["blossom-overview"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-users"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-user", payload.pubkey] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-plan-assignments"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-audit"] }),
      ])
    },
  })
}

export function useBlossomPlanAssignments(planId: string, enabled = true) {
	return useQuery({
		queryKey: ["blossom-plan-assignments", planId],
		queryFn: () => getBlossomPlanAssignments(planId),
		enabled: enabled && Boolean(planId),
	})
}

export function useCreateBlossomMirrorMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: BlossomMirrorPayload) => createBlossomMirrorJob(payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["blossom-workers"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-audit"] }),
      ])
    },
  })
}

export function useUpdateBlossomPolicyMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: BlossomPolicy) => updateBlossomPolicy(payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["blossom-overview"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-policy"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-users"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-audit"] }),
      ])
    },
  })
}

export function useResolveBlossomReportMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: BlossomResolveReportPayload) => resolveBlossomReport(payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["blossom-reports"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-overview"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-analytics"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-audit"] }),
      ])
    },
  })
}

export function useUpsertBlossomPlanMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: BlossomPlan) => upsertBlossomPlan(payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["blossom-plans"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-policy"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-overview"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-audit"] }),
      ])
    },
  })
}

export function useDeleteBlossomPlanMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteBlossomPlan(id),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["blossom-plans"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-policy"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-overview"] }),
        queryClient.invalidateQueries({ queryKey: ["blossom-audit"] }),
      ])
    },
  })
}

export function useInfiniteGroups() {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["groups"],
    queryFn: ({ pageParam }) => getGroupsPage({ limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
  })
}

export function useWoTSummary() {
  return useQuery({ queryKey: ["wot-summary"], queryFn: getWoTSummary })
}

export function useWoTSummarySuspense() {
  return useSuspenseQuery({
    queryKey: ["wot-summary"],
    queryFn: getWoTSummary,
    retry: false,
  })
}

export function useSyncMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: startNegentropySync,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["jobs"] })
    },
  })
}

export function useDownloadMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: startDownloadEvents,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["events-search"] }),
        queryClient.invalidateQueries({ queryKey: ["download-jobs"] }),
      ])
    },
  })
}

export function useDownloadJobs() {
  return useQuery({
    queryKey: ["download-jobs"],
    queryFn: getDownloadJobs,
    refetchInterval: (query) => {
      const items = query.state.data?.items ?? []
      return items.some((item) => item.status === "queued" || item.status === "running") ? 2500 : 5000
    },
  })
}

export function useDownloadJob(jobID: string, enabled = true) {
  return useQuery({
    queryKey: ["download-job", jobID],
    queryFn: () => getDownloadJob(jobID),
    enabled: enabled && Boolean(jobID),
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === "queued" || status === "running" ? 2500 : false
    },
  })
}

export function useJobsQuery(filters: AdminJobsFilters) {
  return useQuery({
    queryKey: ["jobs", filters],
    queryFn: () => getJobs(filters),
    refetchInterval: (query) => {
      const items = query.state.data?.items ?? []
      return items.some((item) => item.status === "queued" || item.status === "running" || item.status === "delayed") ? 2500 : 8000
    },
  })
}

export function useJobQuery(jobID: string, queue?: string, enabled = true) {
  return useQuery({
    queryKey: ["job", jobID, queue],
    queryFn: () => getJob(jobID, queue),
    enabled: enabled && Boolean(jobID),
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === "queued" || status === "running" || status === "delayed" ? 2500 : false
    },
  })
}

export function useRetryJobMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ jobID, queue }: { jobID: string; queue: string }) => retryJob(jobID, queue),
    onSuccess: async (_, payload) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["jobs"] }),
        queryClient.invalidateQueries({ queryKey: ["job", payload.jobID, payload.queue] }),
      ])
    },
  })
}

export function useCancelJobMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ jobID, queue }: { jobID: string; queue: string }) => cancelJob(jobID, queue),
    onSuccess: async (_, payload) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["jobs"] }),
        queryClient.invalidateQueries({ queryKey: ["job", payload.jobID, payload.queue] }),
      ])
    },
  })
}

export function useResumeJobMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ jobID, queue }: { jobID: string; queue: string }) => resumeJob(jobID, queue),
    onSuccess: async (_, payload) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["jobs"] }),
        queryClient.invalidateQueries({ queryKey: ["job", payload.jobID, payload.queue] }),
      ])
    },
  })
}

export function useDeleteJobsHistoryMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: deleteJobsHistory,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["jobs"] }),
        queryClient.invalidateQueries({ queryKey: ["job"] }),
      ])
    },
  })
}

export function useAddTrustedPubkeyMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: addTrustedPubkey,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["wot-summary"] })
    },
  })
}

export function useRemoveTrustedPubkeyMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: removeTrustedPubkey,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["wot-summary"] })
    },
  })
}

export function isFeatureDisabledError(error: any): boolean {
  if (!error) return false
  const status = error.response?.status
  const message = error.response?.data?.error || ""
  return (status === 404 || status === 503) && (message.includes("disabled") || message.includes("not enabled"))
}
