import { ListRefSyncCard } from "./list-ref-sync-card"

interface EventListItemsProps {
  refs: Array<{ id: string; relay?: string }>
}

export function EventListItems({ refs }: EventListItemsProps) {
  if (refs.length === 0) return null

  return (
    <div className="space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/20 p-4">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{"eventDetail.listItems"}</p>
      <div className="space-y-2">
        {refs.map((ref, idx) => (
          <ListRefSyncCard eventID={ref.id} key={`${ref.id}-${idx}`} preferredRelay={ref.relay} />
        ))}
      </div>
    </div>
  )
}