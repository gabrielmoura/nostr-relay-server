import React from "react"
import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { formatDateTime, shortenId } from "@/lib/utils"
import type { EventAuthor } from "@/types/admin"

interface EventMetadataProps {
  event: {
    kind: number
    pubkey: string
    created_at: number
  }
  author: EventAuthor
  kindLabel: string
}

export function EventMetadata({ event, author, kindLabel }: EventMetadataProps) {
  const { t } = useTranslation()

  return (
    <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
      <Badge variant="muted">{t("eventDetail.kindValue", { kind: event.kind })}</Badge>
      <Badge variant="muted">{kindLabel}</Badge>
      <Badge variant="muted">{formatDateTime(event.created_at)}</Badge>
      {author?.nip05 ? (
        <TooltipProvider delayDuration={150}>
          <Tooltip>
            <TooltipTrigger asChild>
              <Link className="underline decoration-dotted underline-offset-2" params={{ pubkey: event.pubkey }} to="/users/$pubkey">
                {author.display_name || `${t("eventDetail.author")} ${shortenId(event.pubkey, 12, 4)}`}
              </Link>
            </TooltipTrigger>
            <TooltipContent>{author.nip05}</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ) : (
        <Link className="underline decoration-dotted underline-offset-2" params={{ pubkey: event.pubkey }} to="/users/$pubkey">
          {author?.display_name || `${t("eventDetail.author")} ${shortenId(event.pubkey, 12, 4)}`}
        </Link>
      )}
    </div>
  )
}

interface EventContentWarningsProps {
  contentWarning: string
}

export function EventContentWarnings({ contentWarning }: EventContentWarningsProps) {
  const { t } = useTranslation()

  if (!contentWarning) return null

  return <Badge variant="warning">{t("eventDetail.contentWarningLabel")}: {contentWarning}</Badge>
}

interface EventStructuredDataProps {
  title?: string
  summary?: string
  alt?: string
  publishedAt?: string
  eventD?: string
  createdAt: number
}

export function EventStructuredData({ title, summary, alt, publishedAt, eventD, createdAt }: EventStructuredDataProps) {
  const { t } = useTranslation()

  if (!title && !summary && !alt && !publishedAt && !eventD) return null

  return (
    <div className="space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/30 p-4 text-sm">
      {title ? <p><span className="font-semibold">{t("eventDetail.titleLabel")}:</span> {title}</p> : null}
      {summary ? <p><span className="font-semibold">{t("eventDetail.summary")}:</span> {summary}</p> : null}
      {alt ? <p><span className="font-semibold">{t("eventDetail.altLabel")}:</span> {alt}</p> : null}
      {publishedAt ? <p><span className="font-semibold">{t("eventDetail.publishedAt")}:</span> {formatDateTime(Number(publishedAt) || createdAt)}</p> : null}
      {eventD ? <p className="break-all"><span className="font-semibold">{t("eventDetail.dTagLabel")}:</span> {eventD}</p> : null}
    </div>
  )
}

interface EventHashtagsProps {
  tags: string[]
}

export function EventHashtags({ tags }: EventHashtagsProps) {
  if (tags.length === 0) return null

  return (
    <div className="flex flex-wrap gap-2">
      {tags.map((tag) => (
        <Link key={tag} search={{ tags: tag }} to="/events/search">
          <Badge variant="muted">#{tag}</Badge>
        </Link>
      ))}
    </div>
  )
}