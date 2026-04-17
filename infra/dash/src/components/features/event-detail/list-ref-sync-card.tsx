import { useState } from "react"
import { Link } from "@tanstack/react-router"
import { Plus, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { useEventDetail, useFetchEventFromRelaysMutation } from "@/hooks/use-admin-data"
import { ApiError } from "@/services/admin"
import { collectMediaForEvent, isImageURL, isVideoURL, VIDEO_KINDS } from "@/lib/event-parser"
import { formatDateTime } from "@/lib/utils"

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
  const [relayInput, setRelayInput] = useState("")
  const [selectedRelays, setSelectedRelays] = useState<string[]>(() => {
    if (preferredRelay && /^wss?:\/\//.test(preferredRelay)) {
      return [preferredRelay, ...commonRelays.filter((relay) => relay !== preferredRelay)]
    }
    return [...commonRelays]
  })
  const [relayFeedback, setRelayFeedback] = useState<RelayResult[]>([])

  const referencedEvent = detailQuery.data?.event
  const referencedImages = detailQuery.data?.image_urls ?? []
  const referencedMedia = referencedEvent ? collectMediaForEvent(referencedEvent, referencedImages) : { images: [], videos: [] }
  const imageEvent = referencedEvent?.kind === 20
  const videoEvent = referencedEvent ? VIDEO_KINDS.includes(referencedEvent.kind) : false

  const addRelay = () => {
    const value = relayInput.trim()
    if (!value) {
      return
    }
    if (!/^wss?:\/\//.test(value)) {
      toast.error(t("eventDetail.useWsUrl"))
      return
    }
    setSelectedRelays((current) => (current.includes(value) ? current : [...current, value]))
    setRelayInput("")
  }

  const toggleRelay = (relay: string) => {
    setSelectedRelays((current) =>
      current.includes(relay) ? current.filter((item) => item !== relay) : [...current, relay]
    )
  }

  const handleSync = async () => {
    try {
      const response = await syncMutation.mutateAsync({ eventID, relays: selectedRelays })
      setRelayFeedback(response.relay_results ?? [])
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
            {t("state.retry")}
          </Button>
        </div>
      </div>

      <Dialog onOpenChange={setOpen} open={open}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>{t("eventDetail.findOnRelays")}</DialogTitle>
            <DialogDescription>{t("eventDetail.adjustRelaysSync")}</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                {t("eventDetail.commonRelays")}
              </p>
              <div className="flex flex-wrap gap-2">
                {commonRelays.map((relay) => (
                  <Button
                    key={`${eventID}-${relay}`}
                    onClick={() => toggleRelay(relay)}
                    size="sm"
                    title={t("eventDetail.toggleRelay")}
                    type="button"
                    variant={selectedRelays.includes(relay) ? "default" : "outline"}
                  >
                    {relay}
                  </Button>
                ))}
              </div>
            </div>

            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("eventDetail.addRelay")}</p>
              <div className="flex gap-2">
                <Input onChange={(event) => setRelayInput(event.target.value)} placeholder={t("eventDetail.relayPlaceholder")} value={relayInput} />
                <Button onClick={addRelay} title={t("eventDetail.addTypedRelay")} type="button" variant="outline">
                  <Plus className="size-4" />
                  {t("eventDetail.include")}
                </Button>
              </div>
            </div>

            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("eventDetail.searchList")}</p>
              <div className="flex max-h-40 flex-wrap gap-2 overflow-auto rounded-md border border-border p-2">
                {selectedRelays.map((relay) => (
                  <Badge className="flex items-center gap-1" key={`${eventID}-selected-${relay}`} variant="muted">
                    <span className="max-w-[220px] truncate">{relay}</span>
                    <button
                      className="cursor-pointer"
                      onClick={() => setSelectedRelays((current) => current.filter((item) => item !== relay))}
                      title={t("eventDetail.removeRelay")}
                      type="button"
                    >
                      <X className="size-3" />
                    </button>
                  </Badge>
                ))}
                {selectedRelays.length === 0 && (
                  <p className="text-xs text-muted-foreground">{t("eventDetail.noRelaySelected")}</p>
                )}
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button onClick={() => setOpen(false)} title={t("eventDetail.closeWithoutSearch")} type="button" variant="outline">
              {t("common.cancel")}
            </Button>
            <Button
              disabled={syncMutation.isPending || selectedRelays.length === 0}
              title={t("eventDetail.searchImportThisEvent")}
              onClick={handleSync}
              type="button"
            >
              {syncMutation.isPending ? t("eventDetail.searching") : t("eventDetail.searchEvent")}
            </Button>
          </DialogFooter>

          {relayFeedback.length > 0 && (
            <div className="space-y-2 rounded-md border border-border bg-muted/30 p-3">
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("eventDetail.resultByRelay")}</p>
              <div className="max-h-40 space-y-2 overflow-auto">
                {relayFeedback.map((entry) => (
                  <div className="flex items-center justify-between gap-2 text-xs" key={`${eventID}-${entry.relay}-${entry.status}`}>
                    <span className="truncate text-muted-foreground">{entry.relay}</span>
                    <Badge variant={entry.status === "found" ? "success" : "muted"}>
                      {t(`eventDetail.relayStatus.${entry.status}`, { defaultValue: entry.status })}
                    </Badge>
                  </div>
                ))}
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}

export const commonRelays: string[] = [
  "wss://relay.damus.io",
  "wss://relay.primal.net",
  "wss://nos.lol",
  "wss://relay.nostr.band",
  "wss://nostr.mom",
]