import { Database, HardDrive, RadioTower, Users } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { formatBytes, formatCount } from "@/lib/utils"
import type { BlossomOverview } from "@/types/admin"

export function BlossomKpiStrip({ overview }: { overview: BlossomOverview }) {
  const { t } = useTranslation()

  const cards = [
    {
      key: "storage",
      icon: HardDrive,
      title: t("blossom.kpis.storage", "Disco em uso"),
      value: formatBytes(overview.storage.used_bytes),
      helper: `${overview.storage.used_percent.toFixed(1)}% ${t("blossom.kpis.ofCapacity", "da capacidade monitorada")}`,
    },
    {
      key: "objects",
      icon: Database,
      title: t("blossom.kpis.objects", "Arquivos"),
      value: formatCount(overview.objects.total),
      helper: `${formatCount(overview.objects.pending_review)} ${t("blossom.kpis.pendingReview", "em revisão")}`,
    },
    {
      key: "traffic",
      icon: RadioTower,
      title: t("blossom.kpis.egress", "Egress do mês"),
      value: formatBytes(overview.traffic.monthly_egress_bytes),
      helper: `${formatBytes(overview.traffic.monthly_ingress_bytes)} ${t("blossom.kpis.ingressMonth", "de ingress no mês")}`,
    },
    {
      key: "users",
      icon: Users,
      title: t("blossom.kpis.users", "Pubkeys ativas"),
      value: formatCount(overview.users.active),
      helper: `${formatCount(overview.users.whitelisted)} ${t("blossom.kpis.whitelisted", "na whitelist")}`,
    },
  ]

  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
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
