import { useMemo, useState } from "react"
import { Link } from "@tanstack/react-router"
import { ArrowLeft, ShieldCheck } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { BlossomObjectSheet } from "@/components/features/blossom/blossom-object-sheet"
import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel } from "@/components/shared/state-panels"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useBlossomBulkReviewMutation, useBlossomPolicy, useInfiniteBlossomObjects } from "@/hooks/use-admin-data"
import { blossomExifVariant, blossomReviewVariant } from "@/lib/blossom"
import { formatBytes, formatDateTime, shortenId } from "@/lib/utils"

export function BlossomReviewPage() {
  const { t } = useTranslation()
  const policyQuery = useBlossomPolicy()
  const [query, setQuery] = useState("")
  const [selectedHash, setSelectedHash] = useState("")
  const objectsQuery = useInfiniteBlossomObjects({ q: query })
  const reviewMutation = useBlossomBulkReviewMutation()

  const reviewEnabled = policyQuery.data?.mode === "mandatory_review"
  const objects = useMemo(() => objectsQuery.data?.pages.flatMap((page) => page.items) ?? [], [objectsQuery.data])
  const reviewItems = useMemo(() => objects.filter((item) => item.review_state === "flagged" || item.review_state === "pending_review"), [objects])

  const runAction = async (hash: string, action: "approve" | "hard_delete" | "requeue_optimization") => {
    try {
      await reviewMutation.mutateAsync({ hashes: [hash], action, reason: "review child route" })
      toast.success(t("blossom.bulk.success", "Ação aplicada com sucesso."))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("common.error"))
    }
  }

  if (!reviewEnabled) {
    return (
      <EmptyPanel
        action={<Button asChild type="button" variant="outline"><Link to="/blossom"><ArrowLeft className="size-4" />{t("blossom.plans.back", "Voltar ao Blossom")}</Link></Button>}
        description={t("blossom.review.disabledDescription", "A revisão só fica disponível quando o modo efetivo exige aprovação manual dos uploads.")}
        title={t("blossom.review.disabledTitle", "Revisão desabilitada")}
      />
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        actions={<Button asChild type="button" variant="outline"><Link to="/blossom"><ArrowLeft className="size-4" />{t("blossom.plans.back", "Voltar ao Blossom")}</Link></Button>}
        description={t("blossom.review.routeDescription", "Aplique decisões de aprovação, exclusão e reprocessamento com foco exclusivo na fila moderada.")}
        title={t("blossom.review.routeTitle", "Fila de revisão Blossom")}
      />

      <div className="grid gap-4 lg:grid-cols-[1fr_auto]">
        <Input onChange={(event) => setQuery(event.target.value)} placeholder={t("blossom.review.search", "Buscar hash, MIME ou uploader")} value={query} />
        <Badge variant="warning">{reviewItems.length} {t("blossom.review.pending", "pendentes")}</Badge>
      </div>

      {reviewItems.length === 0 ? <EmptyPanel description={t("blossom.review.emptyDescription", "Nenhum arquivo está pendente de revisão agora.")} title={t("blossom.review.emptyTitle", "Fila de revisão vazia")} /> : (
        <Card>
          <Table>
            <TableHeader><TableRow><TableHead>SHA-256</TableHead><TableHead>MIME</TableHead><TableHead>Tamanho</TableHead><TableHead>Data</TableHead><TableHead>Estado</TableHead><TableHead>Ações</TableHead></TableRow></TableHeader>
            <TableBody>
              {reviewItems.map((item) => (
                <TableRow key={item.hash}>
                  <TableCell><button className="font-mono text-xs text-primary" onClick={() => setSelectedHash(item.hash)} type="button">{shortenId(item.hash, 14, 8)}</button></TableCell>
                  <TableCell>{item.mime_type}</TableCell>
                  <TableCell>{formatBytes(item.size)}</TableCell>
                  <TableCell>{formatDateTime(item.created_at)}</TableCell>
                  <TableCell><div className="flex flex-wrap gap-2"><Badge variant={blossomReviewVariant(item.review_state)}>{item.review_state}</Badge><Badge variant={blossomExifVariant(item.exif_status)}>{item.exif_status}</Badge></div></TableCell>
                  <TableCell><div className="flex gap-2"><Button onClick={() => void runAction(item.hash, "approve")} size="sm" type="button"><ShieldCheck className="size-4" />{t("blossom.bulk.approve", "Aprovar")}</Button><Button onClick={() => void runAction(item.hash, "hard_delete")} size="sm" type="button" variant="destructive">{t("blossom.bulk.delete", "Excluir")}</Button></div></TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      <BlossomObjectSheet hash={selectedHash} onApprove={(hash) => void runAction(hash, "approve")} onDelete={(hash) => void runAction(hash, "hard_delete")} onOpenChange={(open) => { if (!open) setSelectedHash("") }} onRequeue={(hash) => void runAction(hash, "requeue_optimization")} open={Boolean(selectedHash)} />
    </div>
  )
}
