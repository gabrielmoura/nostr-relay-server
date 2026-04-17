import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import type { EmbeddedRepost } from "@/lib/event-parser"
import { shortenId } from "@/lib/utils"

interface EventRepostCardProps {
  repost: EmbeddedRepost
}

export function EventRepostCard({ repost }: EventRepostCardProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/20 p-4">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("eventDetail.repostedEvent")}</p>
      <p className="text-sm text-foreground">
        {t("eventDetail.kindValue", { kind: repost.kind })} · {shortenId(repost.id, 12, 4)}
      </p>
      {repost.pubkey ? (
        <p className="text-xs text-muted-foreground">{t("eventDetail.author")}: {shortenId(repost.pubkey, 12, 4)}</p>
      ) : null}
      {repost.content ? <p className="line-clamp-4 text-sm text-foreground">{repost.content}</p> : null}
      <Button asChild size="sm" title={t("eventDetail.openOriginalFullscreen")} variant="outline">
        <Link params={{ eventId: repost.id }} to="/events/$eventId">{t("eventDetail.goToOriginal")}</Link>
      </Button>
    </div>
  )
}