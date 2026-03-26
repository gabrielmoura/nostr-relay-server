import { useInfiniteQuery, useMutation, useQuery, useQueryClient, useSuspenseQuery } from "@tanstack/react-query"

import type { BanPayload, EventSearchFilters } from "@/types/admin"
import {
  banUser,
  disconnectConnection,
  fetchEventFromRelays,
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
  searchEventsPage,
  searchUsersPage,
  unbanUser,
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

export function useInfiniteUserSearch(query: string) {
  return useInfiniteQuery({
    initialPageParam: 0,
    queryKey: ["users-search", query],
    queryFn: ({ pageParam }) => searchUsersPage(query, { limit: defaultPageSize, offset: pageParam }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.offset + lastPage.items.length : undefined),
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
