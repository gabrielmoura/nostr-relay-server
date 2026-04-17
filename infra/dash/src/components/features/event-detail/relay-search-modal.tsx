import { useState } from "react"
import { Plus, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { commonRelays, type RelayResult } from "./list-ref-sync-card"

interface RelaySearchModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSearch: (relays: string[]) => Promise<void>
  title: string
  description: string
}

export function RelaySearchModal({ open, onOpenChange, onSearch, title, description }: RelaySearchModalProps) {
  const { t } = useTranslation()
  const [relayInput, setRelayInput] = useState("")
  const [selectedRelays, setSelectedRelays] = useState<string[]>([...commonRelays])

  const addRelay = () => {
    const value = relayInput.trim()
    if (!value) return
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

  const handleSearch = async () => {
    await onSearch(selectedRelays)
  }

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">
              {t("eventDetail.commonRelays")}
            </p>
            <div className="flex flex-wrap gap-2">
              {commonRelays.map((relay) => (
                <Button
                  key={relay}
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
              <Input
                onChange={(event) => setRelayInput(event.target.value)}
                placeholder={t("eventDetail.relayPlaceholder")}
                value={relayInput}
              />
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
                <Badge className="flex items-center gap-1" key={relay} variant="muted">
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
          <Button onClick={() => onOpenChange(false)} title={t("eventDetail.closeWithoutSearch")} type="button" variant="outline">
            {t("common.cancel")}
          </Button>
          <Button
            disabled={selectedRelays.length === 0}
            onClick={handleSearch}
            title={t("eventDetail.searchAndImport")}
            type="button"
          >
            {t("eventDetail.searchEvent")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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