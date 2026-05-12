import { useTranslation } from "react-i18next"
import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { ReportedEventsTargetCount } from "@/types/admin"
import { shortenId } from "@/lib/utils"

interface ReportedEventsTopTargetsChartProps {
  items: ReportedEventsTargetCount[]
  onTargetSelect?: (eventID: string) => void
}

export function ReportedEventsTopTargetsChart({ items, onTargetSelect }: ReportedEventsTopTargetsChartProps) {
  const { t } = useTranslation()
  const handleBarClick = ((payload: any) => {
    const eventID = payload?.target_event_id || payload?.payload?.target_event_id
    if (eventID) {
      onTargetSelect?.(eventID)
    }
  }) as any

  return (
    <Card className="border-primary/5 bg-card/60 backdrop-blur-sm">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-semibold">{t("reported.charts.targetsTitle", "Eventos mais reportados")}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-[220px] w-full">
          <ResponsiveContainer height="100%" width="100%">
            <BarChart data={items} layout="vertical" margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
              <XAxis hide type="number" />
              <YAxis dataKey="target_event_id" type="category" width={110} tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }} tickFormatter={(value: string) => shortenId(value, 12, 4)} tickLine={false} axisLine={false} />
              <Tooltip content={({ active, payload }: any) => active && payload?.length ? (
                <div className="rounded border border-border bg-background p-2 text-xs shadow-sm">
                  <p className="font-medium text-foreground">{shortenId(payload[0].payload.target_event_id, 18, 6)}</p>
                  <p className="text-muted-foreground">{payload[0].value} reports</p>
                </div>
              ) : null} />
              <Bar dataKey="count" fill="#f59e0b" onClick={handleBarClick} radius={[0, 4, 4, 0]} style={{ cursor: "pointer" }} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  )
}
