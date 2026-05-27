import { ArrowLeft } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "@tanstack/react-router"
import { toast } from "sonner"

import { PageHeader } from "@/components/shared/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { useBlossomPolicy, useUpdateBlossomPolicyMutation } from "@/hooks/use-admin-data"
import type { BlossomPolicyMode } from "@/types/admin"

export function BlossomPolicyPage() {
  const { t } = useTranslation()
  const policyQuery = useBlossomPolicy()
  const updatePolicyMutation = useUpdateBlossomPolicyMutation()

  const handleModeChange = async (mode: BlossomPolicyMode) => {
    try {
      await updatePolicyMutation.mutateAsync({ ...(policyQuery.data ?? { mode }), mode })
      toast.success(t("blossom.policy.saved", "Política atualizada com sucesso."))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("common.error"))
    }
  }

  const currentMode = policyQuery.data?.mode ?? "free"

  return (
    <div className="space-y-6">
      <PageHeader
        actions={
          <Button asChild type="button" variant="outline">
            <Link to="/blossom">
              <ArrowLeft className="size-4" />
              {t("common.back", "Voltar")}
            </Link>
          </Button>
        }
        className="rounded-[var(--radius)] border border-primary/15 bg-[linear-gradient(135deg,rgba(30,64,175,0.08),rgba(245,158,11,0.07))] p-5 panel-shadow"
        description={t("blossom.policy.description", "Defina a política geral de uso de uploads do servidor.")}
        title={t("blossom.policy.title", "Política de Uploads")}
      />

      <Card className="max-w-3xl">
        <CardHeader>
          <CardTitle>{t("blossom.policy.modeTitle", "Modo efetivo de publicação")}</CardTitle>
          <CardDescription>
            {t("blossom.policy.modeDescription", "Escolha como os usuários poderão usar o servidor Blossom. Você só pode selecionar uma opção por vez.")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4">
            <Label
              className={`flex cursor-pointer items-start gap-4 rounded-lg border p-4 transition-colors hover:bg-accent ${
                currentMode === "free" ? "border-primary bg-primary/5" : ""
              }`}
              htmlFor="mode-free"
            >
              <input checked={currentMode === "free"} className="mt-1" disabled={updatePolicyMutation.isPending} id="mode-free" name="blossom-policy-mode" onChange={() => void handleModeChange("free")} type="radio" />
              <div className="grid gap-1.5">
                <div className="font-semibold">{t("blossom.policy.modeFree", "Acesso Livre (free)")}</div>
                <p className="text-sm text-muted-foreground">
                  {t("blossom.policy.modeFreeDesc", "Qualquer pessoa pode fazer upload. Cotas de upload serão geridas pelo plano default livre ou de usuário autenticado.")}
                </p>
              </div>
            </Label>

            <Label
              className={`flex cursor-pointer items-start gap-4 rounded-lg border p-4 transition-colors hover:bg-accent ${
                currentMode === "enabled_users" ? "border-primary bg-primary/5" : ""
              }`}
              htmlFor="mode-enabled"
            >
              <input checked={currentMode === "enabled_users"} className="mt-1" disabled={updatePolicyMutation.isPending} id="mode-enabled" name="blossom-policy-mode" onChange={() => void handleModeChange("enabled_users")} type="radio" />
              <div className="grid gap-1.5">
                <div className="font-semibold">{t("blossom.policy.modeEnabled", "Apenas Usuários Habilitados (enabled_users)")}</div>
                <p className="text-sm text-muted-foreground">
                  {t("blossom.policy.modeEnabledDesc", "Somente usuários que estiverem com a tag 'Habilitado' na aba de Usuários poderão realizar upload.")}
                </p>
              </div>
            </Label>

            <Label
              className={`flex cursor-pointer items-start gap-4 rounded-lg border p-4 transition-colors hover:bg-accent ${
                currentMode === "mandatory_review" ? "border-primary bg-primary/5" : ""
              }`}
              htmlFor="mode-mandatory"
            >
              <input checked={currentMode === "mandatory_review"} className="mt-1" disabled={updatePolicyMutation.isPending} id="mode-mandatory" name="blossom-policy-mode" onChange={() => void handleModeChange("mandatory_review")} type="radio" />
              <div className="grid gap-1.5">
                <div className="font-semibold">{t("blossom.policy.modeMandatory", "Revisão Obrigatória (mandatory_review)")}</div>
                <p className="text-sm text-muted-foreground">
                  {t("blossom.policy.modeMandatoryDesc", "Nenhum arquivo fica público automaticamente. Todo arquivo deve ser revisado por um moderador.")}
                </p>
              </div>
            </Label>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
