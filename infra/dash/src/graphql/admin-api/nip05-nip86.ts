import {
  CreateNip86AllowedPubkeyDocument,
  CreateNip86BannedEventDocument,
  CreateNip86BlockedIpDocument,
  DeleteNip05Document,
  DeleteNip86AllowedPubkeyDocument,
  DeleteNip86BannedEventDocument,
  DeleteNip86BlockedIpDocument,
  Nip05IdentitiesDocument,
  Nip86AllowedPubkeysDocument,
  Nip86BannedEventsDocument,
  Nip86BlockedIpsDocument,
  Nip86RelayMetadataDocument,
  UpdateNip86RelayMetadataDocument,
  UpsertNip05Document,
  UserNip05Document,
} from "@/graphql/generated/operations"

import { emptyToNull, graphMutation, graphQuery, keysToSnake, pageFromGraph, pageVars, type RequestContext } from "./core"

export async function handleIdentityRequest<T>(ctx: RequestContext): Promise<T | undefined> {
  const { method, pathname, search, body } = ctx

  switch (true) {
    case method === "GET" && pathname === "/nip05":
      return graphQuery(Nip05IdentitiesDocument, { q: emptyToNull(search.get("q")), page: pageVars(search) }, (data) => pageFromGraph(data.nip05Identities, (item) => keysToSnake(item))) as Promise<T>
    case method === "POST" && pathname === "/nip05":
      return graphMutation(UpsertNip05Document, { input: body }, (data) => keysToSnake(data.upsertNip05)) as Promise<T>
    case method === "DELETE" && pathname.match(/^\/nip05\//) !== null: {
      const name = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphMutation(DeleteNip05Document, { name }, () => ({ name, deleted: true })) as Promise<T>
    }
    case method === "GET" && pathname.match(/^\/users\/[^/]+\/nip05$/) !== null: {
      const pubkey = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphQuery(UserNip05Document, { pubkey }, (data) => ({ pubkey: data.userNip05.pubkey, exists: data.userNip05.exists, name: data.userNip05.identity?.name, display_name: data.userNip05.identity?.displayName, picture: data.userNip05.identity?.picture, relay_hints: data.userNip05.identity?.relayHints, created_at: data.userNip05.identity?.createdAt, updated_at: data.userNip05.identity?.updatedAt })) as Promise<T>
    }
    case method === "GET" && pathname === "/nip86/allowed-pubkeys":
      return graphQuery(Nip86AllowedPubkeysDocument, { q: emptyToNull(search.get("q")), page: pageVars(search) }, (data) => pageFromGraph(data.nip86AllowedPubkeys, (item) => keysToSnake(item))) as Promise<T>
    case method === "POST" && pathname.match(/^\/nip86\/allowed-pubkeys\//) !== null: {
      const pubkey = decodeURIComponent(pathname.split("/")[3] ?? "")
      return graphMutation(CreateNip86AllowedPubkeyDocument, { pubkey, input: { reason: body?.reason ?? null } }, () => ({ ok: true })) as Promise<T>
    }
    case method === "DELETE" && pathname.match(/^\/nip86\/allowed-pubkeys\//) !== null: {
      const pubkey = decodeURIComponent(pathname.split("/")[3] ?? "")
      return graphMutation(DeleteNip86AllowedPubkeyDocument, { pubkey }, () => ({ ok: true })) as Promise<T>
    }
    case method === "GET" && pathname === "/nip86/blocked-ips":
      return graphQuery(Nip86BlockedIpsDocument, { q: emptyToNull(search.get("q")), page: pageVars(search) }, (data) => pageFromGraph(data.nip86BlockedIps, (item) => keysToSnake(item))) as Promise<T>
    case method === "POST" && pathname.match(/^\/nip86\/blocked-ips\//) !== null: {
      const ip = decodeURIComponent(pathname.split("/")[3] ?? "")
      return graphMutation(CreateNip86BlockedIpDocument, { ip, input: { reason: body?.reason ?? null } }, () => ({ ok: true })) as Promise<T>
    }
    case method === "DELETE" && pathname.match(/^\/nip86\/blocked-ips\//) !== null: {
      const ip = decodeURIComponent(pathname.split("/")[3] ?? "")
      return graphMutation(DeleteNip86BlockedIpDocument, { ip }, () => ({ ok: true })) as Promise<T>
    }
    case method === "GET" && pathname === "/nip86/banned-events":
      return graphQuery(Nip86BannedEventsDocument, { q: emptyToNull(search.get("q")), page: pageVars(search) }, (data) => pageFromGraph(data.nip86BannedEvents, (item) => keysToSnake(item))) as Promise<T>
    case method === "POST" && pathname.match(/^\/nip86\/banned-events\//) !== null: {
      const eventId = decodeURIComponent(pathname.split("/")[3] ?? "")
      return graphMutation(CreateNip86BannedEventDocument, { eventId, input: { reason: body?.reason ?? null } }, () => ({ ok: true })) as Promise<T>
    }
    case method === "DELETE" && pathname.match(/^\/nip86\/banned-events\//) !== null: {
      const eventId = decodeURIComponent(pathname.split("/")[3] ?? "")
      return graphMutation(DeleteNip86BannedEventDocument, { eventId }, () => ({ ok: true })) as Promise<T>
    }
    case method === "GET" && pathname === "/nip86/relay-metadata":
      return graphQuery(Nip86RelayMetadataDocument, undefined, (data) => keysToSnake(data.nip86RelayMetadata)) as Promise<T>
    case method === "POST" && pathname === "/nip86/relay-metadata":
      return graphMutation(UpdateNip86RelayMetadataDocument, { input: body }, (data) => ({ updated: true, ...keysToSnake(data.updateNip86RelayMetadata) })) as Promise<T>
    default:
      return undefined
  }
}
