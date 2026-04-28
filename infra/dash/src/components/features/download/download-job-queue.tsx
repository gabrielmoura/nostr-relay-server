import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { Eye, Filter, LoaderCircle, TriangleAlert } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import type { DownloadJob } from "@/types/admin"

type DownloadJobQueueProps = {
  jobs: DownloadJob[]
}

export function DownloadJobQueue({ jobs }: DownloadJobQueueProps) {
  const { t } = useTranslation()
  const [selectedJob, setSelectedJob] = useState<DownloadJob | null>(null)
  const [dialogMode, setDialogMode] = useState<"filters" | "details">("details")

  const sorted = useMemo(() => [...jobs], [jobs])

  return (
    <Card className="panel-shadow">
      <CardHeader>
        <CardTitle>{t("download.jobsTitle", "Fila de trabalhos")}</CardTitle>
        <CardDescription>{t("download.jobsDescription", "Acompanhe downloads em progresso, concluídos e com falha a partir do status real do backend.")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {sorted.length === 0 ? (
          <div className="rounded-[calc(var(--radius)-0.2rem)] border border-dashed border-border px-4 py-6 text-sm text-muted-foreground">
            {t("download.jobsEmpty", "Nenhum trabalho de download foi iniciado ainda.")}
          </div>
        ) : (
          <div className="space-y-3">
            {sorted.map((job) => (
              <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border/80 bg-background/70 p-4" key={job.id}>
                <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                  <div className="space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="font-mono text-xs text-muted-foreground">{job.id}</p>
                      <JobStatusBadge status={job.status} />
                    </div>
                    <p className="text-sm text-foreground">{job.message || t("download.jobDefaultMessage", "Trabalho de download")}</p>
                    <p className="text-xs text-muted-foreground">
                      {t("download.jobMeta", {
                        relays: job.relays.length,
                        timeout: job.timeout,
                        defaultValue: `${job.relays.length} relays · timeout ${job.timeout}s`,
                      })}
                    </p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <Button onClick={() => openDialog(job, "filters", setSelectedJob, setDialogMode)} size="sm" type="button" variant="outline">
                      <Filter className="size-4" />
                      {t("download.viewFilters", "Ver filtros")}
                    </Button>
                    <Button onClick={() => openDialog(job, "details", setSelectedJob, setDialogMode)} size="sm" type="button" variant="outline">
                      <Eye className="size-4" />
                      {t("download.viewDetails", "Ver detalhes")}
                    </Button>
                  </div>
                </div>

                <div className="mt-4 grid gap-3 md:grid-cols-4">
                  <SummaryStat label={t("download.summary.received", "Recebidos")} value={String(job.summary.events_received ?? 0)} />
                  <SummaryStat label={t("download.summary.inserted", "Inseridos")} value={String(job.summary.inserted_events ?? 0)} />
                  <SummaryStat label={t("download.summary.duplicates", "Duplicados")} value={String(job.summary.duplicate_events ?? 0)} />
                  <SummaryStat label={t("download.summary.pages", "Páginas")} value={String(job.summary.pages ?? 0)} />
                </div>

                {job.error ? (
                  <div className="mt-4 flex items-start gap-2 rounded-[calc(var(--radius)-0.25rem)] border border-destructive/30 bg-destructive/5 px-3 py-3 text-sm text-destructive">
                    <TriangleAlert className="mt-0.5 size-4 shrink-0" />
                    <p>{job.error}</p>
                  </div>
                ) : null}
              </div>
            ))}
          </div>
        )}

        <Dialog onOpenChange={(open) => !open && setSelectedJob(null)} open={Boolean(selectedJob)}>
          <DialogContent className="max-w-4xl">
            <DialogHeader>
              <DialogTitle>{dialogMode === "filters" ? t("download.filtersDialogTitle", "Filtros do trabalho") : t("download.detailsDialogTitle", "Detalhes do trabalho")}</DialogTitle>
              <DialogDescription>{selectedJob?.id}</DialogDescription>
            </DialogHeader>

            {selectedJob ? (
              dialogMode === "filters" ? (
                <pre className="max-h-[60vh] overflow-auto rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/30 p-4 text-xs text-foreground">
                  {selectedJob.filter_json}
                </pre>
              ) : (
                <div className="space-y-4">
                  <div className="grid gap-3 md:grid-cols-2">
                    <SummaryStat label={t("download.summary.successfulRelays", "Relays com sucesso")} value={String(selectedJob.summary.successful_relays ?? 0)} />
                    <SummaryStat label={t("download.summary.failedRelays", "Relays com falha")} value={String(selectedJob.summary.failed_relays ?? 0)} />
                  </div>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t("download.table.relay", "Relay")}</TableHead>
                        <TableHead>{t("download.table.status", "Status")}</TableHead>
                        <TableHead>{t("download.table.received", "Recebidos")}</TableHead>
                        <TableHead>{t("download.table.inserted", "Inseridos")}</TableHead>
                        <TableHead>{t("download.table.duplicates", "Duplicados")}</TableHead>
                        <TableHead>{t("download.table.pages", "Páginas")}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {selectedJob.relay_results.map((result) => (
                        <TableRow key={`${selectedJob.id}-${result.relay}`}>
                          <TableCell className="font-mono text-xs">{result.relay}</TableCell>
                          <TableCell><JobStatusBadge status={result.status as any} /></TableCell>
                          <TableCell>{result.events_received}</TableCell>
                          <TableCell>{result.inserted_events}</TableCell>
                          <TableCell>{result.duplicate_events}</TableCell>
                          <TableCell>{result.pages}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )
            ) : null}
          </DialogContent>
        </Dialog>
      </CardContent>
    </Card>
  )
}

function openDialog(job: DownloadJob, mode: "filters" | "details", setSelectedJob: (job: DownloadJob) => void, setDialogMode: (mode: "filters" | "details") => void) {
  setDialogMode(mode)
  setSelectedJob(job)
}

function SummaryStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border/70 bg-card/80 px-3 py-3">
      <p className="text-xs uppercase tracking-[0.14em] text-muted-foreground">{label}</p>
      <p className="mt-1 font-heading text-lg text-foreground">{value}</p>
    </div>
  )
}

function JobStatusBadge({ status }: { status: string }) {
  const { t } = useTranslation()
  if (status === "running" || status === "queued") {
    return (
      <Badge variant="warning">
        <LoaderCircle className="size-3 animate-spin" />
        {t(`download.status.${status}`, { defaultValue: status })}
      </Badge>
    )
  }
  if (status === "completed" || status === "found") {
    return <Badge variant="success">{t(`download.status.${status}`, { defaultValue: status })}</Badge>
  }
  if (status === "failed") {
    return <Badge variant="danger">{t(`download.status.${status}`, { defaultValue: status })}</Badge>
  }
  return <Badge variant="muted">{t(`download.status.${status}`, { defaultValue: status })}</Badge>
}
