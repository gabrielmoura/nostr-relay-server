import { useEffect, useState } from "react"
import { Plus, Save, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { commonRelayPresets, normalizeRelayList, parseRelayCSV, readRelayStorage, writeRelayStorage } from "@/lib/relay-presets"

type RelayListModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  storageScope: string
  title: string
  description: string
  value: string[]
  onApply: (relays: string[]) => void
  submitLabel?: string
}

export function RelayListModal({
  open,
  onOpenChange,
  storageScope,
  title,
  description,
  value,
  onApply,
  submitLabel,
}: RelayListModalProps) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState<string[]>(value)
  const [relayInput, setRelayInput] = useState("")
  const [bulkInput, setBulkInput] = useState("")

  useEffect(() => {
    if (!open) {
      return
    }
    const preferred = value.length > 0 ? value : readRelayStorage(storageScope)
    setDraft(normalizeRelayList(preferred))
  }, [open, storageScope, value])

  const addRelay = () => {
    const next = normalizeRelayList([...draft, relayInput])
    if (next.length === draft.length) {
      toast.error(t("relayPicker.invalidRelay", "Informe uma URL ws:// ou wss:// válida"))
      return
    }
    setDraft(next)
    setRelayInput("")
  }

  const importBulkRelays = () => {
    const imported = parseRelayCSV(bulkInput)
    if (imported.length === 0) {
      toast.error(t("relayPicker.invalidBulk", "Nenhum relay válido foi encontrado"))
      return
    }
    setDraft(normalizeRelayList([...draft, ...imported]))
    setBulkInput("")
  }

  const toggleCommonRelay = (relay: string) => {
    setDraft((current) => (current.includes(relay) ? current.filter((item) => item !== relay) : [...current, relay]))
  }

  const save = () => {
    const normalized = normalizeRelayList(draft)
    writeRelayStorage(storageScope, normalized)
    onApply(normalized)
    onOpenChange(false)
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
            <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("relayPicker.commonRelays", "Relays comuns")}</p>
            <div className="flex flex-wrap gap-2">
              {commonRelayPresets.map((relay) => (
                <Button key={relay} onClick={() => toggleCommonRelay(relay)} size="sm" type="button" variant={draft.includes(relay) ? "default" : "outline"}>
                  {relay}
                </Button>
              ))}
            </div>
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("relayPicker.addOne", "Adicionar relay")}</p>
              <div className="flex gap-2">
                <Input onChange={(event) => setRelayInput(event.target.value)} placeholder="wss://relay.damus.io" value={relayInput} />
                <Button onClick={addRelay} type="button" variant="outline">
                  <Plus className="size-4" />
                  {t("relayPicker.add", "Adicionar")}
                </Button>
              </div>
            </div>

            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("relayPicker.addMany", "Importar lista")}</p>
              <div className="space-y-2">
                <Textarea onChange={(event) => setBulkInput(event.target.value)} placeholder="wss://relay.damus.io, wss://nos.lol" rows={4} value={bulkInput} />
                <Button onClick={importBulkRelays} type="button" variant="outline">
                  <Plus className="size-4" />
                  {t("relayPicker.import", "Importar relays")}
                </Button>
              </div>
            </div>
          </div>

          <div className="space-y-2">
            <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("relayPicker.selected", "Relays selecionados")}</p>
            <div className="flex max-h-48 flex-wrap gap-2 overflow-auto rounded-md border border-border p-3">
              {draft.map((relay) => (
                <Badge className="flex items-center gap-1" key={relay} variant="muted">
                  <span className="max-w-[320px] truncate">{relay}</span>
                  <button className="cursor-pointer" onClick={() => setDraft((current) => current.filter((item) => item !== relay))} type="button">
                    <X className="size-3" />
                  </button>
                </Badge>
              ))}
              {draft.length === 0 ? <p className="text-xs text-muted-foreground">{t("relayPicker.empty", "Nenhum relay selecionado")}</p> : null}
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button onClick={() => onOpenChange(false)} type="button" variant="outline">
            {t("common.cancel")}
          </Button>
          <Button disabled={draft.length === 0} onClick={save} type="button">
            <Save className="size-4" />
            {submitLabel || t("relayPicker.save", "Salvar relays")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
