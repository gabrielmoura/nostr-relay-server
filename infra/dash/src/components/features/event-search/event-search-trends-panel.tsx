import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { EventAggregates } from "@/types/admin"

interface EventSearchTrendsPanelProps {
  aggregates?: EventAggregates
}

export function EventSearchTrendsPanel({ aggregates }: EventSearchTrendsPanelProps) {
  const trends = aggregates?.trends
  const cards = [
    { title: "Tag do mês", value: trends?.top_tag_month || "-", helper: trends?.top_tag_month_count ? `${trends.top_tag_month_count} eventos` : "Sem dado" },
    { title: "Tag do ano", value: trends?.top_tag_year || "-", helper: trends?.top_tag_year_count ? `${trends.top_tag_year_count} eventos` : "Sem dado" },
    { title: "Pico do mês", value: trends?.peak_month || "-", helper: trends?.peak_month_count ? `${trends.peak_month_count} eventos` : "Sem dado" },
    { title: "Pico do ano", value: trends?.peak_year || "-", helper: trends?.peak_year_count ? `${trends.peak_year_count} eventos` : "Sem dado" },
  ]

  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      {cards.map((card) => (
        <Card className="border-primary/5 bg-card/60 backdrop-blur-sm" key={card.title}>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-semibold">{card.title}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-lg font-mono font-bold text-primary">{card.value}</p>
            <p className="mt-1 text-xs text-muted-foreground">{card.helper}</p>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
