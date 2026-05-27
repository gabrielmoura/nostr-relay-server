import { useState } from "react"
import { Link } from "@tanstack/react-router"
import { ArrowLeft } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { BlossomObjectSheet } from "@/components/features/blossom/blossom-object-sheet"
import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel } from "@/components/shared/state-panels"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useInfiniteBlossomReports, useResolveBlossomReportMutation } from "@/hooks/use-admin-data"
import { normalizeFilterIdentifier } from "@/lib/nostr"
import { formatDateTime, shortenId } from "@/lib/utils"
import type { BlossomReportStatus } from "@/types/admin"

const allValue = "__all__"

export function BlossomReportsPage() {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const [reportType, setReportType] = useState("")
  const [status, setStatus] = useState<BlossomReportStatus | "">("")
  const [selectedHash, setSelectedHash] = useState("")
  const reportsQuery = useInfiniteBlossomReports({ q: query, report_type: reportType, status })
  const resolveMutation = useResolveBlossomReportMutation()
  const reports = reportsQuery.data?.pages.flatMap((page) => page.items) ?? []

  const runResolve = async (id: string, nextStatus: "dismissed" | "actioned") => {
    try {
      await resolveMutation.mutateAsync({ id, status: nextStatus, note: "resolved from reports route" })
      toast.success(t("blossom.reports.resolved", "Report atualizado."))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("common.error"))
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        actions={<Button asChild type="button" variant="outline"><Link to="/blossom"><ArrowLeft className="size-4" />{t("blossom.plans.back", "Voltar ao Blossom")}</Link></Button>}
        description={t("blossom.reports.routeDescription", "Investigue relatos BUD-09, conecte evidências ao blob e conclua a decisão em uma superfície dedicada.")}
        title={t("blossom.reports.routeTitle", "Reports Blossom")}
      />

      <div className="grid gap-3 lg:grid-cols-3">
        <Input onChange={(event) => setQuery(normalizeFilterIdentifier("generic", event.target.value))} placeholder={t("blossom.reports.search", "Buscar hash, reporter, evento ou motivo")} value={query} />
        <Input onChange={(event) => setReportType(event.target.value)} placeholder={t("blossom.reports.type", "Tipo de report")} value={reportType} />
        <Select onValueChange={(value) => setStatus(value === allValue ? "" : (value as BlossomReportStatus))} value={status || allValue}>
          <SelectTrigger><SelectValue placeholder={t("blossom.reports.status", "Status")} /></SelectTrigger>
          <SelectContent>
            <SelectItem value={allValue}>{t("common.all", "Todos")}</SelectItem>
            <SelectItem value="open">open</SelectItem>
            <SelectItem value="dismissed">dismissed</SelectItem>
            <SelectItem value="actioned">actioned</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {reports.length === 0 ? <EmptyPanel description={t("blossom.reports.emptyDescription", "Nenhum report corresponde aos filtros atuais.")} title={t("blossom.reports.emptyTitle", "Sem reports nesta visão")} /> : (
        <Card>
          <Table>
            <TableHeader><TableRow><TableHead>Hash</TableHead><TableHead>Reporter</TableHead><TableHead>Tipo</TableHead><TableHead>Motivo</TableHead><TableHead>Status</TableHead><TableHead>Ações</TableHead></TableRow></TableHeader>
            <TableBody>
              {reports.map((item) => (
                <TableRow key={item.id}>
                  <TableCell><button className="font-mono text-xs text-primary" onClick={() => setSelectedHash(item.object_hash)} type="button">{shortenId(item.object_hash, 14, 8)}</button></TableCell>
                  <TableCell className="font-mono text-xs">{shortenId(item.reporter_npub || item.reporter_pubkey, 14, 4)}</TableCell>
                  <TableCell>{item.report_type ?? "-"}</TableCell>
                  <TableCell className="max-w-[24rem] truncate">{item.reason ?? "-"}</TableCell>
                  <TableCell><Badge variant={item.status === "open" ? "warning" : "success"}>{item.status}</Badge></TableCell>
                  <TableCell><div className="flex gap-2">{item.status === "open" ? <><Button onClick={() => void runResolve(item.id, "dismissed")} size="sm" type="button" variant="outline">dismiss</Button><Button onClick={() => void runResolve(item.id, "actioned")} size="sm" type="button">action</Button></> : <span className="text-xs text-muted-foreground">{item.resolved_at ? formatDateTime(item.resolved_at) : "resolvido"}</span>}</div></TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      <BlossomObjectSheet hash={selectedHash} onApprove={() => undefined} onDelete={() => undefined} onOpenChange={(open) => { if (!open) setSelectedHash("") }} onRequeue={() => undefined} open={Boolean(selectedHash)} />
    </div>
  )
}
