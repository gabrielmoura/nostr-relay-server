import type { ReactNode } from "react"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import type { BlossomPlan, BlossomPlanScope } from "@/types/admin"

type BlossomPlanModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  plan?: BlossomPlan
  onSave: (plan: BlossomPlan) => void
  saving: boolean
}

const emptyPlan: BlossomPlan = {
  id: "",
  name: "",
  scope: "free",
  description: "",
  is_default: false,
}

export function BlossomPlanModal({ open, onOpenChange, plan, onSave, saving }: BlossomPlanModalProps) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState<BlossomPlan>(emptyPlan)

  useEffect(() => {
    setDraft(plan ?? emptyPlan)
  }, [plan, open])

  const editing = Boolean(plan)

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{editing ? t("blossom.plans.editPlanTitle", "Editar plano") : t("blossom.plans.createPlanTitle", "Criar plano")}</DialogTitle>
          <DialogDescription>{t("blossom.plans.editorDescription", "Defina escopo, cotas e metadados operacionais do plano.")}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 md:grid-cols-2">
          <Field label={t("blossom.plans.id", "ID tecnico")}>
            <Input disabled={editing} onChange={(event) => setDraft((current) => ({ ...current, id: event.target.value }))} value={draft.id} />
          </Field>
          <Field label={t("blossom.plans.name", "Nome do plano")}>
            <Input onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))} value={draft.name} />
          </Field>
          <Field label={t("blossom.plans.scope", "Escopo") }>
            <Select onValueChange={(value) => setDraft((current) => ({ ...current, scope: value as BlossomPlanScope }))} value={draft.scope}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="free">{t("blossom.policy.modeFreeLabel", "Livre")}</SelectItem>
                <SelectItem value="enabled_users">{t("blossom.policy.modeEnabledLabel", "Somente habilitados")}</SelectItem>
              </SelectContent>
            </Select>
          </Field>
          <Field label={t("blossom.plans.default", "Plano padrao") }>
            <label className="flex h-10 items-center gap-3 rounded-[var(--radius)] border border-border px-3 text-sm">
              <input checked={draft.is_default} className="size-4" onChange={(event) => setDraft((current) => ({ ...current, is_default: event.target.checked }))} type="checkbox" />
              <span>{t("blossom.plans.defaultHint", "Aplicar automaticamente como plano padrao deste escopo")}</span>
            </label>
          </Field>
          <Field label={t("blossom.plans.storageMb", "Armazenamento (MB)")}>
            <Input
              onChange={(event) => setDraft((current) => ({ ...current, storage_quota_bytes: event.target.value ? Number(event.target.value) * 1024 * 1024 : undefined }))}
              placeholder={t("blossom.plans.unlimited", "Ilimitado")}
              type="number"
              value={draft.storage_quota_bytes ? Math.round(draft.storage_quota_bytes / (1024 * 1024)) : ""}
            />
          </Field>
          <Field label={t("blossom.plans.egressMb", "Egress mensal (MB)")}>
            <Input
              onChange={(event) => setDraft((current) => ({ ...current, egress_quota_bytes: event.target.value ? Number(event.target.value) * 1024 * 1024 : undefined }))}
              placeholder={t("blossom.plans.unlimited", "Ilimitado")}
              type="number"
              value={draft.egress_quota_bytes ? Math.round(draft.egress_quota_bytes / (1024 * 1024)) : ""}
            />
          </Field>
        </div>
        <Field label={t("blossom.plans.descriptionField", "Descricao operacional") }>
          <Textarea onChange={(event) => setDraft((current) => ({ ...current, description: event.target.value }))} rows={4} value={draft.description ?? ""} />
        </Field>
        <DialogFooter>
          <Button onClick={() => onOpenChange(false)} type="button" variant="outline">{t("common.cancel", "Cancelar")}</Button>
          <Button disabled={saving} onClick={() => onSave(draft)} type="button">{saving ? t("common.saving", "Salvando...") : t("common.save", "Salvar")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      {children}
    </div>
  )
}
