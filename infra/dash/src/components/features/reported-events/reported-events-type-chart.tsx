import { useTranslation } from "react-i18next"
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { reportTypeColor } from "@/lib/reported-events-analytics"
import type { ReportedEventsTypeCount } from "@/types/admin"

interface ReportedEventsTypeChartProps {
  items: ReportedEventsTypeCount[]
  onTypeSelect?: (type: string) => void
}

export function ReportedEventsTypeChart({ items, onTypeSelect }: ReportedEventsTypeChartProps) {
  const { t } = useTranslation()

  return (
    <Card className="border-primary/5 bg-card/60 backdrop-blur-sm">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-semibold">{t("reported.charts.typesTitle", "Distribuição por tipo")}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-[220px] w-full">
          <ResponsiveContainer height="100%" width="100%">
            <PieChart>
              <Pie cx="50%" cy="50%" data={items} dataKey="count" innerRadius={48} outerRadius={82} nameKey="name" paddingAngle={2} onClick={(payload: { name?: string }) => payload.name && onTypeSelect?.(payload.name)}>
                {items.map((item, index) => (
                  <Cell key={item.name} fill={reportTypeColor(item.name)} style={{ cursor: "pointer" }} />
                ))}
              </Pie>
              <Tooltip content={({ active, payload }: any) => active && payload?.length ? (
                <div className="rounded border border-border bg-background p-2 text-xs shadow-sm">
                  <p className="font-medium text-foreground">{payload[0].name}</p>
                  <p className="text-muted-foreground">{payload[0].value} ocorrências</p>
                </div>
              ) : null} />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  )
}
