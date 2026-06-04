import { Link } from "@tanstack/react-router"
import { Copy, Eye, ExternalLink, Hash, Clock, User, Shield, Repeat, Mail, Lock } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { EventSearchCommunityContext } from "@/components/features/event-search/event-search-community-context"
import { MediaCarousel } from "@/components/features/event-detail/media-carousel"
import { EventVideoPlayer } from "@/components/features/event-detail/event-video-player"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { useUser } from "@/hooks/use-admin-data"
import { useCommunityAddressEvent } from "@/hooks/use-community-address-event"
import {
  collectAltTexts,
  collectMediaAssets,
  communityDisplayNameFromAddress,
  communityNameFromTags,
  parseCommunityApproval,
  parseCommunityAddressTag,
  parseCommunityMetadata,
  parseDMRelays,
  parseEmbeddedEvent,
} from "@/lib/event-parser"
import { toNote } from "@/lib/nostr"
import { getNostrKindMeta } from "@/lib/nostr-kind-meta"
import { eventCommunityLabel, eventHeadline, parseEventRefs, parseProfileContent } from "@/lib/event-search"
import { formatDateTime, shortenId } from "@/lib/utils"
import { getEventTags } from "@/services/admin-event-detail"
import type { EventRecord } from "@/types/admin"

interface EventSearchItemProps {
  eventItem: EventRecord
  index: number
  onOpenJSON: () => void
}

export function EventSearchItem({ eventItem, index, onOpenJSON }: EventSearchItemProps) {
  const { t } = useTranslation()
  const refs = parseEventRefs(eventItem)
  const userQuery = useUser(eventItem.pubkey)
  const profile = eventItem.kind === 0 ? parseProfileContent(eventItem.content) : null
  const community = eventItem.kind === 34550 ? parseCommunityMetadata(eventItem.tags) : null
  const communityApproval = eventItem.kind === 4550 ? parseCommunityApproval(eventItem.tags) : null
  const embeddedApprovalEvent = eventItem.kind === 4550 ? parseEmbeddedEvent(eventItem.content) : null
  const dmRelays = eventItem.kind === 10050 ? parseDMRelays(eventItem.tags) : null
  const communityAddress = parseCommunityAddressTag(eventItem.tags)
  const communityQuery = useCommunityAddressEvent(communityAddress ? `${communityAddress.kind}:${communityAddress.pubkey}:${communityAddress.identifier}` : "")
  const altTexts = collectAltTexts(eventItem.tags)
  const mediaAssets = collectMediaAssets(eventItem.tags, eventItem.content)
  const singleAsset = mediaAssets.length === 1 ? mediaAssets[0] : null
  const multiMedia = mediaAssets.length > 1
  const communityLabel = communityQuery.data ? communityNameFromTags(communityQuery.data.tags) : eventCommunityLabel(eventItem)
  const communityImage = communityQuery.data ? parseCommunityMetadata(communityQuery.data.tags).image : ""
  const communityBadge = eventItem.kind === 1111 ? "Post da comunidade" : eventItem.kind === 4550 ? "Aprovacao da comunidade" : "Comunidade"
  const approvalAlt = embeddedApprovalEvent ? collectAltTexts(embeddedApprovalEvent.tags)[0] : altTexts[0]
  const approvalMedia = embeddedApprovalEvent ? collectMediaAssets(embeddedApprovalEvent.tags, embeddedApprovalEvent.content) : []
  const authorName = userQuery.data?.displayName || profile?.display_name || profile?.name || ""
  const topicTags = getEventTags(eventItem).filter((tag): tag is string => Boolean(tag))
  const kindMeta = getNostrKindMeta(eventItem.kind)
  const kindDescription = kindMeta ? `${kindMeta.description}${kindMeta.nip ? ` · ${kindMeta.nip}` : ""}` : undefined

  const copyEventId = async () => {
    await navigator.clipboard.writeText(eventItem.id)
    toast.success("Event ID copied.")
  }

  const copyEventReference = async (value: string) => {
    const encoded = /^[a-f0-9]{64}$/i.test(value) ? toNote(value) || value : value
    await navigator.clipboard.writeText(encoded)
    toast.success("Event reference copied.")
  }

  return (
    <div className="group relative overflow-hidden rounded-md border border-border/60 bg-card/30 p-3 transition-all duration-200 animate-in fade-in slide-in-from-bottom-1 hover:border-primary/40 hover:bg-secondary/5">
      <div className="absolute bottom-0 left-0 top-0 w-1 bg-primary/10 transition-colors group-hover:bg-primary/40" />

      <div className="ml-2 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex min-w-0 flex-1 gap-3">
          <div className="flex size-7 shrink-0 items-center justify-center rounded-sm bg-secondary/80 text-[10px] font-mono font-bold text-muted-foreground transition-all group-hover:bg-primary group-hover:text-primary-foreground">
            {index + 1}
          </div>

          <div className="min-w-0 flex-1 space-y-2.5">
            <div className="flex flex-wrap items-center gap-2">
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Badge variant="muted" className="h-5 rounded-sm border-primary/20 bg-primary/5 px-1.5 text-[9px] font-mono font-bold text-primary cursor-help">
                      K:{eventItem.kind}
                    </Badge>
                  </TooltipTrigger>
                  <TooltipContent className="max-w-xs text-[11px] leading-relaxed" side="top">
                    <p>{kindDescription || `kind ${eventItem.kind}`}</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
              <h3 className="line-clamp-1 text-sm font-semibold leading-none tracking-tight text-foreground/90">
                {eventHeadline(eventItem)}
              </h3>
            </div>

            <div className="flex flex-wrap items-center gap-x-3 gap-y-1 font-mono text-[10px] text-muted-foreground">
              <div className="flex items-center gap-1.5 transition-colors hover:text-primary">
                <User className="size-3 opacity-60" />
                <Link className="font-medium underline-offset-4 hover:underline" params={{ pubkey: eventItem.pubkey }} to="/users/$pubkey">
                  {authorName || shortenId(eventItem.pubkey, 8, 4)}
                </Link>
              </div>
              <div className="flex items-center gap-1.5">
                <Clock className="size-3 opacity-60" />
                <span>{formatDateTime(eventItem.created_at)}</span>
              </div>
              <div className="flex items-center gap-1.5 opacity-60">
                <Hash className="size-3" />
                <span>{shortenId(eventItem.id, 8, 4)}</span>
              </div>
            </div>

            {communityLabel ? <EventSearchCommunityContext badge={communityBadge} imageUrl={communityImage || undefined} label={communityLabel} /> : null}

            {eventItem.kind === 1111 && communityLabel ? (
              <div className="space-y-2 rounded border border-emerald-500/15 bg-background/60 p-3 text-[11px]">
                {eventItem.content ? <p className="line-clamp-5 whitespace-pre-wrap break-words text-sm text-foreground">{eventItem.content}</p> : null}
                {topicTags.length > 0 ? (
                  <div className="flex flex-wrap gap-1.5">
                    {topicTags.slice(0, 8).map((tag) => (
                      <Badge className="h-5 rounded-sm border-emerald-500/20 bg-emerald-500/8 px-1.5 py-0 text-[10px] text-emerald-700 dark:text-emerald-300" key={`${eventItem.id}-topic-${tag}`} variant="muted">
                        #{tag}
                      </Badge>
                    ))}
                  </div>
                ) : null}
              </div>
            ) : null}

            {eventItem.kind === 30003 && (
              <div className="inline-flex items-center rounded border border-blue-500/20 bg-blue-500/10 px-2 py-0.5 text-[9px] font-bold uppercase text-blue-500">
                NIP-33 List Event
              </div>
            )}

            {eventItem.kind === 0 && profile && (
              <div className="space-y-1 rounded border border-primary/10 bg-primary/5 p-2 text-[11px]">
                <div className="flex items-center justify-between gap-2">
                  <p className="font-bold text-foreground/80">Profile Update</p>
                  {profile.nip05 ? <span className="rounded-sm border bg-background px-1.5 py-0.5 text-[9px] text-muted-foreground">{profile.nip05}</span> : null}
                </div>
                <p className="truncate text-muted-foreground">{profile.display_name || profile.name || "Unnamed user"}</p>
                {profile.about ? <p className="line-clamp-2 italic text-muted-foreground opacity-70">{profile.about}</p> : null}
              </div>
            )}

            {eventItem.kind === 34550 && community && (
              <div className="space-y-2 rounded border border-emerald-500/20 bg-emerald-500/5 p-2 text-[11px]">
                <div className="flex items-center justify-between gap-2">
                  <p className="font-bold text-foreground/80">Community</p>
                  {community.d ? <span className="rounded-sm border bg-background px-1.5 py-0.5 text-[9px] text-muted-foreground">d: {community.d}</span> : null}
                </div>
                {community.description ? <p className="line-clamp-3 text-muted-foreground">{community.description}</p> : null}
                {community.image ? (
                  <div className="flex items-center gap-2">
                    <img alt={community.d || "community image"} className="size-14 rounded border border-border/60 object-cover" src={community.image} />
                    <a className="text-xs text-primary underline underline-offset-2" href={community.image} rel="noreferrer" target="_blank">Abrir imagem</a>
                  </div>
                ) : null}
              </div>
            )}

            {eventItem.kind === 4550 && communityApproval && (
              <div className="min-w-0 space-y-2 overflow-hidden rounded border border-primary/20 bg-primary/5 p-2 text-[11px]">
                <div className="flex items-center gap-2">
                  <Shield className="size-3 text-primary" />
                  <p className="font-bold text-foreground/80">Community Approval</p>
                </div>
                <div className="flex flex-col gap-2">
                  <div className="flex min-w-0 flex-wrap items-center gap-1">
                    <span className="text-muted-foreground">Comunidade:</span>
                    <span className="min-w-0 break-all font-medium text-foreground">{communityLabel || communityDisplayNameFromAddress(communityApproval.communityRef)}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <span className="text-muted-foreground">Post:</span>
                    <Link className="font-mono text-primary underline-offset-2 hover:underline" params={{ eventId: communityApproval.approvedEventId }} to="/events/$eventId">
                      {shortenId(communityApproval.approvedEventId, 8, 4)}
                    </Link>
                  </div>
                  {communityApproval.approvedKind !== undefined ? <span className="text-muted-foreground">Kind: {communityApproval.approvedKind}</span> : null}
                  {communityApproval.postAuthor ? <span className="break-all text-muted-foreground">Autor do post: {shortenId(communityApproval.postAuthor, 12, 4)}</span> : null}
                  {embeddedApprovalEvent?.content ? (
                    <div className="mt-1 line-clamp-3 border-t border-primary/10 pt-2 italic text-foreground/80 opacity-80">
                      "{embeddedApprovalEvent.content}"
                    </div>
                  ) : null}
                  {!embeddedApprovalEvent?.content && approvalAlt ? (
                    <span className="break-words text-muted-foreground">Alt: {approvalAlt}</span>
                  ) : null}
                  {approvalMedia[0]?.type === "image" ? (
                    <div className="max-w-sm overflow-hidden rounded border border-primary/10">
                      <img alt={approvalMedia[0].alt || approvalAlt || "Approved event media"} className="h-28 w-full object-cover" loading="lazy" src={approvalMedia[0].url} />
                    </div>
                  ) : null}
                  {embeddedApprovalEvent ? <div className="rounded border border-primary/10 bg-background/60 p-3 text-xs text-muted-foreground">Evento aprovado disponível no detalhe completo.</div> : null}
                </div>
              </div>
            )}

            {eventItem.kind === 6 && refs[0] && (
              <div className="space-y-2 rounded border border-blue-500/20 bg-blue-500/5 p-2 text-[11px]">
                <div className="flex items-center gap-2">
                  <Repeat className="size-3 text-blue-500" />
                  <p className="font-bold text-foreground/80">Repost</p>
                </div>
                <div className="flex items-center gap-1">
                  <span className="text-muted-foreground">Post original:</span>
                  <Link className="truncate font-mono text-primary underline-offset-2 hover:underline" params={{ eventId: refs[0].id }} to="/events/$eventId">
                    {shortenId(refs[0].id, 12, 4)}
                  </Link>
                </div>
              </div>
            )}

            {eventItem.kind === 10050 && dmRelays && (
              <div className="space-y-2 rounded border border-purple-500/20 bg-purple-500/5 p-2 text-[11px]">
                <div className="flex items-center gap-2">
                  <Mail className="size-3 text-purple-500" />
                  <p className="font-bold text-foreground/80">DM Relays</p>
                  <Lock className="ml-auto size-3 text-muted-foreground opacity-50" />
                </div>
                <div className="flex flex-wrap gap-1">
                  {dmRelays.map((relay) => (
                    <Badge className="border-purple-500/30 bg-background text-[9px] text-purple-600/80" key={relay} variant="muted">
                      {relay}
                    </Badge>
                  ))}
                </div>
              </div>
            )}

            {singleAsset?.type === "image" ? (
              <div className="mt-2 max-w-sm overflow-hidden rounded border border-border/60">
                <img alt={singleAsset.alt || altTexts[0] || "Event content"} className="h-auto max-h-48 w-full object-cover" loading="lazy" src={singleAsset.url} />
              </div>
            ) : null}

            {singleAsset?.type === "video" ? (
              <div className="mt-2 max-w-sm overflow-hidden rounded border border-border/60 bg-black">
                <EventVideoPlayer className="h-32 w-full rounded-none border-0 object-contain" lazy showChrome={false} url={singleAsset.url} />
              </div>
            ) : null}

            {multiMedia ? (
              <div className="mt-2 max-w-sm overflow-hidden rounded border border-border/60 bg-black">
                <MediaCarousel
                  className="rounded-none border-0"
                  lazyVideo
                  media={mediaAssets.map((asset) => ({ type: asset.type, url: asset.url, alt: asset.alt || altTexts[0] }))}
                  mediaClassName="h-48 max-h-48"
                  viewportClassName="max-h-48"
                />
              </div>
            ) : null}

              <div className="flex flex-wrap gap-1.5 pt-0.5">
              {topicTags.slice(0, 12).map((tag) => (
                /^[a-f0-9]{64}$/i.test(tag) ? (
                  <button key={`${eventItem.id}-${tag}`} onClick={() => void copyEventReference(tag)} type="button">
                    <Badge className="h-4.5 rounded-sm border-transparent bg-secondary/40 px-1.5 py-0 text-[10px] font-medium transition-all hover:bg-primary/20 hover:text-primary" variant="muted">
                      #{shortenId(tag, 8, 4)}
                    </Badge>
                  </button>
                ) : (
                  <Link key={`${eventItem.id}-${tag}`} search={{ tags: tag }} to="/events/search">
                    <Badge className="h-4.5 rounded-sm border-transparent bg-secondary/40 px-1.5 py-0 text-[10px] font-medium transition-all hover:bg-primary/20 hover:text-primary" variant="muted">
                      #{tag}
                    </Badge>
                  </Link>
                )
              ))}
              {topicTags.length > 12 ? <Badge className="h-4.5 text-[10px] opacity-40" variant="muted">+{topicTags.length - 12}</Badge> : null}
            </div>
          </div>
        </div>

        <div className="mt-2 flex shrink-0 gap-1 opacity-0 transition-all duration-300 focus-within:opacity-100 group-hover:translate-x-0 group-hover:opacity-100 sm:mt-0 sm:flex-col translate-x-2">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button className="size-7 hover:bg-primary/10 hover:text-primary" onClick={onOpenJSON} size="icon" variant="ghost">
                  <Eye className="size-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="left">
                <p className="text-[10px]">Inspect JSON</p>
              </TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button asChild className="size-7 hover:bg-primary/10 hover:text-primary" size="icon" variant="ghost">
                  <Link params={{ eventId: eventItem.id }} to="/events/$eventId">
                    <ExternalLink className="size-3.5" />
                  </Link>
                </Button>
              </TooltipTrigger>
              <TooltipContent side="left">
                <p className="text-[10px]">View Detail</p>
              </TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button className="size-7 hover:bg-primary/10 hover:text-primary" onClick={copyEventId} size="icon" variant="ghost">
                  <Copy className="size-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="left">
                <p className="text-[10px]">Copy ID</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      </div>
    </div>
  )
}
