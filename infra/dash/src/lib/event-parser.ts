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
}

export type CommunityMetadata = {
  d: string
  description: string
  image: string
  moderators: string[]
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

export function collectMediaForEvent(event: EventRecord, fallbackImages: string[]): { images: string[]; videos: string[] } {
  const images = new Set<string>()
  const videos = new Set<string>()

  for (const img of fallbackImages) {
    images.add(img)
  }

  const imeta = parseImetaResources(event.tags as TagTuple[])
  for (const img of imeta.imageURLs) {
    images.add(img)
  }

  const candidateURLs = [
    ...imeta.mediaURLs,
    ...parseMediaURLsFromTags(event.tags as TagTuple[]),
    ...extractURLsFromText(event.content || ""),
  ]

  for (const url of candidateURLs) {
    if (isVideoURL(url)) {
      videos.add(url)
      continue
    }
    if (isImageURL(url)) {
      images.add(url)
    }
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

export function parseEmbeddedRepost(content: string, kind: number): EmbeddedRepost | null {
  if (kind !== 6 && kind !== 16) {
    return null
  }

  const trimmed = content.trim()
  if (!trimmed.startsWith("{")) {
    return null
  }

  try {
    const parsed = JSON.parse(trimmed) as { id?: unknown; kind?: unknown; pubkey?: unknown; content?: unknown }
    if (typeof parsed.id !== "string") {
      return null
    }

    return {
      id: parsed.id,
      kind: typeof parsed.kind === "number" ? parsed.kind : -1,
      pubkey: typeof parsed.pubkey === "string" ? parsed.pubkey : "",
      content: typeof parsed.content === "string" ? parsed.content : "",
    }
  } catch {
    return null
  }
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
