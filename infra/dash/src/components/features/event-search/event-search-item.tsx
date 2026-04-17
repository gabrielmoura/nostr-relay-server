import { Link } from "@tanstack/react-router"
import { Copy, Eye } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { getEventTags } from "@/services/admin"
import { formatDateTime, shortenId } from "@/lib/utils"
import { parseEventRefs, parseServers, parseProfileContent, eventHeadline, type EventRef } from "@/lib/event-search"
import type { EventRecord } from "@/types/admin"

interface EventSearchItemProps {
  eventItem: EventRecord
  index: number
  onOpenJSON: () => void
}

export function EventSearchItem({ eventItem, index, onOpenJSON }: EventSearchItemProps) {
  const { t } = useTranslation()
  const refs = parseEventRefs(eventItem)
  const servers = parseServers(eventItem)
  const profile = eventItem.kind === 0 ? parseProfileContent(eventItem.content) : null

  const copyEventId = async () => {
    await navigator.clipboard.writeText(eventItem.id)
    toast.success("Event ID copied.")
  }

  return (
    <div className="rounded-[calc(var(--radius)-0.15rem)] border border-border bg-card p-4">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="flex min-w-0 flex-1 gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-md bg-destructive text-sm font-heading font-semibold text-white">
            {index + 1}
          </div>
          <div className="min-w-0 space-y-3">
            <div>
              <p className="text-sm font-semibold text-foreground">
                [kind:{eventItem.kind}] {eventHeadline(eventItem)}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                <Link
                  className="font-medium text-foreground underline decoration-dotted underline-offset-2 hover:text-primary"
                  params={{ pubkey: eventItem.pubkey }}
                  to="/users/$pubkey"
                >
                  Author: {shortenId(eventItem.pubkey, 12, 4)}
                </Link>
                {" · "}
                {formatDateTime(eventItem.created_at)}
                {" · event_id: "}
                {shortenId(eventItem.id, 10, 4)}
              </p>
            </div>

            {eventItem.kind === 30003 && <Badge variant="muted">List event (kind 30003)</Badge>}

            {eventItem.kind === 1010 && refs.length > 0 && (
              <div className="space-y-1 rounded-md border border-warning/40 bg-warning/5 p-3 text-xs">
                <p className="font-semibold text-foreground">Content Change Event</p>
                <p className="text-muted-foreground">This event updates content of:</p>
                {refs.map((ref) => (
                  <Link
                    className="block font-mono underline decoration-dotted underline-offset-2"
                    key={`cc-${ref.id}`}
                    params={{ eventId: ref.id }}
                    to="/events/$eventId"
                  >
                    {ref.id}
                  </Link>
                ))}
              </div>
            )}

            {eventItem.kind === 10063 && servers.length > 0 && (
              <div className="space-y-2 rounded-md border border-border bg-muted/20 p-3">
                <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Blossom Servers</p>
                <div className="space-y-1">
                  {servers.map((server) => (
                    <a
                      className="block break-all text-xs text-primary underline decoration-dotted underline-offset-2"
                      href={server}
                      key={server}
                      rel="noreferrer"
                      target="_blank"
                    >
                      {server}
                    </a>
                  ))}
                </div>
              </div>
            )}

            {eventItem.kind === 0 && profile && (
              <div className="rounded-md border border-border bg-muted/20 p-3 text-xs">
                <p className="font-semibold text-foreground">Profile event</p>
                <p className="text-muted-foreground">Name: {profile.display_name || profile.name || "-"}</p>
                {profile.nip05 ? <p className="text-muted-foreground">NIP-05: {profile.nip05}</p> : null}
                {profile.about ? <p className="mt-1 line-clamp-2 text-muted-foreground">{profile.about}</p> : null}
              </div>
            )}

            <div className="flex flex-wrap gap-2">
              {getEventTags(eventItem).map((tag) => (
                <Link key={`${eventItem.id}-${tag}`} search={{ tags: tag }} to="/events/search">
                  <Badge variant="muted">#{tag}</Badge>
                </Link>
              ))}
            </div>
          </div>
        </div>
        <div className="flex shrink-0 flex-wrap gap-2">
          <Button onClick={onOpenJSON} size="sm" variant="outline">
            <Eye className="size-4" />
            View JSON
          </Button>
          <Button asChild size="sm" variant="outline">
            <Link params={{ eventId: eventItem.id }} to="/events/$eventId">
              View Event
            </Link>
          </Button>
          <Button onClick={copyEventId} size="sm" variant="outline">
            <Copy className="size-4" />
            Copy ID
          </Button>
        </div>
      </div>
    </div>
  )
}