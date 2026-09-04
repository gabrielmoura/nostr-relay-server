import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from "react"

type QueryKey = unknown[]

export type CompatQueryClient = {
  invalidateQueries: (options: { queryKey: QueryKey }) => Promise<void>
}

type CompatQueryState<T> = {
  data?: T
}

export type CompatQueryOptions<T> = {
  queryKey: QueryKey
  queryFn: () => Promise<T>
  enabled?: boolean
  refetchInterval?: number | false | ((query: { state: CompatQueryState<T> }) => number | false | undefined)
  retry?: boolean | number
}

export type CompatInfiniteQueryOptions<T> = {
  queryKey: QueryKey
  queryFn: (context: { pageParam: number }) => Promise<T>
  initialPageParam: number
  getNextPageParam: (lastPage: T, allPages: T[]) => number | undefined
  enabled?: boolean
}

export type CompatMutationOptions<TData, TVars> = {
  mutationFn: (variables: TVars) => Promise<TData>
  onSuccess?: (data: TData, variables: TVars) => Promise<void> | void
}

type CompatInfiniteData<T> = {
  pages: T[]
}

export type CompatMutationResult<TData, TVars> = {
  mutateAsync: (variables: TVars) => Promise<TData>
  mutate: (variables: TVars) => void
  isPending: boolean
  isSuccess: boolean
  data?: TData
  error: unknown
  reset: () => void
}

export type CompatQueryResult<T> = {
  data?: T
  isLoading: boolean
  isError: boolean
  error: unknown
  isFetching: boolean
  refetch: () => Promise<T | undefined>
}

export type CompatInfiniteQueryResult<T> = CompatQueryResult<CompatInfiniteData<T>> & {
  fetchNextPage: () => Promise<void>
  hasNextPage: boolean
  isFetchingNextPage: boolean
}

type SuspenseResource<T> = {
  status: "pending" | "success" | "error"
  promise?: Promise<void>
  data?: T
  error?: unknown
}

const querySubscribers = new Set<{
  key: QueryKey
  callback: () => void
}>()

const suspenseCache = new Map<string, SuspenseResource<unknown>>()

function serializeKey(key: QueryKey): string {
  return JSON.stringify(key)
}

function keysMatchPrefix(candidate: QueryKey, prefix: QueryKey): boolean {
  if (prefix.length > candidate.length) {
    return false
  }

  return prefix.every((value, index) => JSON.stringify(value) === JSON.stringify(candidate[index]))
}

type CacheEntry<T> = {
  data?: T
  error?: unknown
  isLoading: boolean
  isFetching: boolean
  promise?: Promise<T>
  listeners: Set<() => void>
}

type InfiniteCacheEntry<T> = {
  pages: T[]
  error?: unknown
  isLoading: boolean
  isFetching: boolean
  isFetchingNextPage: boolean
  promise?: Promise<T>
  listeners: Set<() => void>
}

const queryCache = new Map<string, CacheEntry<any>>()
const infiniteQueryCache = new Map<string, InfiniteCacheEntry<any>>()

function getCacheEntry<T>(keyStr: string, enabled = true): CacheEntry<T> {
  let entry = queryCache.get(keyStr)
  if (!entry) {
    entry = {
      isLoading: enabled,
      isFetching: false,
      listeners: new Set(),
    }
    queryCache.set(keyStr, entry)
  }
  return entry
}

function getInfiniteCacheEntry<T>(keyStr: string, enabled = true): InfiniteCacheEntry<T> {
  let entry = infiniteQueryCache.get(keyStr)
  if (!entry) {
    entry = {
      pages: [],
      isLoading: enabled,
      isFetching: false,
      isFetchingNextPage: false,
      listeners: new Set(),
    }
    infiniteQueryCache.set(keyStr, entry)
  }
  return entry
}

function notifyInvalidation(prefix: QueryKey) {
  for (const entry of querySubscribers) {
    if (keysMatchPrefix(entry.key, prefix)) {
      entry.callback()
    }
  }

  for (const key of queryCache.keys()) {
    const parsed = JSON.parse(key) as QueryKey
    if (keysMatchPrefix(parsed, prefix)) {
      queryCache.delete(key)
    }
  }

  for (const key of infiniteQueryCache.keys()) {
    const parsed = JSON.parse(key) as QueryKey
    if (keysMatchPrefix(parsed, prefix)) {
      infiniteQueryCache.delete(key)
    }
  }

  for (const key of suspenseCache.keys()) {
    const parsed = JSON.parse(key) as QueryKey
    if (keysMatchPrefix(parsed, prefix)) {
      suspenseCache.delete(key)
    }
  }
}

export function useQueryClient(): CompatQueryClient {
  return useMemo(
    () => ({
      invalidateQueries: async ({ queryKey }) => {
        notifyInvalidation(queryKey)
      },
    }),
    [],
  )
}

export function useQuery<T>({ queryKey, queryFn, enabled = true, refetchInterval }: CompatQueryOptions<T>): CompatQueryResult<T> {
  const serializedKey = serializeKey(queryKey)
  const [, forceUpdate] = useReducer((x) => x + 1, 0)

  const entry = getCacheEntry<T>(serializedKey, enabled)

  const queryFnRef = useRef(queryFn)
  useEffect(() => {
    queryFnRef.current = queryFn
  }, [queryFn])

  const refetchIntervalRef = useRef(refetchInterval)
  useEffect(() => {
    refetchIntervalRef.current = refetchInterval
  }, [refetchInterval])

  useEffect(() => {
    const listener = () => forceUpdate()
    entry.listeners.add(listener)
    return () => {
      entry.listeners.delete(listener)
    }
  }, [entry])

  const execute = useCallback(async () => {
    if (!enabled) return undefined

    const cacheEntry = getCacheEntry<T>(serializedKey, enabled)
    if (cacheEntry.promise) {
      return cacheEntry.promise
    }

    cacheEntry.isFetching = true
    cacheEntry.error = null
    if (cacheEntry.data === undefined) {
      cacheEntry.isLoading = true
    }
    cacheEntry.listeners.forEach((l) => l())

    const promise = queryFnRef.current()
    cacheEntry.promise = promise

    try {
      const data = await promise
      cacheEntry.data = data
      cacheEntry.isLoading = false
      cacheEntry.isFetching = false
      cacheEntry.promise = undefined
      cacheEntry.listeners.forEach((l) => l())
      return data
    } catch (err) {
      cacheEntry.error = err
      cacheEntry.isLoading = false
      cacheEntry.isFetching = false
      cacheEntry.promise = undefined
      cacheEntry.listeners.forEach((l) => l())
      throw err
    }
  }, [enabled, serializedKey])

  useEffect(() => {
    if (!enabled) return
    const cacheEntry = getCacheEntry<T>(serializedKey, enabled)
    if (cacheEntry.data === undefined && !cacheEntry.isFetching) {
      void execute().catch(() => undefined)
    }
  }, [enabled, execute, serializedKey])

  useEffect(() => {
    if (!enabled) return
    const subEntry = {
      key: queryKey,
      callback: () => {
        const cacheEntry = getCacheEntry<T>(serializedKey, enabled)
        cacheEntry.promise = undefined
        void execute().catch(() => undefined)
      },
    }
    querySubscribers.add(subEntry)
    return () => {
      querySubscribers.delete(subEntry)
    }
  }, [enabled, execute, queryKey, serializedKey])

  useEffect(() => {
    if (!enabled) return
    const intervalConfig = refetchIntervalRef.current
    const nextInterval = typeof intervalConfig === "function" ? intervalConfig({ state: { data: entry.data } }) : intervalConfig
    if (!nextInterval || nextInterval <= 0) return

    const timer = window.setInterval(() => {
      const cacheEntry = getCacheEntry<T>(serializedKey, enabled)
      cacheEntry.promise = undefined
      void execute().catch(() => undefined)
    }, nextInterval)

    return () => window.clearInterval(timer)
  }, [entry.data, enabled, execute, serializedKey])

  return {
    data: entry.data,
    isLoading: entry.data === undefined && entry.isLoading,
    isError: entry.error != null,
    error: entry.error,
    isFetching: entry.isFetching,
    refetch: async () => {
      const cacheEntry = getCacheEntry<T>(serializedKey, enabled)
      cacheEntry.promise = undefined
      return execute()
    },
  }
}

export function useInfiniteQuery<T>({ queryKey, queryFn, initialPageParam, getNextPageParam, enabled = true }: CompatInfiniteQueryOptions<T>): CompatInfiniteQueryResult<T> {
  const serializedKey = serializeKey(queryKey)
  const [, forceUpdate] = useReducer((x) => x + 1, 0)

  const entry = getInfiniteCacheEntry<T>(serializedKey, enabled)

  const queryFnRef = useRef(queryFn)
  useEffect(() => {
    queryFnRef.current = queryFn
  }, [queryFn])

  const getNextPageParamRef = useRef(getNextPageParam)
  useEffect(() => {
    getNextPageParamRef.current = getNextPageParam
  }, [getNextPageParam])

  useEffect(() => {
    const listener = () => forceUpdate()
    entry.listeners.add(listener)
    return () => {
      entry.listeners.delete(listener)
    }
  }, [entry])

  const runFirstPage = useCallback(async () => {
    if (!enabled) return undefined
    const cacheEntry = getInfiniteCacheEntry<T>(serializedKey, enabled)
    if (cacheEntry.promise) return cacheEntry.promise

    cacheEntry.isFetching = true
    cacheEntry.error = null
    if (cacheEntry.pages.length === 0) {
      cacheEntry.isLoading = true
    }
    cacheEntry.listeners.forEach((l) => l())

    const promise = queryFnRef.current({ pageParam: initialPageParam })
    cacheEntry.promise = promise

    try {
      const first = await promise
      cacheEntry.pages = [first]
      cacheEntry.isLoading = false
      cacheEntry.isFetching = false
      cacheEntry.promise = undefined
      cacheEntry.listeners.forEach((l) => l())
      return first
    } catch (err) {
      cacheEntry.error = err
      cacheEntry.isLoading = false
      cacheEntry.isFetching = false
      cacheEntry.promise = undefined
      cacheEntry.listeners.forEach((l) => l())
      throw err
    }
  }, [enabled, initialPageParam, serializedKey])

  const fetchNextPage = useCallback(async () => {
    const cacheEntry = getInfiniteCacheEntry<T>(serializedKey, enabled)
    const currentPages = cacheEntry.pages
    const lastPage = currentPages[currentPages.length - 1]
    if (!enabled || !lastPage) return
    const next = getNextPageParamRef.current(lastPage, currentPages)
    if (next === undefined) return

    cacheEntry.isFetchingNextPage = true
    cacheEntry.error = null
    cacheEntry.listeners.forEach((l) => l())

    try {
      const nextPage = await queryFnRef.current({ pageParam: next })
      cacheEntry.pages = [...cacheEntry.pages, nextPage]
      cacheEntry.isFetchingNextPage = false
      cacheEntry.listeners.forEach((l) => l())
    } catch (err) {
      cacheEntry.error = err
      cacheEntry.isFetchingNextPage = false
      cacheEntry.listeners.forEach((l) => l())
      throw err
    }
  }, [enabled, serializedKey])

  useEffect(() => {
    if (!enabled) return
    const cacheEntry = getInfiniteCacheEntry<T>(serializedKey, enabled)
    if (cacheEntry.pages.length === 0 && !cacheEntry.isFetching) {
      void runFirstPage()
    }
  }, [enabled, runFirstPage, serializedKey])

  useEffect(() => {
    if (!enabled) return
    const subEntry = {
      key: queryKey,
      callback: () => {
        const cacheEntry = getInfiniteCacheEntry<T>(serializedKey, enabled)
        cacheEntry.promise = undefined
        void runFirstPage()
      },
    }
    querySubscribers.add(subEntry)
    return () => {
      querySubscribers.delete(subEntry)
    }
  }, [enabled, runFirstPage, queryKey, serializedKey])

  const hasNextPage = (() => {
    const lastPage = entry.pages[entry.pages.length - 1]
    return Boolean(lastPage) && getNextPageParamRef.current(lastPage as T, entry.pages) !== undefined
  })()

  return {
    data: entry.pages.length > 0 ? { pages: entry.pages } : undefined,
    isLoading: entry.pages.length === 0 && entry.isLoading,
    isError: entry.error != null,
    error: entry.error,
    isFetching: entry.isFetching,
    refetch: async () => {
      const cacheEntry = getInfiniteCacheEntry<T>(serializedKey, enabled)
      cacheEntry.promise = undefined
      const firstPage = await runFirstPage()
      return firstPage !== undefined ? { pages: [firstPage] } : undefined
    },
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage: entry.isFetchingNextPage,
  }
}

export function useMutation<TData, TVars>({ mutationFn, onSuccess }: CompatMutationOptions<TData, TVars>): CompatMutationResult<TData, TVars> {
  const [isPending, setIsPending] = useState(false)
  const [isSuccess, setIsSuccess] = useState(false)
  const [data, setData] = useState<TData | undefined>(undefined)
  const [error, setError] = useState<unknown>(null)

  const mutateAsync = useCallback(async (variables: TVars) => {
    setIsPending(true)
    setIsSuccess(false)
    setError(null)
    try {
      const result = await mutationFn(variables)
      await onSuccess?.(result, variables)
      setData(result)
      setIsSuccess(true)
      return result
    } catch (err) {
      setError(err)
      throw err
    } finally {
      setIsPending(false)
    }
  }, [mutationFn, onSuccess])

  return {
    mutateAsync,
    mutate: (variables: TVars) => {
      void mutateAsync(variables)
    },
    isPending,
    isSuccess,
    data,
    error,
    reset: () => {
      setIsPending(false)
      setIsSuccess(false)
      setData(undefined)
      setError(null)
    },
  }
}

function loadSuspenseResource<T>(queryKey: QueryKey, queryFn: () => Promise<T>): SuspenseResource<T> {
  const cacheKey = serializeKey(queryKey)
  const cached = suspenseCache.get(cacheKey) as SuspenseResource<T> | undefined
  if (cached) {
    return cached
  }

  const resource: SuspenseResource<T> = { status: "pending" }
  resource.promise = queryFn().then(
    (data) => {
      resource.status = "success"
      resource.data = data
    },
    (error) => {
      resource.status = "error"
      resource.error = error
    },
  )
  suspenseCache.set(cacheKey, resource)
  return resource
}

export function useSuspenseQuery<T>({ queryKey, queryFn }: CompatQueryOptions<T>): { data: T; refetch: () => Promise<void> } {
  const [, forceUpdate] = useReducer((value) => value + 1, 0)

  useEffect(() => {
    const entry = {
      key: queryKey,
      callback: () => {
        suspenseCache.delete(serializeKey(queryKey))
        forceUpdate()
      },
    }
    querySubscribers.add(entry)
    return () => {
      querySubscribers.delete(entry)
    }
  }, [queryKey])

  const resource = loadSuspenseResource(queryKey, queryFn)
  if (resource.status === "pending") {
    throw resource.promise
  }
  if (resource.status === "error") {
    throw resource.error
  }

  return {
    data: resource.data as T,
    refetch: async () => {
      suspenseCache.delete(serializeKey(queryKey))
      forceUpdate()
    },
  }
}
