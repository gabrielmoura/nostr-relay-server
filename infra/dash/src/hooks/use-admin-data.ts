import { useInfiniteQuery, useMutation, useQuery, useQueryClient, useSuspenseQuery } from "@tanstack/react-query"

import type { BanPayload, EventSearchFilters, NIP86ReasonPayload, NIP86RelayMetadataPayload } from "@/types/admin"
import {
  allowNIP86PubKey,
  banUser,
  banNIP86Event,
  blockNIP86IP,
  deleteNIP05Identity,
  disconnectConnection,
  fetchEventFromRelays,
  getNIP86AllowedPubKeysPage,
  getNIP86BannedEventsPage,
  getNIP86BlockedIPsPage,
  getNIP86RelayMetadata,
  getNIP05Page,
  importEventsFiles,
  getEventDetail,
  getEventReports,
  getEventSearchAggregates,
  getEventSearchTimeline,
  getBannedUsersPage,
  getBanStatus,
  getConnectionsPage,
  getLoggedUsersPage,
  getReportedEventsPage,
  getRelayOverview,
  getStreamStatus,
  getUser,
  getUserNIP05,
  searchEventsPage,
  searchUsersPage,
  unallowNIP86PubKey,
  unbanNIP86Event,
  unblockNIP86IP,
  upsertNIP05Identity,
  unbanUser,
  updateNIP86RelayMetadata,
} from "@/services/admin"

const defaultPageSize = 50

export function useRelayOverview() {
  return useQuery({ queryKey: ["relay-overview"], queryFn: getRelayOverview })
}

export function useStreamStatus() {
  return useQuery({ queryKey: ["stream-status"], queryFn: getStreamStatus, refetchInterval: 10000 })
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

export function useInfiniteReportedEvents(query: string, type: string) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["reported-events", query, type],
    queryFn: ({ pageParam }) => getReportedEventsPage(query, type, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
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
