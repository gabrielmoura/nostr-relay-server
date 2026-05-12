import { Card, CardContent } from "@/components/ui/card"
import type { EventAggregates } from "@/types/admin"

interface EventSearchAnalyticsKpiStripProps {
  aggregates?: EventAggregates
}

export function EventSearchAnalyticsKpiStrip({ aggregates }: EventSearchAnalyticsKpiStripProps) {
  const total = aggregates?.total ?? 0
  const topKind = aggregates?.kinds[0]?.kind
  const topTag = aggregates?.top_tags[0]?.tag
  const activeAuthors = aggregates?.top_authors.length ?? 0

  const items = [
    { label: "Total global", value: total.toLocaleString(), helper: "Eventos no recorte atual" },
    { label: "Autores ativos", value: activeAuthors.toLocaleString(), helper: "Autores líderes no recorte" },
    { label: "Kind dominante", value: topKind ? String(topKind) : "-", helper: topTag ? `Tag líder: #${topTag}` : "Sem tag dominante" },
  ]

  return (
    <div className="grid gap-3 md:grid-cols-3">
      {items.map((item) => (
        <Card className="overflow-hidden border-primary/5 bg-card/60 backdrop-blur-sm" key={item.label}>
          <CardContent className="p-4">
            <p className="text-xs font-bold uppercase tracking-wider text-muted-foreground">{item.label}</p>
            <p className="mt-2 text-xl font-mono font-bold tracking-tight text-primary leading-none">{item.value}</p>
            <p className="mt-1 text-xs text-muted-foreground">{item.helper}</p>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
