import { Activity, Ban, Cable, Database, ShieldCheck, Users } from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"
import type { RelayMetricCard } from "@/types/admin"
import { cn } from "@/lib/utils"

const iconMap = {
  "Conexoes ativas": Cable,
  "Conexoes logadas": Users,
  "Usuarios banidos": Ban,
  "Eventos indexados": Database,
  "Eventos / min": Activity,
  "Status do relay": ShieldCheck,
} as const

export function MetricCard({ card }: { card: RelayMetricCard }) {
  const Icon = iconMap[card.label as keyof typeof iconMap] ?? Activity

  return (
    <Card className="overflow-hidden">
      <CardContent className="flex h-full flex-col gap-4 p-4">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">{card.label}</span>
          <div
            className={cn(
              "rounded-full p-2",
              card.tone === "success" && "bg-emerald-50 text-emerald-600",
              card.tone === "danger" && "bg-red-50 text-red-600",
              card.tone === "warning" && "bg-orange-50 text-orange-600",
              !card.tone && "bg-stone-100 text-stone-700",
            )}
          >
            <Icon className="size-4" />
          </div>
        </div>
        <div>
          <p className={cn("font-mono text-3xl font-semibold tracking-[-0.04em] text-foreground", card.tone === "success" && "text-emerald-600", card.tone === "danger" && "text-red-600")}>{card.value}</p>
          {card.helper ? <p className="mt-1 text-xs text-muted-foreground">{card.helper}</p> : null}
        </div>
      </CardContent>
    </Card>
  )
}
