import { useTranslation } from "react-i18next"
import { Area, AreaChart, ResponsiveContainer, Tooltip, XAxis } from "recharts"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { ReportedEventsTimelinePoint } from "@/types/admin"

interface ReportedEventsTrendChartProps {
  points: ReportedEventsTimelinePoint[]
  onBucketSelect?: (bucket: string) => void
}

export function ReportedEventsTrendChart({ points, onBucketSelect }: ReportedEventsTrendChartProps) {
  const { t } = useTranslation()
  const handleActiveDotClick = ((_: unknown, eventData: any) => {
    const bucket = eventData?.payload?.bucket
    if (bucket) {
      onBucketSelect?.(bucket)
    }
  }) as any

  return (
    <Card className="border-primary/5 bg-card/60 backdrop-blur-sm">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-semibold">{t("reported.charts.volumeTitle", "Volume de reports")}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-[220px] w-full">
          <ResponsiveContainer height="100%" width="100%">
            <AreaChart data={points} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="reportedVolume" x1="0" x2="0" y1="0" y2="1">
                  <stop offset="5%" stopColor="var(--color-primary)" stopOpacity={0.35} />
                  <stop offset="95%" stopColor="var(--color-primary)" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="bucket" tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }} tickLine={false} axisLine={false} />
              <Tooltip content={({ active, payload }: any) => active && payload?.length ? (
                <div className="rounded border border-border bg-background p-2 text-xs shadow-sm">
                  <p className="font-medium text-foreground">{payload[0].payload.bucket}</p>
                  <p className="text-muted-foreground">{payload[0].value} reports</p>
                </div>
              ) : null} />
              <Area activeDot={{ onClick: handleActiveDotClick, r: 5, style: { cursor: "pointer" } }} dataKey="count" fill="url(#reportedVolume)" stroke="var(--color-primary)" strokeWidth={2} type="monotone" />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  )
}
