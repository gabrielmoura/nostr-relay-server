import { useTranslation } from "react-i18next"
import { Link } from "@tanstack/react-router"

import { Badge } from "@/components/ui/badge"

interface TagBadgeListProps {
  label: string
  tags: (string | number)[]
}

export function TagBadgeList({ label, tags }: TagBadgeListProps) {
  if (tags.length === 0) return null

  return (
    <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
      <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <div className="mt-2 flex flex-wrap gap-2">
        {tags.map((value) => <Badge key={String(value)} variant="muted">{String(value)}</Badge>)}
      </div>
    </div>
  )
}

interface EventRefListProps {
  label: string
  refs: string[]
  type: "event" | "user"
}

export function EventRefList({ label, refs, type }: EventRefListProps) {
  if (refs.length === 0) return null

  return (
    <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
      <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <div className="mt-2 space-y-1">
        {refs.map((value) => (
          <Link
            className="block break-all text-xs text-primary underline decoration-dotted underline-offset-2"
            key={value}
            params={type === "event" ? { eventId: value } : { pubkey: value }}
            to={type === "event" ? "/events/$eventId" : "/users/$pubkey"}
          >
            {value}
          </Link>
        ))}
      </div>
    </div>
  )
}

interface AddressRefListProps {
  label: string
  refs: string[]
}

export function AddressRefList({ label, refs }: AddressRefListProps) {
  if (refs.length === 0) return null

  return (
    <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
      <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <div className="mt-2 space-y-1">
        {refs.map((value) => <p className="break-all font-mono text-xs text-foreground" key={value}>{value}</p>)}
      </div>
    </div>
  )
}

interface LinksAndQuotesProps {
  qRefs: string[]
  rRefs: string[]
}

export function LinksAndQuotes({ qRefs, rRefs }: LinksAndQuotesProps) {
  const { t } = useTranslation()
  
  if (qRefs.length === 0 && rRefs.length === 0) return null

  return (
    <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
      <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{"eventDetail.linksAndQuotes"}</p>
      <div className="mt-2 space-y-1">
        {qRefs.map((value) => <p className="break-all font-mono text-xs text-foreground" key={`q-${value}`}>{t("eventDetail.qRefPrefix")} {value}</p>)}
        {rRefs.map((value) => (
          <a className="block break-all text-xs text-primary underline decoration-dotted underline-offset-2" href={value} key={`r-${value}`} rel="noreferrer" target="_blank">
            {t("eventDetail.rRefPrefix")} {value}
          </a>
        ))}
      </div>
    </div>
  )
}