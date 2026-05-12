import { Shield } from "lucide-react"
import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"
import { useReferencedNostrEvent } from "@/hooks/use-referenced-nostr-event"
import { collectAltTexts, type EmbeddedRepost } from "@/lib/event-parser"
import { shortenId } from "@/lib/utils"

interface CommunityApprovalCardProps {
  communityRef: string
  approvedEventId: string
  approvedKind?: number
  postAuthor: string
  approvedEvent?: EmbeddedRepost | null
}

export function CommunityApprovalCard({ communityRef, approvedEventId, approvedKind, postAuthor, approvedEvent }: CommunityApprovalCardProps) {
  const { t } = useTranslation()
  const referencedEvent = useReferencedNostrEvent(approvedEventId)
  const resolvedEvent = approvedEvent ?? referencedEvent.data ?? null
  const previewAlt = collectAltTexts(resolvedEvent?.tags ?? [])[0]
  const previewContent = resolvedEvent?.content || ""

  return (
    <div className="space-y-4 rounded-md border border-primary/20 bg-primary/5 p-4 min-w-0 overflow-hidden">
      <div className="flex items-center gap-2">
        <Shield className="size-5 text-primary" />
        <h3 className="font-semibold text-foreground/90">Aprovação de Comunidade</h3>
      </div>
      
      <div className="grid min-w-0 gap-2 text-sm">
        <div className="flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-2">
          <span className="text-muted-foreground font-medium">Comunidade:</span>
          <span className="break-all font-mono text-foreground">{communityRef}</span>
        </div>
        
        <div className="flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-2">
          <span className="text-muted-foreground font-medium">Post Aprovado:</span>
            <Link 
              className="break-all font-mono text-primary underline decoration-dotted underline-offset-2 transition-colors hover:text-primary/80" 
              params={{ eventId: approvedEventId }} 
              to="/events/$eventId"
            >
            {approvedEventId}
          </Link>
        </div>
        
        {approvedKind !== undefined && (
          <div className="flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-2">
            <span className="text-muted-foreground font-medium">Kind do Post:</span>
            <span className="inline-flex items-center rounded-sm bg-primary/10 px-2 py-0.5 text-xs font-mono text-primary">
              {approvedKind}
            </span>
          </div>
        )}

        {postAuthor && (
          <div className="flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-2">
            <span className="text-muted-foreground font-medium">Autor do Post:</span>
            <Link 
              className="break-all font-mono text-primary underline decoration-dotted underline-offset-2 transition-colors hover:text-primary/80" 
              params={{ pubkey: postAuthor }} 
              to="/users/$pubkey"
            >
              {shortenId(postAuthor, 12, 12)}
            </Link>
          </div>
        )}

        {previewContent ? (
          <div className="rounded-md border border-primary/10 bg-background/60 p-3">
            <p className="text-xs uppercase tracking-[0.16em] text-muted-foreground">Evento aprovado</p>
            <p className="mt-2 break-words text-sm text-foreground">{previewContent}</p>
          </div>
        ) : previewAlt ? (
          <div className="rounded-md border border-primary/10 bg-background/60 p-3">
            <p className="text-xs uppercase tracking-[0.16em] text-muted-foreground">Alt</p>
            <p className="mt-2 break-words text-sm text-foreground">{previewAlt}</p>
          </div>
        ) : null}
      </div>
    </div>
  )
}
