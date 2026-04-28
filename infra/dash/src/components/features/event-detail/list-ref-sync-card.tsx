import { useMemo, useState } from "react"
import { Link } from "@tanstack/react-router"
import { RefreshCcw } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { useEventDetail, useFetchEventFromRelaysMutation } from "@/hooks/use-admin-data"
import { ApiError } from "@/services/admin"
import { collectMediaForEvent, isImageURL, isVideoURL, VIDEO_KINDS } from "@/lib/event-parser"
import { readRelayStorage } from "@/lib/relay-presets"
import { formatDateTime } from "@/lib/utils"
import { RelayResults, RelaySearchModal } from "./relay-search-modal"

interface ListRefSyncCardProps {
  eventID: string
  preferredRelay?: string
}

export type RelayResult = { relay: string; status: string; error?: string }

export function ListRefSyncCard({ eventID, preferredRelay }: ListRefSyncCardProps) {
  const { t } = useTranslation()
  const detailQuery = useEventDetail(eventID)
  const syncMutation = useFetchEventFromRelaysMutation()
  const [open, setOpen] = useState(false)
  const [status, setStatus] = useState("")
  const [relayFeedback, setRelayFeedback] = useState<RelayResult[]>([])
  const preferredRelays = useMemo(() => {
    const base = readRelayStorage("external-relays")
    if (preferredRelay && /^wss?:\/\//.test(preferredRelay) && !base.includes(preferredRelay)) {
      return [preferredRelay, ...base]
    }
    return base
  }, [preferredRelay])

  const referencedEvent = detailQuery.data?.event
  const referencedImages = detailQuery.data?.image_urls ?? []
  const referencedMedia = referencedEvent ? collectMediaForEvent(referencedEvent, referencedImages) : { images: [], videos: [] }
  const imageEvent = referencedEvent?.kind === 20
  const videoEvent = referencedEvent ? VIDEO_KINDS.includes(referencedEvent.kind) : false

  const handleSync = async (selectedRelays: string[]) => {
    try {
      const response = await syncMutation.mutateAsync({ eventID, relays: selectedRelays })
      setRelayFeedback(response.relay_results ?? [])
      if (!response.found) {
        setStatus(t("eventDetail.syncStatusError", { error: response.message || t("eventDetail.syncFailed") }))
        return
      }
      setStatus(
        t("eventDetail.syncStatusOk", {
          persisted: response.persisted ? t("eventDetail.imported") : t("eventDetail.alreadyExisted"),
          relay: response.source_relay || preferredRelay || t("eventDetail.defaultRelay"),
        })
      )
      await detailQuery.refetch()
      setOpen(false)
    } catch (error) {
      if (error instanceof ApiError && error.details && typeof error.details === "object") {
        const details = error.details as { relay_results?: RelayResult[] }
        setRelayFeedback(details.relay_results ?? [])
      }
      setStatus(t("eventDetail.syncStatusError", { error: error instanceof Error ? error.message : t("eventDetail.syncFailed") }))
    }
  }

  const getLocalStateLabel = () => {
    if (detailQuery.data) return t("eventDetail.present")
    if (detailQuery.isError) return t("eventDetail.absent")
    return t("eventDetail.checking")
  }

  return (
    <div className="rounded-md border border-border bg-card px-3 py-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="min-w-0">
          <Link
            className="break-all font-mono text-xs text-primary underline decoration-dotted underline-offset-2"
            params={{ eventId: eventID }}
            to="/events/$eventId"
          >
            {eventID}
          </Link>
          <p className="text-xs text-muted-foreground">
            {t("eventDetail.preferredRelay")}: {preferredRelay || t("eventDetail.defaultRelay")}
          </p>
          <p className="text-xs text-muted-foreground">
            {t("eventDetail.localState")}: {getLocalStateLabel()}
          </p>

          {referencedEvent && (
            <div className="mt-1 rounded border border-border bg-muted/20 px-2 py-1">
              <p className="text-xs text-foreground">
                {t("eventDetail.kindValue", { kind: referencedEvent.kind })} · {formatDateTime(referencedEvent.created_at)}
              </p>

              {imageEvent && (
                <div className="mt-2 space-y-2">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
                    {t("eventDetail.imageEvent")}
                  </p>
                  {referencedMedia.images.length > 0 ? (
                    <div className="grid gap-2 sm:grid-cols-2">
                      {referencedMedia.images.slice(0, 4).map((url) => (
                        <a href={url} key={url} rel="noreferrer" target="_blank">
                          <img
                            alt={t("eventDetail.referencedImageAlt")}
                            className="h-28 w-full rounded border border-border object-cover"
                            src={url}
                          />
                        </a>
                      ))}
                    </div>
                  ) : (
                    <p className="text-xs text-muted-foreground">{t("eventDetail.noImageDetected")}</p>
                  )}
                </div>
              )}

              {videoEvent && (
                <div className="mt-2 space-y-2">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
                    {t("eventDetail.videoEvent")}
                  </p>
                  {referencedMedia.videos[0] ? (
                    <video
                      className="max-h-56 w-full rounded border border-border bg-black"
                      controls
                      poster={referencedMedia.images[0] || undefined}
                      preload="metadata"
                      src={referencedMedia.videos[0]}
                    />
                  ) : (
                    <p className="text-xs text-muted-foreground">{t("eventDetail.noVideoDetected")}</p>
                  )}
                </div>
              )}

              {!imageEvent && !videoEvent && (
                <div className="mt-2 space-y-2">
                  <p className="line-clamp-4 max-w-full whitespace-pre-wrap break-all text-xs text-muted-foreground">
                    {referencedEvent.content || t("eventDetail.noTextContent")}
                  </p>

                  {referencedMedia.images.length > 0 && (
                    <div className="grid gap-2 sm:grid-cols-2">
                      {referencedMedia.images.slice(0, 2).map((url) => (
                        <a href={url} key={url} rel="noreferrer" target="_blank">
                          <img
                            alt={t("eventDetail.referencedMediaAlt")}
                            className="h-24 w-full rounded border border-border object-cover"
                            src={url}
                          />
                        </a>
                      ))}
                    </div>
                  )}

                  {referencedMedia.videos[0] && (
                    <video
                      className="max-h-56 w-full rounded border border-border bg-black"
                      controls
                      poster={referencedMedia.images[0] || undefined}
                      preload="metadata"
                      src={referencedMedia.videos[0]}
                    />
                  )}
                </div>
              )}
            </div>
          )}

          {status && <p className="text-xs text-muted-foreground">{status}</p>}
        </div>

        <div className="flex gap-2">
          <Button asChild size="sm" title={t("eventDetail.openReferencedEvent")} variant="outline">
            <Link params={{ eventId: eventID }} to="/events/$eventId">
              {t("eventDetail.open")}
            </Link>
          </Button>
          <Button onClick={() => setOpen(true)} size="sm" title={t("eventDetail.openCustomSearchModal")} variant="outline">
            <RefreshCcw className="size-4" />
            {t("state.retry")}
          </Button>
        </div>
      </div>

      <RelaySearchModal
        description={t("eventDetail.adjustRelaysSync")}
        onOpenChange={setOpen}
        onSearch={handleSync}
        open={open}
        relays={preferredRelays}
        title={t("eventDetail.findOnRelays")}
      />

      {relayFeedback.length > 0 ? <RelayResults results={relayFeedback} /> : null}
    </div>
  )
}
