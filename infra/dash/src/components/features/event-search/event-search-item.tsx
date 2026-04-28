import { Link } from "@tanstack/react-router"
import { Copy, Eye, ExternalLink, Hash, Clock, User } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
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
    <div className="group relative rounded-md border border-border/60 bg-card/30 p-3 hover:border-primary/40 hover:bg-secondary/5 transition-all duration-200 animate-in fade-in slide-in-from-bottom-1 overflow-hidden">
      {/* Visual background indicator for kind */}
      <div className="absolute left-0 top-0 bottom-0 w-1 bg-primary/10 group-hover:bg-primary/40 transition-colors" />

      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between ml-2">
        <div className="flex min-w-0 flex-1 gap-3">
          <div className="flex size-7 shrink-0 items-center justify-center rounded-sm bg-secondary/80 text-[10px] font-mono font-bold text-muted-foreground group-hover:bg-primary group-hover:text-primary-foreground transition-all">
            {index + 1}
          </div>
          
          <div className="min-w-0 flex-1 space-y-2.5">
            <div className="flex items-center gap-2 flex-wrap">
              <Badge variant="muted" className="h-5 rounded-sm px-1.5 text-[9px] font-mono font-bold border-primary/20 bg-primary/5 text-primary">
                K:{eventItem.kind}
              </Badge>
              <h3 className="line-clamp-1 text-sm font-semibold tracking-tight text-foreground/90 leading-none">
                {eventHeadline(eventItem)}
              </h3>
            </div>

            <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[10px] text-muted-foreground font-mono">
              <div className="flex items-center gap-1.5 hover:text-primary transition-colors">
                <User className="size-3 opacity-60" />
                <Link
                  className="font-medium underline-offset-4 hover:underline"
                  params={{ pubkey: eventItem.pubkey }}
                  to="/users/$pubkey"
                >
                  {shortenId(eventItem.pubkey, 8, 4)}
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

            {/* Specialized Content Displays */}
            {eventItem.kind === 30003 && (
               <div className="inline-flex items-center px-2 py-0.5 rounded bg-blue-500/10 text-blue-500 text-[9px] font-bold uppercase border border-blue-500/20">
                 NIP-33 List Event
               </div>
            )}

            {eventItem.kind === 1010 && refs.length > 0 && (
              <div className="space-y-1 rounded border border-yellow-500/20 bg-yellow-500/5 p-2 text-[11px]">
                <p className="font-bold text-yellow-600 dark:text-yellow-400">Content Change</p>
                <div className="space-y-0.5">
                  {refs.map((ref) => (
                    <Link
                      className="block font-mono underline decoration-dotted underline-offset-4 opacity-80 hover:opacity-100 hover:text-primary transition-all truncate"
                      key={`cc-${ref.id}`}
                      params={{ eventId: ref.id }}
                      to="/events/$eventId"
                    >
                      {ref.id}
                    </Link>
                  ))}
                </div>
              </div>
            )}

            {eventItem.kind === 0 && profile && (
              <div className="rounded border border-primary/10 bg-primary/5 p-2 text-[11px] space-y-1">
                <div className="flex items-center justify-between">
                   <p className="font-bold text-foreground/80">Profile Update</p>
                   {profile.nip05 && <span className="text-[9px] px-1.5 py-0.5 bg-background border rounded-sm text-muted-foreground">{profile.nip05}</span>}
                </div>
                <p className="text-muted-foreground truncate">{profile.display_name || profile.name || "Unnamed user"}</p>
                {profile.about && <p className="line-clamp-2 text-muted-foreground opacity-70 italic">{profile.about}</p>}
              </div>
            )}

            <div className="flex flex-wrap gap-1.5 pt-0.5">
              {getEventTags(eventItem).slice(0, 12).map((tag) => (
                <Link key={`${eventItem.id}-${tag}`} search={{ tags: tag }} to="/events/search">
                  <Badge variant="muted" className="h-4.5 rounded-sm px-1.5 py-0 text-[10px] font-medium border-transparent bg-secondary/40 hover:bg-primary/20 hover:text-primary transition-all">
                    #{tag}
                  </Badge>
                </Link>
              ))}
              {getEventTags(eventItem).length > 12 && (
                <Badge variant="muted" className="h-4.5 text-[10px] opacity-40">
                  +{getEventTags(eventItem).length - 12}
                </Badge>
              )}
            </div>
          </div>
        </div>

        <div className="flex sm:flex-col shrink-0 gap-1 mt-2 sm:mt-0 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-all duration-300 transform translate-x-2 group-hover:translate-x-0">
          <TooltipProvider>
             <Tooltip>
                <TooltipTrigger asChild>
                   <Button onClick={onOpenJSON} size="icon" variant="ghost" className="size-7 hover:bg-primary/10 hover:text-primary">
                     <Eye className="size-3.5" />
                   </Button>
                </TooltipTrigger>
                <TooltipContent side="left">
                   <p className="text-[10px]">Inspect JSON</p>
                </TooltipContent>
             </Tooltip>

             <Tooltip>
                <TooltipTrigger asChild>
                   <Button asChild size="icon" variant="ghost" className="size-7 hover:bg-primary/10 hover:text-primary">
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
                   <Button onClick={copyEventId} size="icon" variant="ghost" className="size-7 hover:bg-primary/10 hover:text-primary">
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