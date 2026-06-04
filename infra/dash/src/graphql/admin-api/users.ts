import {
  ActiveConnectionsDocument,
  AdminOverviewDocument,
  AdminStreamStatusDocument,
  AuthedConnectionsDocument,
  BanUserDocument,
  BannedUsersDocument,
  DisconnectConnectionDocument,
  LoggedUsersDocument,
  SearchUsersDocument,
  UnbanUserDocument,
  UserBanStatusDocument,
  UserProfileDocument,
} from "@/graphql/generated/operations"

import { emptyToNull, graphMutation, graphQuery, keysToSnake, pageFromGraph, pageVars, type RequestContext } from "./core"

export async function handleUsersRequest<T>(ctx: RequestContext): Promise<T | undefined> {
  const { method, pathname, search, body } = ctx

  switch (true) {
    case method === "GET" && pathname === "/overview":
      return graphQuery(AdminOverviewDocument, undefined, (data) => keysToSnake(data.adminOverview)) as Promise<T>
    case method === "GET" && pathname === "/stream/status":
      return graphQuery(AdminStreamStatusDocument, undefined, (data) => keysToSnake(data.adminStreamStatus)) as Promise<T>
    case method === "GET" && pathname === "/connections/active":
      return graphQuery(ActiveConnectionsDocument, { page: pageVars(search) }, (data) => pageFromGraph(data.activeConnections, (item) => keysToSnake(item))) as Promise<T>
    case method === "GET" && pathname === "/connections/authed":
      return graphQuery(AuthedConnectionsDocument, { page: pageVars(search) }, (data) => pageFromGraph(data.authedConnections, (item) => keysToSnake(item))) as Promise<T>
    case method === "POST" && pathname.match(/^\/connections\/[^/]+\/disconnect$/) !== null: {
      const wsid = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphMutation(DisconnectConnectionDocument, { wsid, reason: body?.reason ?? null }, () => ({ ws_id: wsid, disconnected: true })) as Promise<T>
    }
    case method === "GET" && pathname === "/users/logged":
      return graphQuery(LoggedUsersDocument, { page: pageVars(search) }, (data) => pageFromGraph(data.loggedUsers, (item) => ({ ...keysToSnake(item.profile), connectionCount: item.connectionCount, connection_count: item.connectionCount, lastSeenAt: item.lastSeenAt, last_seen_at: item.lastSeenAt, connectionState: item.connectionState, connection_state: item.connectionState }))) as Promise<T>
    case method === "GET" && pathname === "/users/banned":
      return graphQuery(BannedUsersDocument, { q: emptyToNull(search.get("q")), page: pageVars(search) }, (data) => pageFromGraph(data.bannedUsers, (item) => keysToSnake(item))) as Promise<T>
    case method === "GET" && pathname === "/users/search":
      return graphQuery(SearchUsersDocument, { q: emptyToNull(search.get("q")), page: pageVars(search) }, (data) => pageFromGraph(data.searchUsers, (item) => keysToSnake(item))) as Promise<T>
    case method === "GET" && pathname.match(/^\/users\/[^/]+\/profile$/) !== null: {
      const pubkey = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphQuery(UserProfileDocument, { pubkey }, (data) => keysToSnake(data.userProfile)) as Promise<T>
    }
    case method === "GET" && pathname.match(/^\/users\/[^/]+\/ban$/) !== null: {
      const pubkey = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphQuery(UserBanStatusDocument, { pubkey }, (data) => keysToSnake(data.userBanStatus)) as Promise<T>
    }
    case method === "POST" && pathname.match(/^\/users\/[^/]+\/ban$/) !== null: {
      const pubkey = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphMutation(BanUserDocument, { pubkey, input: { reason: body?.reason ?? null, relatedIds: body?.related_ids ?? [] } }, (data) => keysToSnake(data.banUser)) as Promise<T>
    }
    case method === "DELETE" && pathname.match(/^\/users\/[^/]+\/ban$/) !== null: {
      const pubkey = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphMutation(UnbanUserDocument, { pubkey }, (data) => keysToSnake(data.unbanUser)) as Promise<T>
    }
    default:
      return undefined
  }
}
