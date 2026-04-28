import { Suspense, useState } from "react"
import { useTranslation } from "react-i18next"
import { Plus, Trash2, UserCheck } from "lucide-react"
import { toast } from "sonner"

import { PageHeader } from "@/components/shared/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useAddTrustedPubkeyMutation, useRemoveTrustedPubkeyMutation, useWoTSummarySuspense, isFeatureDisabledError } from "@/hooks/use-admin-data"
import { FeatureDisabledPanel } from "@/components/shared/feature-disabled-panel"
import { Skeleton } from "@/components/ui/skeleton"
import { ErrorBoundary } from "react-error-boundary"

export function WoTPage() {
  return (
    <ErrorBoundary
      fallbackRender={({ error }: { error: any }) => {
        if (isFeatureDisabledError(error)) {
          return (
            <FeatureDisabledPanel
              configKey="wot.enabled"
              description="A funcionalidade Web of Trust permite filtrar eventos com base em uma rede de confiança a partir de pubkeys confiáveis."
              howToEnable="wot:
  enabled: true
  trusted_pubkeys:
    - <sua_pubkey_hex>"
              title="Web of Trust Desabilitado"
            />
          )
        }
        throw error
      }}
    >
      <Suspense fallback={<WoTSkeleton />}>
        <WoTContent />
      </Suspense>
    </ErrorBoundary>
  )
}

function WoTContent() {
  const { t } = useTranslation()
  const summaryQuery = useWoTSummarySuspense()
  const addMutation = useAddTrustedPubkeyMutation()
  const removeMutation = useRemoveTrustedPubkeyMutation()

  const [newPubkey, setNewPubkey] = useState("")

  const handleAdd = async () => {
    if (!newPubkey.trim()) return
    try {
      await addMutation.mutateAsync(newPubkey.trim())
      setNewPubkey("")
      toast.success(t("wot.successAdd", "Pubkey adicionada com sucesso"))
    } catch (error) {
      console.error("Add failed:", error)
      toast.error(t("common.error"))
    }
  }

  const handleRemove = async (pubkey: string) => {
    try {
      await removeMutation.mutateAsync(pubkey)
      toast.success(t("wot.successRemove", "Pubkey removida com sucesso"))
    } catch (error) {
      console.error("Remove failed:", error)
      toast.error(t("common.error"))
    }
  }

  const summary = summaryQuery.data

  return (
    <div className="space-y-6">
      <PageHeader
        description={t("wot.description")}
        title={t("wot.title")}
      />

      <div className="grid gap-6 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">{t("wot.summary.nodes")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{summary?.total_nodes ?? 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">{t("wot.summary.edges")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{summary?.total_edges ?? 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">{t("wot.summary.lastComputed")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-sm font-medium">{summary?.last_computed_at ? new Date(summary.last_computed_at).toLocaleString() : t("userDetail.notAvailable")}</div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <UserCheck className="size-5" />
            {t("wot.trustedPubkeys")}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Input
              onChange={(e) => setNewPubkey(e.target.value)}
              placeholder={t("wot.addPlaceholder")}
              value={newPubkey}
            />
            <Button disabled={addMutation.isPending} onClick={handleAdd}>
              {addMutation.isPending ? <Plus className="size-4 animate-spin" /> : <Plus className="size-4" />}
              {t("wot.addAction")}
            </Button>
          </div>

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("common.pubkey")}</TableHead>
                <TableHead className="text-right">{t("common.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(summary?.trusted_pubkeys || []).map((pubkey: string) => (
                <TableRow key={pubkey}>
                  <TableCell className="font-mono text-xs break-all">{pubkey}</TableCell>
                  <TableCell className="text-right">
                    <Button
                      disabled={removeMutation.isPending}
                      onClick={() => handleRemove(pubkey)}
                      size="icon"
                      variant="ghost"
                    >
                      <Trash2 className="size-4 text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
              {(!summary?.trusted_pubkeys || summary.trusted_pubkeys.length === 0) && (
                <TableRow>
                  <TableCell className="text-center text-muted-foreground" colSpan={2}>
                    {t("nip05.emptyTitle", "Nenhuma pubkey confiável encontrada")}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}

function WoTSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-24 w-full" />
      <div className="grid gap-6 md:grid-cols-3">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
      <Skeleton className="h-64 w-full" />
    </div>
  )
}
