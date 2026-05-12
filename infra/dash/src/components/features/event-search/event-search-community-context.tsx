import { Badge } from "@/components/ui/badge"

interface EventSearchCommunityContextProps {
  badge: string
  imageUrl?: string
  label: string
}

export function EventSearchCommunityContext({ badge, imageUrl, label }: EventSearchCommunityContextProps) {
  return (
    <div className="flex min-w-0 items-center gap-2 rounded-md border border-emerald-500/25 bg-emerald-500/8 px-2.5 py-2 text-[11px] text-emerald-700 dark:text-emerald-300">
      {imageUrl ? <img alt={label} className="size-8 shrink-0 rounded-md border border-emerald-500/20 object-cover" src={imageUrl} /> : null}
      <Badge variant="muted" className="shrink-0 border-emerald-500/30 bg-background/85 text-[9px] font-semibold text-emerald-700 dark:text-emerald-300">
        {badge}
      </Badge>
      <div className="min-w-0">
        <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-emerald-700/80 dark:text-emerald-300/80">Comunidade</p>
        <p className="truncate text-sm font-medium text-foreground">{label}</p>
      </div>
    </div>
  )
}
