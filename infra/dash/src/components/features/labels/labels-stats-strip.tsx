import { Hash, Tags, Waypoints } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { AdminLabelsSummary } from "@/types/admin"
import { formatCount } from "@/lib/utils"
import { topSummaryLabel } from "@/lib/labels"

export function LabelsStatsStrip({ summary }: { summary: AdminLabelsSummary }) {
  const { t } = useTranslation()

  const cards = [
    {
      key: "events",
      icon: Tags,
      title: t("labels.stats.events", "Label events"),
      value: formatCount(summary.total_events),
      helper: t("labels.stats.eventsHelper", "Total de eventos kind 1985 no filtro atual."),
    },
    {
      key: "targets",
      icon: Waypoints,
      title: t("labels.stats.targets", "Distinct targets"),
      value: formatCount(summary.total_targets),
      helper: t("labels.stats.targetsHelper", "Entidades unicas atingidas pelos labels filtrados."),
    },
    {
      key: "topLabel",
      icon: Hash,
      title: t("labels.stats.topLabel", "Top label"),
      value: topSummaryLabel(summary.labels),
      helper: t("labels.stats.topLabelHelper", "Label mais recorrente dentro do recorte atual."),
    },
  ]

  return (
    <div className="grid gap-4 md:grid-cols-3">
      {cards.map((card) => {
        const Icon = card.icon
        return (
          <Card className="border-border/70 bg-card/95" key={card.key}>
            <CardHeader className="flex flex-row items-center justify-between gap-3 pb-3">
              <CardTitle className="text-sm text-muted-foreground">{card.title}</CardTitle>
              <div className="rounded-full bg-secondary p-2 text-primary">
                <Icon className="size-4" />
              </div>
            </CardHeader>
            <CardContent>
              <p className="font-heading text-2xl font-semibold text-foreground">{card.value}</p>
              <p className="mt-1 text-xs text-muted-foreground">{card.helper}</p>
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}
