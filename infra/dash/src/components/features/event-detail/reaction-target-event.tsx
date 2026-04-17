import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { useEventDetail } from "@/hooks/use-admin-data"
import { shortenId } from "@/lib/utils"

interface ReactionTargetEventProps {
  eventID: string
}

export function ReactionTargetEvent({ eventID }: ReactionTargetEventProps) {
  const { t } = useTranslation()
  const targetEventQuery = useEventDetail(eventID)

  if (!targetEventQuery.data) {
    return null
  }

  return (
    <div className="space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/20 p-4">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("eventDetail.reactionTargetEvent")}</p>
      {targetEventQuery.isLoading ? <p className="text-sm text-muted-foreground">{t("eventDetail.loadingTargetEvent")}</p> : null}
      {targetEventQuery.data ? (
        <>
          <p className="text-sm text-foreground">
            {t("eventDetail.kindValue", { kind: targetEventQuery.data.event.kind })} · {shortenId(targetEventQuery.data.event.id, 12, 4)}
          </p>
          <p className="line-clamp-3 text-sm text-muted-foreground">
            {targetEventQuery.data.event.content || t("eventDetail.noTextContent")}
          </p>
          <Button asChild size="sm" title={t("eventDetail.openFullTargetEvent")} variant="outline">
            <Link params={{ eventId: eventID }} to="/events/$eventId">{t("eventDetail.viewFullTargetEvent")}</Link>
          </Button>
        </>
      ) : null}
      {targetEventQuery.isError ? <p className="text-xs text-muted-foreground">{t("eventDetail.targetLoadError")}</p> : null}
    </div>
  )
}