import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { useReferencedNostrEvent } from "@/hooks/use-referenced-nostr-event"
import { Button } from "@/components/ui/button"
import { collectAltTexts, type EmbeddedRepost } from "@/lib/event-parser"
import { shortenId } from "@/lib/utils"

interface EventRepostCardProps {
  repost: EmbeddedRepost
}

export function EventRepostCard({ repost }: EventRepostCardProps) {
  const { t } = useTranslation()
  const referencedEvent = useReferencedNostrEvent(repost.id)
  const fallbackEvent = referencedEvent.data
  const previewContent = repost.content || fallbackEvent?.content || ""
  const previewAlt = repost.tags.length > 0 ? collectAltTexts(repost.tags)[0] : collectAltTexts(fallbackEvent?.tags ?? [])[0]
  const resolvedKind = repost.kind > -1 ? repost.kind : fallbackEvent?.kind ?? -1
  const resolvedPubkey = repost.pubkey || fallbackEvent?.pubkey || ""

  return (
    <div className="space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/20 p-4 min-w-0">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("eventDetail.repostedEvent")}</p>
      <p className="text-sm text-foreground">
        {t("eventDetail.kindValue", { kind: resolvedKind })} · {shortenId(repost.id, 12, 4)}
      </p>
      {resolvedPubkey ? (
        <p className="text-xs text-muted-foreground">{t("eventDetail.author")}: {shortenId(resolvedPubkey, 12, 4)}</p>
      ) : null}
      {previewContent ? <p className="line-clamp-4 break-words text-sm text-foreground">{previewContent}</p> : null}
      {!previewContent && previewAlt ? <p className="text-sm text-muted-foreground">{t("eventDetail.altPrefix")} {previewAlt}</p> : null}
      <Button asChild size="sm" title={t("eventDetail.openOriginalFullscreen")} variant="outline">
        <Link params={{ eventId: repost.id }} to="/events/$eventId">{t("eventDetail.goToOriginal")}</Link>
      </Button>
    </div>
  )
}
