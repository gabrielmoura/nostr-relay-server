import type { ReactNode } from "react"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { z } from "zod"

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

const quotaSchema = z.string().trim().refine((value) => {
  if (value === "") return true
  const bytes = Number(value) * 1024 * 1024
  return /^\d+(?:\.\d+)?$/.test(value) && Number.isSafeInteger(bytes) && bytes >= 0
}, "Informe um valor não negativo em MB que represente um número inteiro de bytes.")

const planSchema = z.object({
  id: z.string().trim().min(1, "Informe o ID do plano."),
  name: z.string().trim().min(1, "Informe o nome do plano."),
  storage: quotaSchema,
  egress: quotaSchema,
})

function quotaInput(bytes: number | null | undefined) {
  if (bytes == null) return ""
  const whole = Math.floor(bytes / (1024 * 1024))
  const remainder = bytes % (1024 * 1024)
  if (remainder === 0) return String(whole)
  return `${whole}.${(remainder / (1024 * 1024)).toFixed(20).slice(2).replace(/0+$/, "")}`
}

export function BlossomPlanModal({ open, onOpenChange, plan, onSave, saving }: BlossomPlanModalProps) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState<BlossomPlan>(emptyPlan)
  const [storage, setStorage] = useState("")
  const [egress, setEgress] = useState("")
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    setDraft(plan ?? emptyPlan)
    setStorage(quotaInput(plan?.storage_quota_bytes))
    setEgress(quotaInput(plan?.egress_quota_bytes))
    setErrors({})
  }, [plan, open])

  const editing = Boolean(plan)

  const save = () => {
    const result = planSchema.safeParse({ id: draft.id, name: draft.name, storage, egress })
    if (!result.success) {
      const next: Record<string, string> = {}
      for (const issue of result.error.issues) next[String(issue.path[0])] ??= issue.message
      setErrors(next)
      return
    }
    setErrors({})
    onSave({ ...draft, id: result.data.id, name: result.data.name,
      storage_quota_bytes: result.data.storage === "" ? undefined : Number(result.data.storage) * 1024 * 1024,
      egress_quota_bytes: result.data.egress === "" ? undefined : Number(result.data.egress) * 1024 * 1024 })
  }

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{editing ? t("blossom.plans.editPlanTitle", "Editar plano") : t("blossom.plans.createPlanTitle", "Criar plano")}</DialogTitle>
          <DialogDescription>{t("blossom.plans.editorDescription", "Defina escopo, cotas e metadados operacionais do plano.")}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 md:grid-cols-2">
          <Field id="blossom-plan-id" label={t("blossom.plans.id", "ID tecnico")} error={errors.id}>
            <Input id="blossom-plan-id" aria-invalid={Boolean(errors.id)} disabled={editing} onChange={(event) => setDraft((current) => ({ ...current, id: event.target.value }))} value={draft.id} />
          </Field>
          <Field id="blossom-plan-name" label={t("blossom.plans.name", "Nome do plano")} error={errors.name}>
            <Input id="blossom-plan-name" aria-invalid={Boolean(errors.name)} onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))} value={draft.name} />
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
          <Field id="blossom-plan-storage" label={t("blossom.plans.storageMb", "Armazenamento (MB)")} error={errors.storage}>
            <Input
              id="blossom-plan-storage" aria-invalid={Boolean(errors.storage)}
              onChange={(event) => setStorage(event.target.value)}
              placeholder={t("blossom.plans.unlimited", "Ilimitado")}
              type="number"
              min="0" step="any"
              value={storage}
            />
          </Field>
          <Field id="blossom-plan-egress" label={t("blossom.plans.egressMb", "Egress mensal (MB)")} error={errors.egress}>
            <Input
              id="blossom-plan-egress" aria-invalid={Boolean(errors.egress)}
              onChange={(event) => setEgress(event.target.value)}
              placeholder={t("blossom.plans.unlimited", "Ilimitado")}
              type="number"
              min="0" step="any"
              value={egress}
            />
          </Field>
        </div>
        <p className="text-sm text-muted-foreground">Deixe a cota em branco para ilimitado; zero impede o uso dessa cota.</p>
        <Field label={t("blossom.plans.descriptionField", "Descricao operacional") }>
          <Textarea onChange={(event) => setDraft((current) => ({ ...current, description: event.target.value }))} rows={4} value={draft.description ?? ""} />
        </Field>
        <DialogFooter>
          <Button onClick={() => onOpenChange(false)} type="button" variant="outline">{t("common.cancel", "Cancelar")}</Button>
          <Button disabled={saving} onClick={save} type="button">{saving ? t("common.saving", "Salvando...") : t("common.save", "Salvar")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function Field({ id, label, children, error }: { id?: string; label: string; children: ReactNode; error?: string }) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      {children}
      {error ? <p className="text-sm text-destructive" role="alert">{error}</p> : null}
    </div>
  )
}
