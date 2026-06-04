import { adminApolloClient } from "@/graphql/client"
import { GraphQLApiError, keysToSnake, pageFromGraph } from "@/graphql/helpers"
import { ActiveConnectionsDocument, AuthedConnectionsDocument, BanUserDocument, BannedUsersDocument, DisconnectConnectionDocument, LoggedUsersDocument, SearchUsersDocument, UnbanUserDocument, UserBanStatusDocument, UserProfileDocument } from "@/graphql/generated/operations"
import { buildNpubLike, toTitleCase } from "@/lib/utils"
import { env } from "@/lib/env"
import { mockConnections, mockUsers, seedBannedUsers } from "@/mocks/admin"
import type { AdminPage, BanPayload, BannedUser, ConnectionRecord, LoggedUser, UserProfile } from "@/types/admin"

import { ApiError, type PageParams } from "./admin"

function toGraphApiError(error: unknown): never {
  if (error instanceof GraphQLApiError) throw new ApiError(error.message, error.status, error.details, error.requestId)
  throw error
}
function isMockEnabled() { return env.mockOnFailure }
function fallbackPage<T>(items: T[], limit: number, offset: number): AdminPage<T> { const sliced = items.slice(offset, offset + limit); return { items: sliced, total: items.length, limit, offset, has_more: offset + sliced.length < items.length } }
function normalizeProfile(input: Record<string, unknown>): UserProfile {
  const pubkey = String(input.pubkey ?? input.public_key ?? "")
  const displayName = String(input.displayName ?? input.display_name ?? input.name ?? pubkey)
  const handleValue = input.handle ?? input.name ?? displayName
  return { pubkey, npub: String(input.npub ?? (pubkey ? buildNpubLike(pubkey) : "")), displayName, handle: String(handleValue ? `@${String(handleValue).replace(/^@/, "")}` : `@${toTitleCase(pubkey.slice(0, 8))}`), picture: input.picture ? String(input.picture) : undefined, nip05: input.nip05 ? String(input.nip05) : undefined, metadata: input.metadata ? String(input.metadata) : input.about ? String(input.about) : undefined, status: (input.status as UserProfile["status"]) ?? undefined, reason: input.reason ? String(input.reason) : undefined, related_ids: Array.isArray(input.related_ids) ? input.related_ids.map((item) => String(item)) : undefined, created_at: input.created_at ? String(input.created_at) : undefined, trustScore: typeof input.trustScore === "number" ? input.trustScore : typeof input.trust_score === "number" ? input.trust_score : undefined, relayCount: typeof input.relayCount === "number" ? input.relayCount : typeof input.relay_count === "number" ? input.relay_count : undefined, followers: typeof input.followers === "number" ? input.followers : undefined }
}

export async function getConnectionsPage(mode: "active" | "authed", params: PageParams) {
  try {
    if (mode === "active") {
      const result = await adminApolloClient.query({ query: ActiveConnectionsDocument, variables: { page: { limit: params.limit, offset: params.offset } } })
      const data = result.data
      if (!data) throw new ApiError("GraphQL query returned no data")
      return pageFromGraph(data.activeConnections, (item: any) => keysToSnake(item)) as AdminPage<ConnectionRecord>
    }
    const result = await adminApolloClient.query({ query: AuthedConnectionsDocument, variables: { page: { limit: params.limit, offset: params.offset } } })
    const data = result.data
    if (!data) throw new ApiError("GraphQL query returned no data")
    return pageFromGraph(data.authedConnections, (item: any) => keysToSnake(item)) as AdminPage<ConnectionRecord>
  } catch (error) {
    if (!isMockEnabled()) toGraphApiError(error)
    const source = mode === "active" ? mockConnections : mockConnections.filter((connection) => Boolean(connection.authed))
    return fallbackPage(source, params.limit, params.offset)
  }
}

export async function disconnectConnection(wsid: string) {
  try {
    await adminApolloClient.mutate({ mutation: DisconnectConnectionDocument, variables: { wsid, reason: "manual moderation" } })
    return { ws_id: wsid, disconnected: true }
  } catch (error) { toGraphApiError(error) }
}

export async function getLoggedUsersPage(params: PageParams) {
  try {
    const result = await adminApolloClient.query({ query: LoggedUsersDocument, variables: { page: { limit: params.limit, offset: params.offset } } })
    const data = result.data; if (!data) throw new ApiError("GraphQL query returned no data")
    return pageFromGraph(data.loggedUsers, (item: any) => ({ ...keysToSnake(item.profile), connectionCount: item.connectionCount, connection_count: item.connectionCount, lastSeenAt: item.lastSeenAt, last_seen_at: item.lastSeenAt, connectionState: item.connectionState, connection_state: item.connectionState }))
  } catch (error) {
    if (!isMockEnabled()) toGraphApiError(error)
    const users = mockUsers.slice(0, 3).map<LoggedUser>((user, index) => ({ ...user, status: "online", connectionCount: Math.max(1, 4 - index), lastSeenAt: new Date(Date.now() - index * 120000).toISOString(), connectionState: index === 0 ? "stable" : "attention" }))
    return fallbackPage(users, params.limit, params.offset)
  }
}

export async function getBannedUsersPage(query: string, params: PageParams) {
  try {
    const result = await adminApolloClient.query({ query: BannedUsersDocument, variables: { q: query || null, page: { limit: params.limit, offset: params.offset } } })
    const data = result.data; if (!data) throw new ApiError("GraphQL query returned no data")
    const page = pageFromGraph(data.bannedUsers, (item: any) => keysToSnake(item))
    return { ...page, items: page.items.map((item: any) => normalizeProfile(item) as BannedUser) }
  } catch (error) {
    if (!isMockEnabled()) toGraphApiError(error)
    const source = seedBannedUsers.filter((item) => !query || [item.displayName, item.handle, item.reason, item.npub].some((value) => value?.toLowerCase().includes(query.toLowerCase())))
    return fallbackPage(source, params.limit, params.offset)
  }
}

export async function getBanStatus(pubkey: string) {
  try {
    const result = await adminApolloClient.query({ query: UserBanStatusDocument, variables: { pubkey } })
    const data = result.data; if (!data) throw new ApiError("GraphQL query returned no data")
    return keysToSnake(data.userBanStatus)
  } catch (error) { toGraphApiError(error) }
}

export async function banUser(payload: BanPayload) {
  try {
    const result = await adminApolloClient.mutate({ mutation: BanUserDocument, variables: { pubkey: payload.pubkey, input: { reason: payload.reason ?? null, relatedIds: payload.related_ids ?? [] } } })
    const data = result.data; if (!data) throw new ApiError("GraphQL mutation returned no data")
    return keysToSnake(data.banUser)
  } catch (error) { toGraphApiError(error) }
}

export async function unbanUser(pubkey: string) {
  try { const result = await adminApolloClient.mutate({ mutation: UnbanUserDocument, variables: { pubkey } }); const data = result.data; if (!data) throw new ApiError("GraphQL mutation returned no data"); return keysToSnake(data.unbanUser) } catch (error) { toGraphApiError(error) }
}

export async function searchUsersPage(query: string, params: PageParams) {
  try {
    const result = await adminApolloClient.query({ query: SearchUsersDocument, variables: { q: query || null, page: { limit: params.limit, offset: params.offset } } })
    const data = result.data; if (!data) throw new ApiError("GraphQL query returned no data")
    const page = pageFromGraph(data.searchUsers, (item: any) => keysToSnake(item))
    return { ...page, items: page.items.map((item: any) => normalizeProfile(item)) }
  } catch (error) {
    if (!isMockEnabled()) toGraphApiError(error)
    const filtered = mockUsers.filter((user) => !query || [user.displayName, user.handle, user.npub, user.nip05, user.metadata].filter(Boolean).some((value) => value!.toLowerCase().includes(query.toLowerCase())))
    return fallbackPage(filtered, params.limit, params.offset)
  }
}

export async function getUser(pubkey: string) {
  try {
    const result = await adminApolloClient.query({ query: UserProfileDocument, variables: { pubkey } })
    const data = result.data; if (!data) throw new ApiError("GraphQL query returned no data")
    return normalizeProfile(keysToSnake(data.userProfile))
  } catch (error) {
    if (!isMockEnabled()) toGraphApiError(error)
    const fallback = mockUsers.find((user) => user.pubkey === pubkey)
    if (!fallback) throw error
    return fallback
  }
}
