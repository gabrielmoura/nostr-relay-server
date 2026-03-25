import { useEffect, useMemo, useRef } from "react"
import { useVirtualizer } from "@tanstack/react-virtual"

import { cn } from "@/lib/utils"

type VirtualizedListProps<T> = {
  items: T[]
  total: number
  estimateSize: number
  fetchMore?: () => void
  hasMore?: boolean
  isFetchingMore?: boolean
  className?: string
  heightClassName?: string
  renderItem: (item: T, index: number) => React.ReactNode
}

export function VirtualizedList<T>({
  items,
  total,
  estimateSize,
  fetchMore,
  hasMore,
  isFetchingMore,
  className,
  heightClassName,
  renderItem,
}: VirtualizedListProps<T>) {
  const parentRef = useRef<HTMLDivElement | null>(null)
  const rowCount = useMemo(() => (hasMore ? items.length + 1 : items.length), [hasMore, items.length])

  const virtualizer = useVirtualizer({
    count: rowCount,
    getScrollElement: () => parentRef.current,
    estimateSize: () => estimateSize,
    overscan: 8,
  })

  const virtualItems = virtualizer.getVirtualItems()

  useEffect(() => {
    const last = virtualItems.at(-1)
    if (!last || !hasMore || isFetchingMore || !fetchMore) {
      return
    }
    if (last.index >= items.length - 5) {
      fetchMore()
    }
  }, [fetchMore, hasMore, isFetchingMore, items.length, virtualItems])

  return (
    <div className={cn("rounded-[var(--radius)] border border-border bg-card", className)}>
      <div className={cn("max-h-[70vh] overflow-auto", heightClassName)} ref={parentRef}>
        <div className="relative w-full" style={{ height: `${virtualizer.getTotalSize()}px` }}>
          {virtualItems.map((virtualItem) => {
            const item = items[virtualItem.index]
            return (
              <div
                className="absolute top-0 left-0 w-full px-3 py-2"
                data-index={virtualItem.index}
                key={virtualItem.key}
                ref={virtualizer.measureElement}
                style={{ transform: `translateY(${virtualItem.start}px)` }}
              >
                {item ? renderItem(item, virtualItem.index) : hasMore ? <div className="rounded-md border border-dashed border-border px-4 py-6 text-center text-sm text-muted-foreground">Carregando mais resultados...</div> : null}
              </div>
            )
          })}
        </div>
      </div>
      <div className="border-t border-border px-3 py-2 text-xs text-muted-foreground">{items.length} de {total} itens carregados</div>
    </div>
  )
}
