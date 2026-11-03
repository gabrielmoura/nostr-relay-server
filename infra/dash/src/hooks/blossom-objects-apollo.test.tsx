// @vitest-environment jsdom

import { act, cleanup, renderHook, waitFor } from "@testing-library/react"
import { afterEach, expect, it, vi } from "vitest"

import { adminApolloClient } from "@/graphql/client"
import { useBlossomBulkReviewMutation, useInfiniteBlossomObjects } from "./use-admin-data"

const item = (hash: string) => ({ __typename: "AdminBlossomObject", hash, mimeType: "image/png", size: 1, createdAt: "2024-01-01T00:00:00Z", thumbnailURL: null, reviewState: "ready", exifStatus: "pending" })

afterEach(async () => {
  cleanup()
  await adminApolloClient.clearStore()
  vi.unstubAllGlobals()
})

it("loads more through Apollo and refreshes the active list after review", async () => {
  const requests: Array<{ operationName: string; variables: { page?: { offset: number } } }> = []
  const fetchMock = vi.fn(async (_input: unknown, init: RequestInit) => {
    const request = JSON.parse(String(init.body))
    requests.push(request)
    const offset = request.variables?.page?.offset ?? 0
    const data = request.operationName === "BlossomObjects"
      ? { blossomObjects: { __typename: "AdminBlossomObjectPage", pageInfo: { __typename: "PageInfo", total: 3, limit: 2, offset, hasMore: offset === 0 }, items: offset === 0 ? [item("a"), item("b")] : [item("c")] } }
      : { reviewBlossomObjects: { __typename: "AdminAck", ok: true } }
    return new Response(JSON.stringify({ data }), { status: 200, headers: { "content-type": "application/json" } })
  })
  vi.stubGlobal("fetch", fetchMock)

  const hook = renderHook(() => ({ list: useInfiniteBlossomObjects({ q: "image" }), review: useBlossomBulkReviewMutation() }))
  await waitFor(() => expect(hook.result.current.list.data?.pages[0]?.items).toHaveLength(2))
  await act(async () => { await hook.result.current.list.fetchNextPage() })
  await waitFor(() => expect(hook.result.current.list.data?.pages[0]?.items).toHaveLength(3))
  expect(requests.filter((request) => request.operationName === "BlossomObjects").map((request) => request.variables.page?.offset)).toEqual([0, 2])

  await act(async () => { await hook.result.current.review.mutateAsync({ hashes: ["a"], action: "approve" }) })
  await waitFor(() => expect(requests.filter((request) => request.operationName === "BlossomObjects").length).toBeGreaterThan(2))
  await waitFor(() => expect(hook.result.current.list.data?.pages[0]?.items).toHaveLength(2))
})

it("keeps loaded objects and exposes a pagination error", async () => {
  vi.stubGlobal("fetch", vi.fn(async (_input: unknown, init: RequestInit) => {
    const request = JSON.parse(String(init.body))
    if (request.variables.page.offset > 0) throw new Error("page unavailable")
    return new Response(JSON.stringify({ data: { blossomObjects: {
      __typename: "AdminBlossomObjectPage", pageInfo: { __typename: "PageInfo", total: 2, limit: 1, offset: 0, hasMore: true }, items: [item("a")],
    } } }), { status: 200, headers: { "content-type": "application/json" } })
  }))
  const hook = renderHook(() => useInfiniteBlossomObjects({}))
  await waitFor(() => expect(hook.result.current.data?.pages[0]?.items).toHaveLength(1))
  await act(async () => { await hook.result.current.fetchNextPage() })
  expect(hook.result.current.isError).toBe(true)
  expect(hook.result.current.data?.pages[0]?.items).toHaveLength(1)
})

it("shows cached objects immediately on revisit while refreshing", async () => {
  let resolveRefresh: ((response: Response) => void) | undefined
  const response = () => new Response(JSON.stringify({ data: { blossomObjects: {
    __typename: "AdminBlossomObjectPage", pageInfo: { __typename: "PageInfo", total: 1, limit: 50, offset: 0, hasMore: false }, items: [item("cached")],
  } } }), { status: 200, headers: { "content-type": "application/json" } })
  const fetchMock = vi.fn()
    .mockResolvedValueOnce(response())
    .mockImplementationOnce(() => new Promise<Response>((resolve) => { resolveRefresh = resolve }))
  vi.stubGlobal("fetch", fetchMock)
  const first = renderHook(() => useInfiniteBlossomObjects({ q: "cached" }))
  await waitFor(() => expect(first.result.current.data?.pages[0]?.items).toHaveLength(1))
  first.unmount()
  const second = renderHook(() => useInfiniteBlossomObjects({ q: "cached" }))
  expect(second.result.current.data?.pages[0]?.items).toHaveLength(1)
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
  await act(async () => { resolveRefresh?.(response()) })
})
