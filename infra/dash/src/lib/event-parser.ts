import type { EventRecord } from "@/types/admin"

export type TagTuple = string[]

export type EventRefTag = {
  id: string
  relay?: string
}

export type ImetaResources = {
  imageURLs: string[]
  mediaURLs: string[]
  mimeTypes: string[]
  altTexts: string[]
}

export type EmbeddedRepost = {
  id: string
  kind: number
  pubkey: string
  content: string
  tags: TagTuple[]
  created_at?: number
}

export type EventMediaAsset = {
  url: string
  type: "image" | "video"
  mimeType?: string
  alt?: string
  source: "fallback" | "imeta" | "tag" | "content"
}

export type CommunityMetadata = {
  d: string
  description: string
  image: string
  moderators: string[]
}

export type CommunityApprovalSummary = {
  communityRef: string
  approvedEventId: string
  approvedKind?: number
  postAuthor: string
}

export type AddressPointer = {
  kind: number
  pubkey: string
  identifier: string
}

export function firstTagValue(tags: TagTuple[], key: string): string {
  for (const tag of tags) {
    if (tag[0] === key && tag[1]) {
      return tag[1]
    }
  }
  return ""
}

export function tagValues(tags: TagTuple[], key: string): string[] {
  const values: string[] = []
  for (const tag of tags) {
    if (tag[0] === key && tag[1]) {
      values.push(tag[1])
    }
  }
  return values
}

export function unique<T>(values: T[]): T[] {
  return Array.from(new Set(values.filter(Boolean))) as T[]
}

export function parseImetaResources(tags: TagTuple[]): ImetaResources {
  const imageURLs: string[] = []
  const mediaURLs: string[] = []
  const mimeTypes: string[] = []
  const altTexts: string[] = []

  for (const tag of tags) {
    if (tag[0] !== "imeta") {
      continue
    }
    for (const item of tag.slice(1)) {
      if (item.startsWith("image ")) {
        imageURLs.push(item.slice("image ".length).trim())
      }
      if (item.startsWith("url ")) {
        mediaURLs.push(item.slice("url ".length).trim())
      }
      if (item.startsWith("m ")) {
        mimeTypes.push(item.slice("m ".length).trim())
      }
      if (item.startsWith("alt ")) {
        altTexts.push(item.slice("alt ".length).trim())
      }
    }
  }

  return {
    imageURLs: unique(imageURLs),
    mediaURLs: unique(mediaURLs),
    mimeTypes: unique(mimeTypes),
    altTexts: unique(altTexts),
  }
}

export function parseMediaURLsFromTags(tags: TagTuple[]): string[] {
  const urls = unique([...tagValues(tags, "url"), ...tagValues(tags, "r")])
  return urls.filter((url) => /^https?:\/\//.test(url))
}

export function parseEventRefTags(tags: TagTuple[]): EventRefTag[] {
  const refs: EventRefTag[] = []
  for (const tag of tags) {
    if (tag[0] === "e" && tag[1]) {
      refs.push({ id: tag[1], relay: tag[2] })
    }
  }
  return refs
}

export function extractURLsFromText(content: string): string[] {
  const matches = content.match(/https?:\/\/[^\s"'<>()]+/g)
  return matches ?? []
}

export function isImageURL(url: string): boolean {
  return /\.(png|jpe?g|gif|webp|avif)(\?|$)/i.test(url)
}

export function isVideoURL(url: string): boolean {
  return /\.(mp4|webm|ogg|mov|m3u8)(\?|$)/i.test(url)
}

export function inferMediaType(url: string, mimeType?: string): "image" | "video" | null {
  if (mimeType) {
    if (mimeType.startsWith("image/")) return "image"
    if (mimeType.startsWith("video/")) return "video"
  }

  if (isVideoURL(url)) return "video"
  if (isImageURL(url)) return "image"
  return null
}

export function collectAltTexts(tags: TagTuple[]): string[] {
  return unique([firstTagValue(tags, "alt"), ...parseImetaResources(tags).altTexts])
}

export function parseAddressPointer(value: string): AddressPointer | null {
  const [kindValue, pubkey, ...identifierParts] = value.split(":")
  const kind = Number(kindValue)
  const identifier = identifierParts.join(":")

  if (Number.isNaN(kind) || !pubkey || !identifier) {
    return null
  }

  return {
    kind,
    pubkey,
    identifier,
  }
}

export function parseCommunityAddressTag(tags: TagTuple[]): AddressPointer | null {
  const communityTag = tags.find((tag) => (tag[0] === "a" || tag[0] === "A") && tag[1]?.startsWith("34550:"))
  if (!communityTag?.[1]) {
    return null
  }

  return parseAddressPointer(communityTag[1])
}

export function communityDisplayNameFromAddress(value: string): string {
  return parseAddressPointer(value)?.identifier || value
}

export function communityNameFromTags(tags: TagTuple[]): string {
  const name = firstTagValue(tags, "name")
  if (name) {
    return name
  }

  const dTag = firstTagValue(tags, "d")
  if (dTag) {
    return dTag
  }

  const communityAddress = parseCommunityAddressTag(tags)
  return communityAddress?.identifier || ""
}

export function collectMediaAssets(tags: TagTuple[], content: string, fallbackImages: string[] = []): EventMediaAsset[] {
  const assets = new Map<string, EventMediaAsset>()
  const imeta = parseImetaResources(tags)

  for (const img of fallbackImages) {
    assets.set(img, { url: img, type: "image", source: "fallback" })
  }

  for (const tag of tags) {
    if (tag[0] !== "imeta") {
      continue
    }

    let url = ""
    let mimeType = ""
    let alt = ""

    for (const item of tag.slice(1)) {
      if (item.startsWith("url ")) url = item.slice(4).trim()
      if (item.startsWith("m ")) mimeType = item.slice(2).trim()
      if (item.startsWith("alt ")) alt = item.slice(4).trim()
      if (!url && item.startsWith("image ")) url = item.slice(6).trim()
    }

    if (!url) {
      continue
    }

    const type = inferMediaType(url, mimeType)
    if (!type) {
      continue
    }

    assets.set(url, {
      url,
      type,
      mimeType: mimeType || undefined,
      alt: alt || undefined,
      source: "imeta",
    })
  }

  for (const url of [...imeta.imageURLs, ...parseMediaURLsFromTags(tags), ...extractURLsFromText(content || "")]) {
    const type = inferMediaType(url)
    if (!type) {
      continue
    }

    if (!assets.has(url)) {
      assets.set(url, {
        url,
        type,
        source: extractURLsFromText(content || "").includes(url) ? "content" : "tag",
      })
    }
  }

  return Array.from(assets.values())
}

export function collectMediaForEvent(event: EventRecord, fallbackImages: string[]): { images: string[]; videos: string[] } {
  const images = new Set<string>()
  const videos = new Set<string>()

  for (const asset of collectMediaAssets(event.tags as TagTuple[], event.content || "", fallbackImages)) {
    if (asset.type === "video") {
      videos.add(asset.url)
      continue
    }
    images.add(asset.url)
  }

  return {
    images: Array.from(images),
    videos: Array.from(videos),
  }
}

export function pickVideoURL(urls: string[]): string {
  const preferred = urls.find((url) => /\.(mp4|webm|ogg|mov)(\?|$)/i.test(url) || /video/i.test(url))
  return preferred ?? urls[0] ?? ""
}

export function parseEmbeddedEvent(content: string): EmbeddedRepost | null {
  const trimmed = content.trim()
  if (!trimmed.startsWith("{")) {
    return null
  }

  try {
    const parsed = JSON.parse(trimmed) as {
      id?: unknown
      kind?: unknown
      pubkey?: unknown
      content?: unknown
      tags?: unknown
      created_at?: unknown
    }
    if (typeof parsed.id !== "string") {
      return null
    }

    return {
      id: parsed.id,
      kind: typeof parsed.kind === "number" ? parsed.kind : -1,
      pubkey: typeof parsed.pubkey === "string" ? parsed.pubkey : "",
      content: typeof parsed.content === "string" ? parsed.content : "",
      tags: Array.isArray(parsed.tags) ? parsed.tags.filter((tag): tag is string[] => Array.isArray(tag) && tag.every((item) => typeof item === "string")) : [],
      created_at: typeof parsed.created_at === "number" ? parsed.created_at : undefined,
    }
  } catch {
    return null
  }
}

export function parseEmbeddedRepost(content: string, kind: number): EmbeddedRepost | null {
  if (kind !== 6 && kind !== 16) {
    return null
  }

  return parseEmbeddedEvent(content)
}

export function extractThreadRefs(tags: TagTuple[]): { root: string; reply: string } {
  const rootTag = tags.find((tag) => tag[0] === "e" && tag[3] === "root")
  const replyTag = tags.find((tag) => tag[0] === "e" && tag[3] === "reply")
  return {
    root: rootTag?.[1] ?? "",
    reply: replyTag?.[1] ?? "",
  }
}

export function parseCommunityMetadata(tags: TagTuple[]): CommunityMetadata {
  const moderators: string[] = []
  for (const tag of tags) {
    if (tag[0] === "p" && tag[1] && tag[3] === "moderator") {
      moderators.push(tag[1])
    }
  }

  return {
    d: firstTagValue(tags, "d"),
    description: firstTagValue(tags, "description"),
    image: firstTagValue(tags, "image"),
    moderators: unique(moderators),
  }
}

export function parseCommunityApproval(tags: TagTuple[]): CommunityApprovalSummary {
  const aTag = tags.find((tag) => tag[0] === "a" && tag[1]?.startsWith("34550:"))
  const eTag = tags.find((tag) => tag[0] === "e")
  const kTag = tags.find((tag) => tag[0] === "k")
  const pTag = tags.find((tag) => tag[0] === "p")

  return {
    communityRef: aTag?.[1] ?? "",
    approvedEventId: eTag?.[1] ?? "",
    approvedKind: kTag?.[1] ? Number(kTag[1]) : undefined,
    postAuthor: pTag?.[1] ?? "",
  }
}

export function parseDMRelays(tags: TagTuple[]): string[] {
  return tags.filter((tag) => tag[0] === "relay" && tag[1]).map((tag) => tag[1] as string)
}

export const COMMUNITY_APPROVAL_KIND = 4550
export const DM_RELAY_KIND = 10050
export const VIDEO_KINDS = [21, 22, 34235]
export const IMAGE_KIND = 20
export const REPOST_KINDS = [6, 16]
export const REACTION_KIND = 7
export const LIST_KIND = 30003

export const commonRelays = [
  "wss://relay.damus.io",
  "wss://relay.primal.net",
  "wss://nos.lol",
  "wss://relay.nostr.band",
  "wss://nostr.mom",
]
