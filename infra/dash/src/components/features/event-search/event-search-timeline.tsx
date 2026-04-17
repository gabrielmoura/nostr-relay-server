import { useState } from "react"
import { useTranslation } from "react-i18next"

import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
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
  const [bucket, setBucket] = useState<"hour" | "day">("hour")

  if (isLoading) return null
  if (isError) return null
  if (!data) return null

  const maxCount = data.points[0]?.count ?? 1

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <Button onClick={() => setBucket("hour")} size="sm" variant={bucket === "hour" ? "default" : "outline"}>
          {t("eventSearch.hour")}
        </Button>
        <Button onClick={() => setBucket("day")} size="sm" variant={bucket === "day" ? "default" : "outline"}>
          {t("eventSearch.day")}
        </Button>
      </div>

      {data.points.length === 0 ? (
        <Card>
          <CardContent className="space-y-3 p-4">
            <p className="text-center text-sm text-muted-foreground">{t("eventSearch.timelineEmptyTitle")}</p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="space-y-3 p-4">
            {data.points.map((point) => (
              <div className="space-y-1" key={point.ts}>
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>{formatDateTime(point.ts)}</span>
                  <span>{t("eventSearch.eventsCount", { count: point.count })}</span>
                </div>
                <div className="h-2 rounded bg-muted">
                  <div
                    className="h-full rounded bg-primary"
                    style={{ width: `${Math.max(3, (point.count / maxCount) * 100)}%` }}
                  />
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  )
}