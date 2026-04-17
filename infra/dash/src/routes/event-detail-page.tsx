import React, { Suspense } from "react"
import { QueryErrorResetBoundary } from "@tanstack/react-query"
import { useParams } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { BanUserDialog } from "@/components/features/ban-user-dialog"
import { PageHeader } from "@/components/shared/page-header"
import { ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  EventContentWarnings,
  EventHashtags,
  EventMetadata,
  EventStructuredData,
} from "@/components/features/event-detail/event-metadata"
import { EventMedia, EventImetaInfo } from "@/components/features/event-detail/event-media"
import { EventRepostCard } from "@/components/features/event-detail/event-repost-card"
import { ReactionTargetEvent } from "@/components/features/event-detail/reaction-target-event"
import { EventListItems } from "@/components/features/event-detail/event-list-items"
import {
  EventRefList,
  TagBadgeList,
  AddressRefList,
  LinksAndQuotes,
} from "@/components/features/event-detail/nostr-references"
import { EventDetailErrorState } from "@/components/features/event-detail/event-detail-error-state"
import { useEventDetailSuspense } from "@/hooks/use-admin-data"
import { firstTagValue, parseImetaResources, parseEventRefTags, parseEmbeddedRepost, tagValues, REACTION_KIND, REPOST_KINDS, LIST_KIND, unique } from "@/lib/event-parser"

type EventDetailBoundaryProps = {
  children: React.ReactNode
  fallbackRender: (error: Error, reset: () => void) => React.ReactNode
  onReset?: () => void
}

type EventDetailBoundaryState = {
  error: Error | null
}

class EventDetailErrorBoundary extends React.Component<EventDetailBoundaryProps, EventDetailBoundaryState> {
  state: EventDetailBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error) {
    return { error }
  }

  reset = () => {
    this.setState({ error: null })
    this.props.onReset?.()
  }

  render() {
    if (this.state.error) {
      return this.props.fallbackRender(this.state.error, this.reset)
    }
    return this.props.children
  }
}

function EventDetailPageContent({ eventID }: { eventID: string }) {
  const { t } = useTranslation()
  const query = useEventDetailSuspense(eventID)
  const detail = query.data
  const event = detail.event
  const tags = event.tags ?? []

  const title = firstTagValue(tags, "title")
  const summary = firstTagValue(tags, "summary")
  const contentWarning = firstTagValue(tags, "content-warning")
  const alt = firstTagValue(tags, "alt")
  const eventD = firstTagValue(tags, "d")
  const publishedAt = firstTagValue(tags, "published_at")

  const topicTags = unique([...detail.hashtags, ...tagValues(tags, "t")])
  const eRefs = unique(tagValues(tags, "e"))
  const pRefs = unique(tagValues(tags, "p"))
  const aRefs = unique(tagValues(tags, "a"))
  const qRefs = unique(tagValues(tags, "q"))
  const rRefs = unique(tagValues(tags, "r"))
  const kRefs = unique(tagValues(tags, "k").map(Number))

  const imeta = parseImetaResources(tags)
  const embeddedRepost = parseEmbeddedRepost(event.content, event.kind)

  const imageURLs = unique([...detail.image_urls, ...imeta.imageURLs])
  const mediaURLs = unique([...imeta.mediaURLs, ...detail.image_urls])

  const targetEventID = eRefs[0] ?? ""
  const listRefs = event.kind === LIST_KIND ? parseEventRefTags(tags) : []
  const kindLabel = t(`eventDetail.kindLabels.${event.kind}`, { defaultValue: t("eventDetail.specializedKind") })

  return (
    <div className="space-y-6">
      <PageHeader
        actions={
          <>
            <BanUserDialog
              contextEventId={event.id}
              defaultPubkey={event.pubkey}
              defaultReason={t("moderation.ban.defaultReasonFromEvent", { eventId: event.id.slice(0, 14) })}
              triggerLabel={t("moderation.ban.trigger")}
              triggerVariant="warning"
            />
          </>
        }
        description={t("eventDetail.description")}
        title={t("eventDetail.title")}
      />

      <div className="grid gap-4 xl:grid-cols-[1.3fr_0.7fr]">
        <Card>
          <CardHeader>
            <CardTitle>{t("eventDetail.event")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <EventMetadata event={event} author={detail.author} kindLabel={kindLabel} />
            <EventContentWarnings contentWarning={contentWarning} />
            <EventStructuredData alt={alt} createdAt={event.created_at} eventD={eventD} publishedAt={publishedAt} summary={summary} title={title} />

            <p className="max-w-full overflow-hidden whitespace-pre-wrap break-all rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/50 p-4 text-sm leading-relaxed text-foreground">
              {event.content || t("eventDetail.noTextContent")}
            </p>

            <EventHashtags tags={topicTags} />
            <EventMedia content={event.content} imageURLs={imageURLs} kind={event.kind} tags={tags} />

            {REPOST_KINDS.includes(event.kind) && embeddedRepost && <EventRepostCard repost={embeddedRepost} />}
            {REACTION_KIND === event.kind && targetEventID && <ReactionTargetEvent eventID={targetEventID} />}
            {LIST_KIND === event.kind && listRefs.length > 0 && <EventListItems refs={listRefs} />}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("eventDetail.nostrIdentifiers")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {Object.entries(detail.identifiers).map(([key, value]) =>
              value ? (
                <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3" key={key}>
                  <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{key}</p>
                  <p className="mt-1 break-all font-mono text-xs text-foreground">{value}</p>
                </div>
              ) : null
            )}
            <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
              <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{t("eventDetail.eventIdLabel")}</p>
              <p className="mt-1 break-all font-mono text-xs text-foreground">{event.id}</p>
            </div>

            <TagBadgeList label={t("eventDetail.kTags")} tags={kRefs} />
            <AddressRefList label={t("eventDetail.aReferences")} refs={aRefs} />
            <EventRefList label={t("eventDetail.eReferences")} refs={eRefs} type="event" />
            <EventRefList label={t("eventDetail.pReferences")} refs={pRefs} type="user" />
            <LinksAndQuotes qRefs={qRefs} rRefs={rRefs} />
            <EventImetaInfo altTexts={imeta.altTexts} mimeTypes={imeta.mimeTypes} />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export function EventDetailPage() {
  const { t } = useTranslation()
  const { eventId } = useParams({ from: "/events/$eventId" })

  return (
    <QueryErrorResetBoundary>
      {({ reset }) => (
        <EventDetailErrorBoundary
          fallbackRender={(error, resetBoundary) => (
            <EventDetailErrorState
              error={error}
              eventID={eventId}
              onRetry={() => {
                reset()
                resetBoundary()
              }}
            />
          )}
          onReset={reset}
        >
          <Suspense fallback={<LoadingPanel label={t("eventDetail.loading")} />}>
            <EventDetailPageContent eventID={eventId} />
          </Suspense>
        </EventDetailErrorBoundary>
      )}
    </QueryErrorResetBoundary>
  )
}