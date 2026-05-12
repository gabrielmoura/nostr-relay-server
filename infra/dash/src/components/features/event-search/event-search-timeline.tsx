import { useTranslation } from "react-i18next"

import { Card, CardContent } from "@/components/ui/card"
import { Area, AreaChart, ResponsiveContainer, Tooltip, XAxis } from "recharts"
import { formatDateTime } from "@/lib/utils"

interface TimelinePoint {
  ts: number
  count: number
}

interface TimelineData {
  points: TimelinePoint[]
}

interface EventSearchTimelineProps {
  data?: TimelineData
  isLoading: boolean
  isError: boolean
  onRetry: () => void
}

export function EventSearchTimeline({ data, isLoading, isError, onRetry }: EventSearchTimelineProps) {
  const { t } = useTranslation()

  if (isLoading) return null
  if (isError) return null
  if (!data) return null

  return (
    <div className="space-y-4">
      {data.points.length === 0 ? (
        <Card>
          <CardContent className="space-y-3 p-4">
            <p className="text-center text-sm text-muted-foreground">{t("eventSearch.timelineEmptyTitle")}</p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="p-4 pt-6">
            <div className="h-48 w-full">
              <ResponsiveContainer height="100%" width="100%">
                <AreaChart data={data.points} margin={{ top: 0, right: 0, left: 0, bottom: 0 }}>
                  <defs>
                    <linearGradient id="colorCount" x1="0" x2="0" y1="0" y2="1">
                      <stop offset="5%" stopColor="var(--color-primary)" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="var(--color-primary)" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <XAxis
                    dataKey="ts"
                    tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }}
                    tickFormatter={(value: number) => {
                      const date = new Date(value * 1000)
                      return `${String(date.getUTCMonth() + 1).padStart(2, "0")}/${String(date.getUTCFullYear()).slice(-2)}`
                    }}
                    tickLine={false}
                    axisLine={false}
                  />
                  <Tooltip
                    content={({ active, payload }: any) => {
                      if (active && payload && payload.length) {
                        return (
                          <div className="rounded border border-border bg-background p-2 text-xs shadow-sm">
                            <p className="mb-1 font-medium text-foreground">{formatDateTime(payload[0].payload.ts)}</p>
                            <p className="text-muted-foreground">{t("eventSearch.eventsCount", { count: payload[0].value })}</p>
                          </div>
                        )
                      }
                      return null
                    }}
                  />
                  <Area
                    dataKey="count"
                    fill="url(#colorCount)"
                    stroke="var(--color-primary)"
                    strokeWidth={2}
                    type="monotone"
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
