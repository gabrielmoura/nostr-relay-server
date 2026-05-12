import { useTranslation } from "react-i18next"
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { EventAggregateTag } from "@/types/admin"

interface EventSearchTopTagsChartProps {
  items: EventAggregateTag[]
  onTagSelect?: (tag: string) => void
}

export function EventSearchTopTagsChart({ items, onTagSelect }: EventSearchTopTagsChartProps) {
  const { t } = useTranslation()

  return (
    <Card className="border-primary/5 bg-card/60 backdrop-blur-sm">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-semibold">{t("eventSearch.commonTags")}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-[240px] w-full">
          <ResponsiveContainer height="100%" width="100%">
            <PieChart>
              <Pie data={items} cx="50%" cy="50%" innerRadius={48} outerRadius={82} paddingAngle={2} dataKey="count" nameKey="tag" onClick={(payload: any) => payload?.tag && onTagSelect?.(payload.tag)}>
                {items.map((item, index) => (
                  <Cell key={item.tag} fill={`oklch(var(--color-primary) / ${Math.max(0.22, 1 - (index * 0.12))})`} style={{ cursor: onTagSelect ? "pointer" : "default" }} />
                ))}
              </Pie>
              <Tooltip content={({ active, payload }: any) => active && payload?.length ? (
                <div className="rounded border border-border bg-background p-2 text-xs shadow-sm">
                  <p className="font-medium text-foreground">#{payload[0].name}</p>
                  <p className="text-muted-foreground">{payload[0].value} eventos</p>
                </div>
              ) : null} />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  )
}
