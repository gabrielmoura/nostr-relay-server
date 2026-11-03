import { describe, expect, it } from "vitest"

import { createAdminCache } from "./client"
import { BlossomObjectsDocument } from "./generated/operations"

const item = (hash: string) => ({
  __typename: "AdminBlossomObject" as const, hash, mimeType: "image/png", size: 1,
  createdAt: "2024-01-01T00:00:00Z", thumbnailURL: null, reviewState: "ready", exifStatus: "pending",
})
const result = (hashes: string[], offset: number, hasMore: boolean) => ({
  __typename: "Query" as const,
  blossomObjects: {
    __typename: "AdminBlossomObjectPage" as const,
    pageInfo: { __typename: "PageInfo" as const, total: 4, limit: 2, offset, hasMore },
    items: hashes.map(item),
  },
})
const variables = (q: string, offset: number) => ({ filter: { q }, page: { limit: 2, offset } })

describe("Apollo Blossom object cache", () => {
  it("merges offset pages, deduplicates overlaps and tracks the next server offset", () => {
    const cache = createAdminCache()
    cache.writeQuery({ query: BlossomObjectsDocument, variables: variables("image", 0), data: result(["a", "b"], 0, true) })
    cache.writeQuery({ query: BlossomObjectsDocument, variables: variables("image", 2), data: result(["b", "c"], 2, true) })
    const page = cache.readQuery({ query: BlossomObjectsDocument, variables: variables("image", 0) })!.blossomObjects
    expect(page.items.map((object) => object.hash)).toEqual(["a", "b", "c"])
    expect(page.pageInfo.offset + page.items.length).toBe(4)
    cache.writeQuery({ query: BlossomObjectsDocument, variables: variables("image", 4), data: result(["d"], 4, false) })
    expect(cache.readQuery({ query: BlossomObjectsDocument, variables: variables("image", 0) })!.blossomObjects.items.map((object) => object.hash)).toEqual(["a", "b", "c", "d"])
  })

  it("separates filters, replaces the first page on refresh and evicts all variants", () => {
    const cache = createAdminCache()
    cache.writeQuery({ query: BlossomObjectsDocument, variables: variables("a", 0), data: result(["a"], 0, false) })
    cache.writeQuery({ query: BlossomObjectsDocument, variables: variables("b", 0), data: result(["b"], 0, false) })
    cache.writeQuery({ query: BlossomObjectsDocument, variables: variables("a", 0), data: result(["c"], 0, false) })
    expect(cache.readQuery({ query: BlossomObjectsDocument, variables: variables("a", 0) })!.blossomObjects.items.map((object) => object.hash)).toEqual(["c"])
    expect(cache.readQuery({ query: BlossomObjectsDocument, variables: variables("b", 0) })!.blossomObjects.items.map((object) => object.hash)).toEqual(["b"])
    cache.evict({ id: "ROOT_QUERY", fieldName: "blossomObjects" })
    expect(cache.readQuery({ query: BlossomObjectsDocument, variables: variables("a", 0) })).toBeNull()
    expect(cache.readQuery({ query: BlossomObjectsDocument, variables: variables("b", 0) })).toBeNull()
  })
})
