import { useTranslation } from "react-i18next"

import { EventMedia } from "@/components/features/event-detail/event-media"
import { useReferencedNostrEvent } from "@/hooks/use-referenced-nostr-event"
import type { EmbeddedRepost } from "@/lib/event-parser"

interface ApprovedEventPreviewProps {
  event?: EmbeddedRepost | null
  eventID: string
}

export function ApprovedEventPreview({ event, eventID }: ApprovedEventPreviewProps) {
  const { t } = useTranslation()
  const referencedEvent = useReferencedNostrEvent(event ? "" : eventID)
  const resolved = event ?? (referencedEvent.data ? {
    id: referencedEvent.data.id,
    kind: referencedEvent.data.kind,
    pubkey: referencedEvent.data.pubkey,
    content: referencedEvent.data.content,
    tags: referencedEvent.data.tags,
    created_at: referencedEvent.data.created_at,
  } : null)

  if (!resolved) {
    return null
  }

  return (
    <div className="min-w-0 space-y-3 rounded-md border border-primary/10 bg-background/70 p-4">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("eventDetail.approvedEventPreview", "Evento aprovado")}</p>
      {resolved.content ? <p className="break-words whitespace-pre-wrap text-sm text-foreground">{resolved.content}</p> : null}
      <EventMedia content={resolved.content} imageURLs={[]} kind={resolved.kind} tags={resolved.tags} />
    </div>
  )
}
