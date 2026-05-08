import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { LabelCategoryPicker } from "@/components/features/labels/label-category-picker"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { useBanMutation, useCreateLabelMutation } from "@/hooks/use-admin-data"
import { labelNamespaces, labelPresets, labelTargetOptions, normalizeLabelValue } from "@/lib/labels"
import { normalizeNip19TargetInput } from "@/lib/nostr"
import { ApiError } from "@/services/admin"
import type { AdminLabelTarget, AdminLabelTargetType } from "@/types/admin"

const customNamespaceValue = "__custom__"

type CreateLabelDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CreateLabelDialog({ open, onOpenChange }: CreateLabelDialogProps) {
  const { t } = useTranslation()
  const createMutation = useCreateLabelMutation()
  const banMutation = useBanMutation()

  const [namespaceMode, setNamespaceMode] = useState<string>(labelNamespaces[0] ?? "ugc")
  const [customNamespace, setCustomNamespace] = useState("")
  const [targetType, setTargetType] = useState<AdminLabelTargetType>("event")
  const [targetValue, setTargetValue] = useState("")
  const [relayHint, setRelayHint] = useState("")
  const [comment, setComment] = useState("")
  const [selectedLabels, setSelectedLabels] = useState<string[]>([])
  const [customLabel, setCustomLabel] = useState("")
  const [shouldBan, setShouldBan] = useState(false)

  const namespace = namespaceMode === customNamespaceValue ? customNamespace.trim() : namespaceMode
  const isBusy = createMutation.isPending || banMutation.isPending

  const targetPlaceholder = useMemo(() => {
    switch (targetType) {
      case "pubkey":
        return t("labels.create.targetPlaceholders.pubkey", "hex pubkey, npub ou nprofile")
      case "address":
        return t("labels.create.targetPlaceholders.address", "30023:pubkey:identifier")
      case "reference":
        return t("labels.create.targetPlaceholders.reference", "https://example.com ou relay")
      case "topic":
        return t("labels.create.targetPlaceholders.topic", "nome do topico")
      default:
        return t("labels.create.targetPlaceholders.event", "id em hex, note ou nevent")
    }
  }, [targetType, t])

  const resetForm = () => {
    setNamespaceMode(labelNamespaces[0] ?? "ugc")
    setCustomNamespace("")
    setTargetType("event")
    setTargetValue("")
    setRelayHint("")
    setComment("")
    setSelectedLabels([])
    setCustomLabel("")
    setShouldBan(false)
  }

  const handleTogglePreset = (value: string) => {
    const normalized = normalizeLabelValue(value)
    if (selectedLabels.includes(normalized)) {
      setSelectedLabels((current) => current.filter((item) => item !== normalized))
      return
    }

    const preset = labelPresets.find((item) => item.value === normalized)
    if (preset && namespaceMode !== customNamespaceValue) {
      setNamespaceMode(preset.namespace)
    }
    setSelectedLabels((current) => [...current, normalized])
  }

  const handleAddCustomLabel = () => {
    const normalized = normalizeLabelValue(customLabel)
    if (!normalized || selectedLabels.includes(normalized)) {
      return
    }
    setSelectedLabels((current) => [...current, normalized])
    setCustomLabel("")
  }

  const handleSubmit = async () => {
    if (!namespace) {
      toast.error(t("labels.create.errors.namespace", "Informe um namespace."))
      return
    }
    if (!targetValue.trim()) {
      toast.error(t("labels.create.errors.target", "Informe o alvo do label."))
      return
    }
    if (selectedLabels.length === 0) {
      toast.error(t("labels.create.errors.labels", "Selecione ao menos um label."))
      return
    }

    const normalizedTargetValue = normalizeNip19TargetInput(targetType, targetValue)

    const target: AdminLabelTarget = {
      type: targetType,
      value: normalizedTargetValue,
      relay_hint: relayHint.trim() || undefined,
    }

    try {
      await createMutation.mutateAsync({
        namespace,
        labels: selectedLabels,
        comment: comment.trim() || undefined,
        target,
      })

      if (shouldBan && targetType === "pubkey") {
        await banMutation.mutateAsync({
          pubkey: target.value,
          reason: `Labeled: ${selectedLabels.join(", ")}`,
        })
      }

      toast.success(t("labels.create.success", "Label publicado com sucesso."))
      onOpenChange(false)
      resetForm()
    } catch (error) {
      const message = error instanceof ApiError && error.requestId
        ? `${error.message} (request-id: ${error.requestId})`
        : error instanceof Error
          ? error.message
          : t("common.error")
      toast.error(message)
    }
  }

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{t("labels.create.title", "Criar label NIP-32")}</DialogTitle>
          <DialogDescription>{t("labels.create.description", "Publique um evento kind 1985 assinado pelo relay administrativo.")}</DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <Label>{t("labels.create.targetType", "Tipo de alvo")}</Label>
            <Select onValueChange={(value) => setTargetType(value as AdminLabelTargetType)} value={targetType}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {labelTargetOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label>{t("labels.create.namespace", "Namespace")}</Label>
            <Select onValueChange={setNamespaceMode} value={namespaceMode}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {labelNamespaces.map((item) => (
                  <SelectItem key={item} value={item}>{item}</SelectItem>
                ))}
                <SelectItem value={customNamespaceValue}>{t("labels.create.customNamespace", "Custom")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        {namespaceMode === customNamespaceValue ? (
          <div className="space-y-2">
            <Label>{t("labels.create.customNamespaceLabel", "Namespace customizado")}</Label>
            <Input onChange={(event) => setCustomNamespace(event.target.value)} placeholder="com.example.ontology" value={customNamespace} />
          </div>
        ) : null}

        <div className="space-y-2">
          <Label>{t("labels.create.targetValue", "Valor do alvo")}</Label>
          <Input onChange={(event) => setTargetValue(event.target.value)} placeholder={targetPlaceholder} value={targetValue} />
        </div>

        {targetType === "event" || targetType === "pubkey" ? (
          <div className="space-y-2">
            <Label>{t("labels.create.relayHint", "Relay hint (opcional)")}</Label>
            <Input onChange={(event) => setRelayHint(event.target.value)} placeholder="wss://relay.example" value={relayHint} />
          </div>
        ) : null}

        <div className="space-y-2">
          <Label>{t("labels.create.labels", "Labels")}</Label>
          <LabelCategoryPicker
            customValue={customLabel}
            onAddCustom={handleAddCustomLabel}
            onCustomValueChange={setCustomLabel}
            onRemove={(value) => setSelectedLabels((current) => current.filter((item) => item !== value))}
            onTogglePreset={handleTogglePreset}
            selected={selectedLabels}
          />
        </div>

        <div className="space-y-2">
          <Label>{t("labels.create.comment", "Comentário")}</Label>
          <Textarea onChange={(event) => setComment(event.target.value)} placeholder={t("labels.create.commentPlaceholder", "Justificativa, contexto ou observacao operacional")}
            value={comment}
          />
        </div>

        {targetType === "pubkey" ? (
          <button className="flex cursor-pointer items-center gap-2 rounded-[calc(var(--radius)-0.25rem)] border border-border px-3 py-2 text-left text-sm"
            onClick={() => setShouldBan((current) => !current)} type="button"
          >
            <span className={`size-3 rounded-full ${shouldBan ? "bg-accent" : "bg-muted"}`} />
            <span>{t("labels.create.banToggle", "Tambem banir esta pubkey apos publicar o label")}</span>
          </button>
        ) : null}

        <DialogFooter>
          <Button onClick={() => onOpenChange(false)} type="button" variant="outline">
            {t("common.cancel")}
          </Button>
          <Button disabled={isBusy} onClick={handleSubmit} type="button">
            {isBusy ? t("common.saving") : t("labels.create.submit", "Publicar label")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
