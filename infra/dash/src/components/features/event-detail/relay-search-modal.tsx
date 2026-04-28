import { useTranslation } from "react-i18next"

import { RelayListModal } from "@/components/shared/relay-list-modal"
import { Badge } from "@/components/ui/badge"

export type RelayResult = { relay: string; status: string; error?: string }

interface RelaySearchModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSearch: (relays: string[]) => Promise<void>
  title: string
  description: string
  relays: string[]
}

export function RelaySearchModal({ open, onOpenChange, onSearch, title, description, relays }: RelaySearchModalProps) {
  const { t } = useTranslation()

  return (
    <RelayListModal
      description={description}
      onApply={(selected) => void onSearch(selected)}
      onOpenChange={onOpenChange}
      open={open}
      storageScope="external-relays"
      submitLabel={t("eventDetail.searchEvent")}
      title={title}
      value={relays}
    />
  )
}

interface RelayResultsProps {
  results: RelayResult[]
}

export function RelayResults({ results }: RelayResultsProps) {
  const { t } = useTranslation()

  if (results.length === 0) return null

  return (
    <div className="space-y-2 rounded-md border border-border bg-muted/30 p-3">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("eventDetail.resultByRelay")}</p>
      <div className="max-h-40 space-y-2 overflow-auto">
        {results.map((entry) => (
          <div className="flex items-center justify-between gap-2 text-xs" key={`${entry.relay}-${entry.status}`}>
            <span className="truncate text-muted-foreground">{entry.relay}</span>
            <Badge variant={entry.status === "found" ? "success" : "muted"}>
              {t(`eventDetail.relayStatus.${entry.status}`, { defaultValue: entry.status })}
            </Badge>
          </div>
        ))}
      </div>
    </div>
  )
}
