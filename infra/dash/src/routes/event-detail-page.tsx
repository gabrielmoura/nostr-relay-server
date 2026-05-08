import React, { Suspense } from "react"
import { Link } from "@tanstack/react-router"
import { QueryErrorResetBoundary } from "@tanstack/react-query"
import { useParams } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { BanUserDialog } from "@/components/features/ban-user-dialog"
import { PageHeader } from "@/components/shared/page-header"
import { ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { Button } from "@/components/ui/button"
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
import { UserAvatarChip } from "@/components/shared/user-avatar-chip"
import { Badge } from "@/components/ui/badge"
import { useEventDetailSuspense, useEventReports, useInfiniteEventSearch, useInfiniteLabels, useUser } from "@/hooks/use-admin-data"
import { labelBadgeVariant } from "@/lib/labels"
import { firstTagValue, parseCommunityMetadata, parseImetaResources, parseEventRefTags, parseEmbeddedRepost, tagValues, REACTION_KIND, REPOST_KINDS, LIST_KIND, unique } from "@/lib/event-parser"
import { shortenId } from "@/lib/utils"

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
  const community = event.kind === 34550 ? parseCommunityMetadata(tags) : null

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

  const relatedEvents = unique([...eRefs, ...qRefs].filter((value) => value && value !== event.id))
  const repliesQuery = useInfiniteEventSearch({ query: "", authors: [], kinds: [], tags: [`e:${event.id}`], limit: 20 })
  const replyEvents = repliesQuery.data?.pages.flatMap((page) => page.items) ?? []
  const labelsQuery = useInfiniteLabels({ target_type: "event", target: event.id, limit: 20 })
  const labelEvents = labelsQuery.data?.pages.flatMap((page) => page.items) ?? []
  const reportsQuery = useEventReports(event.id)
  const reports = reportsQuery.data?.pages.flatMap((page) => page.items) ?? []
  const respondingUsers = unique(replyEvents.map((item) => item.pubkey)).map((pubkey) => {
    const reply = replyEvents.find((item) => item.pubkey === pubkey)
    return { pubkey, reply }
  })
  const moderators = community?.moderators ?? []

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

            {community ? <CommunityMetadataCard dTag={community.d} description={community.description} image={community.image} /> : null}

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

      <div className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("eventDetail.relatedEvents", "Eventos associados")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {relatedEvents.length > 0 ? relatedEvents.map((relatedID) => (
              <div className="flex items-center justify-between gap-3 rounded-[calc(var(--radius)-0.25rem)] border border-border p-3" key={relatedID}>
                <p className="font-mono text-xs text-foreground">{shortenId(relatedID, 18, 6)}</p>
                <Link className="text-sm text-primary underline underline-offset-2" params={{ eventId: relatedID }} to="/events/$eventId">
                  {t("eventDetail.openEvent", "Abrir evento")}
                </Link>
              </div>
            )) : <p className="text-sm text-muted-foreground">{t("eventDetail.noRelatedEvents", "Nenhum evento associado encontrado nos tags e/q.")}</p>}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("eventDetail.responders", "Usuários que responderam")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {respondingUsers.length > 0 ? respondingUsers.map((user) => <ResponderCard key={user.pubkey} pubkey={user.pubkey} replyCount={replyEvents.filter((item) => item.pubkey === user.pubkey).length} />) : <p className="text-sm text-muted-foreground">{t("eventDetail.noResponders", "Nenhuma resposta indexada para este evento.")}</p>}
          </CardContent>
        </Card>

        {moderators.length > 0 ? (
          <Card>
            <CardHeader>
              <CardTitle>{t("eventDetail.moderators", "Usuários moderadores")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {moderators.map((pubkey) => <ResponderCard key={pubkey} pubkey={pubkey} />)}
            </CardContent>
          </Card>
        ) : null}

        <Card>
          <CardHeader>
            <CardTitle>{t("eventDetail.labels", "Labels")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {labelEvents.length > 0 ? labelEvents.map((labelEvent) => (
              <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3" key={labelEvent.id}>
                <div className="flex flex-wrap gap-2">
                  <Badge variant="muted">{labelEvent.namespace}</Badge>
                  {labelEvent.labels.map((label) => (
                    <Badge key={`${labelEvent.id}-${label}`} variant={labelBadgeVariant(label)}>{label}</Badge>
                  ))}
                </div>
                {labelEvent.content ? <p className="mt-2 text-sm text-foreground">{labelEvent.content}</p> : null}
                <p className="mt-2 font-mono text-xs text-muted-foreground">{shortenId(labelEvent.id, 18, 6)}</p>
              </div>
            )) : <p className="text-sm text-muted-foreground">{t("eventDetail.noLabels", "Nenhum label associado a este evento.")}</p>}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("eventDetail.reports", "Reports")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {reports.length > 0 ? reports.map((report) => (
              <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3" key={report.report_event_id}>
                <div className="flex items-center justify-between gap-3">
                  <p className="text-sm font-medium text-foreground">{report.report_type || t("reported.other", "other")}</p>
                  <p className="text-xs text-muted-foreground">{new Date(report.created_at * 1000).toLocaleString()}</p>
                </div>
                <p className="mt-1 text-sm text-muted-foreground">{report.content || t("reported.noComment")}</p>
                <p className="mt-2 font-mono text-xs text-muted-foreground">{shortenId(report.reporter_npub || report.reporter_pubkey, 16, 6)}</p>
              </div>
            )) : <p className="text-sm text-muted-foreground">{t("eventDetail.noReports", "Nenhum report associado a este evento.")}</p>}
          </CardContent>
        </Card>
      </div>

      {replyEvents.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>{t("eventDetail.replyEvents", "Eventos de resposta")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {replyEvents.map((reply) => (
              <div className="flex items-center justify-between gap-3 rounded-[calc(var(--radius)-0.25rem)] border border-border p-3" key={reply.id}>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-foreground">kind {reply.kind}</p>
                  <p className="line-clamp-2 text-sm text-muted-foreground">{reply.content || t("eventDetail.noTextContent")}</p>
                  <p className="text-xs text-muted-foreground">{shortenId(reply.pubkey, 16, 6)}</p>
                  {reply.tags.some((tag) => tag[0] === "image") ? (
                    <a className="text-xs text-primary underline underline-offset-2" href={reply.tags.find((tag) => tag[0] === "image")?.[1]} rel="noreferrer" target="_blank">
                      {t("eventDetail.showImage", "Exibir imagem")}
                    </a>
                  ) : null}
                </div>
                <Link className="text-sm text-primary underline underline-offset-2" params={{ eventId: reply.id }} to="/events/$eventId">
                  {t("eventDetail.openEvent", "Abrir evento")}
                </Link>
              </div>
            ))}
            {repliesQuery.hasNextPage ? (
              <Button onClick={() => void repliesQuery.fetchNextPage()} type="button" variant="outline">
                {repliesQuery.isFetchingNextPage ? t("labels.timeline.loadingMore", "Carregando...") : t("labels.timeline.loadMore", "Carregar mais")}
              </Button>
            ) : null}
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}

function CommunityMetadataCard({ dTag, description, image }: { dTag: string; description: string; image: string }) {
  return (
    <div className="rounded-[calc(var(--radius)-0.2rem)] border border-emerald-500/20 bg-emerald-500/5 p-4 space-y-3">
      {dTag ? <p className="text-sm font-semibold text-foreground">d: {dTag}</p> : null}
      {description ? <p className="whitespace-pre-wrap text-sm text-muted-foreground">{description}</p> : null}
      {image ? <img alt={dTag || "community image"} className="max-h-60 rounded border border-border object-cover" src={image} /> : null}
    </div>
  )
}

function ResponderCard({ pubkey, replyCount }: { pubkey: string; replyCount?: number }) {
  const { t } = useTranslation()
  const userQuery = useUser(pubkey)
  const user = userQuery.data

  return (
    <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
      {user ? <UserAvatarChip compact subtitle={replyCount ? t("eventDetail.replyCount", { count: replyCount, defaultValue: `${replyCount} resposta(s)` }) : undefined} user={user} /> : <p className="text-sm text-muted-foreground">{shortenId(pubkey, 16, 6)}</p>}
      <div className="mt-3 flex justify-end">
        <Link className="text-sm text-primary underline underline-offset-2" params={{ pubkey }} to="/users/$pubkey">
          {t("eventDetail.openUser", "Abrir usuário")}
        </Link>
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
