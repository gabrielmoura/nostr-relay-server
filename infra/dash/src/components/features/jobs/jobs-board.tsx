import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { AlertTriangle, Clock3, LoaderCircle, RefreshCcw, SquareStack, XCircle } from "lucide-react"

import { useCancelJobMutation, useDeleteJobsHistoryMutation, useJobQuery, useJobsQuery, useResumeJobMutation, useRetryJobMutation } from "@/hooks/use-admin-data"
import type { AdminJob, AdminJobsFilters, AdminJobStatus } from "@/types/admin"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Skeleton } from "@/components/ui/skeleton"

type JobsBoardProps = {
  title: string
  description: string
  filters: AdminJobsFilters
  emptyTitle: string
  emptyDescription: string
}

export function JobsBoard({ title, description, filters, emptyTitle, emptyDescription }: JobsBoardProps) {
  const { t } = useTranslation()
  const [statusFilter, setStatusFilter] = useState<AdminJobStatus | "">("")
  const [selectedJob, setSelectedJob] = useState<AdminJob | null>(null)

  const jobsQuery = useJobsQuery({ ...filters, status: statusFilter || undefined, limit: 30 })
  const retryMutation = useRetryJobMutation()
  const cancelMutation = useCancelJobMutation()
  const resumeMutation = useResumeJobMutation()
  const deleteHistoryMutation = useDeleteJobsHistoryMutation()
  const detailQuery = useJobQuery(selectedJob?.id ?? "", selectedJob?.queue, Boolean(selectedJob))

  const jobs = jobsQuery.data?.items ?? []
  const activeCount = jobs.filter((job) => job.status === "queued" || job.status === "running" || job.status === "delayed").length
  const failedCount = jobs.filter((job) => job.status === "failed" || job.status === "dead").length
  const completedCount = jobs.filter((job) => job.status === "succeeded").length

  const selected = detailQuery.data ?? selectedJob

  const summary = useMemo(
    () => [
      { label: t("jobs.summary.active", "Ativos"), value: activeCount, icon: LoaderCircle },
      { label: t("jobs.summary.completed", "Concluídos"), value: completedCount, icon: SquareStack },
      { label: t("jobs.summary.failed", "Falhos/DLQ"), value: failedCount, icon: AlertTriangle },
    ],
    [activeCount, completedCount, failedCount, t],
  )

  const clearableCount = jobs.filter((job) => job.status === "succeeded" || job.status === "failed" || job.status === "dead").length

  const handleClearHistory = async () => {
    if (!filters.job_name || clearableCount === 0) {
      return
    }
    await deleteHistoryMutation.mutateAsync({
      job_name: filters.job_name,
      queue: filters.queue,
      statuses: ["succeeded", "failed", "dead"],
    })
  }

  return (
    <Card className="panel-shadow border-border/70 bg-card/90">
      <CardHeader className="space-y-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <CardTitle>{title}</CardTitle>
            <CardDescription>{description}</CardDescription>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              onChange={(event) => setStatusFilter(event.target.value as AdminJobStatus | "")}
              value={statusFilter}
            >
              <option value="">{t("jobs.filters.allStatuses", "Todos os estados")}</option>
              <option value="queued">{t("jobs.status.queued", "queued")}</option>
              <option value="delayed">{t("jobs.status.delayed", "delayed")}</option>
              <option value="running">{t("jobs.status.running", "running")}</option>
              <option value="succeeded">{t("jobs.status.succeeded", "succeeded")}</option>
              <option value="failed">{t("jobs.status.failed", "failed")}</option>
              <option value="dead">{t("jobs.status.dead", "dead")}</option>
              <option value="canceled">{t("jobs.status.canceled", "canceled")}</option>
            </select>
            <Button onClick={() => jobsQuery.refetch()} size="sm" type="button" variant="outline">
              <RefreshCcw className={`size-4 ${jobsQuery.isFetching ? "animate-spin" : ""}`} />
              {t("jobs.actions.refresh", "Atualizar")}
            </Button>
            <Button disabled={clearableCount === 0 || deleteHistoryMutation.isPending || !filters.job_name} onClick={() => void handleClearHistory()} size="sm" type="button" variant="outline">
              {t("jobs.actions.clearHistory", "Limpar histórico")}
            </Button>
          </div>
        </div>

        <div className="grid gap-3 md:grid-cols-3">
          {summary.map((item) => (
            <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border/70 bg-background/70 px-4 py-3" key={item.label}>
              <div className="flex items-center justify-between">
                <p className="text-xs uppercase tracking-[0.14em] text-muted-foreground">{item.label}</p>
                <item.icon className="size-4 text-primary" />
              </div>
              <p className="mt-2 font-heading text-2xl text-foreground">{item.value}</p>
            </div>
          ))}
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {jobsQuery.isLoading ? <JobsBoardSkeleton /> : null}

        {!jobsQuery.isLoading && jobsQuery.error ? (
          <div className="rounded-[calc(var(--radius)-0.25rem)] border border-destructive/30 bg-destructive/5 px-4 py-4 text-sm text-destructive">
            <p className="font-medium">{t("jobs.errors.loadTitle", "Falha ao carregar trabalhos")}</p>
            <p className="mt-1 text-destructive/90">{jobsQuery.error instanceof Error ? jobsQuery.error.message : t("common.error", "Erro inesperado")}</p>
          </div>
        ) : null}

        {!jobsQuery.isLoading && !jobsQuery.error && jobs.length === 0 ? (
          <div className="rounded-[calc(var(--radius)-0.2rem)] border border-dashed border-border px-4 py-8 text-sm">
            <p className="font-medium text-foreground">{emptyTitle}</p>
            <p className="mt-1 text-muted-foreground">{emptyDescription}</p>
          </div>
        ) : null}

        {!jobsQuery.isLoading && !jobsQuery.error && jobs.length > 0 ? (
          <div className="space-y-3">
            {jobs.map((job) => {
              const detail = getJobDetailSummary(job)
              return (
                <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border/70 bg-background/70 p-4" key={`${job.queue}-${job.id}`}>
                  <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div className="space-y-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <p className="font-mono text-xs text-muted-foreground">{job.id}</p>
                        <JobStatusBadge status={job.status} />
                        <Badge variant="muted">{job.job_name}</Badge>
                      </div>
                      <p className="text-sm text-foreground">{detail.title}</p>
                      <p className="text-xs text-muted-foreground">{detail.meta}</p>
                      {detail.filterSummary ? <p className="text-xs text-muted-foreground">{detail.filterSummary}</p> : null}
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button onClick={() => setSelectedJob(job)} size="sm" type="button" variant="outline">
                        {t("jobs.actions.details", "Detalhes")}
                      </Button>
                      {(job.status === "failed" || job.status === "dead" || job.status === "succeeded") ? (
                        <Button
                          disabled={retryMutation.isPending}
                          onClick={() => retryMutation.mutate({ jobID: job.id, queue: job.queue })}
                          size="sm"
                          type="button"
                          variant="outline"
                        >
                          {t("jobs.actions.retry", "Reenfileirar")}
                        </Button>
                      ) : null}
                      {job.status === "canceled" ? (
                        <Button disabled={resumeMutation.isPending} onClick={() => resumeMutation.mutate({ jobID: job.id, queue: job.queue })} size="sm" type="button" variant="outline">
                          {t("jobs.actions.resume", "Retomar")}
                        </Button>
                      ) : null}
                      {(job.status === "queued" || job.status === "delayed" || job.status === "running") ? (
                        <Button
                          className="border-destructive/40 text-destructive hover:bg-destructive/10"
                          disabled={cancelMutation.isPending}
                          onClick={() => cancelMutation.mutate({ jobID: job.id, queue: job.queue })}
                          size="sm"
                          type="button"
                          variant="outline"
                        >
                          {t("jobs.actions.cancel", "Cancelar")}
                        </Button>
                      ) : null}
                    </div>
                  </div>
                  <div className="mt-4 grid gap-3 md:grid-cols-4">
                    <Stat label={t("jobs.card.queue", "Fila")} value={job.queue} />
                    <Stat label={t("jobs.card.priority", "Prioridade")} value={job.priority} />
                    <Stat label={t("jobs.card.attempts", "Tentativas")} value={`${job.attempts}/${job.max_attempts}`} />
                    <Stat label={t("jobs.card.createdAt", "Criado em")} value={formatTimestamp(job.created_at)} />
                  </div>
                  {job.last_error ? (
                    <div className="mt-4 flex items-start gap-2 rounded-[calc(var(--radius)-0.25rem)] border border-destructive/30 bg-destructive/5 px-3 py-3 text-sm text-destructive">
                      <XCircle className="mt-0.5 size-4 shrink-0" />
                      <p>{job.last_error}</p>
                    </div>
                  ) : null}
                </div>
              )
            })}
          </div>
        ) : null}
      </CardContent>

      <Dialog onOpenChange={(open) => !open && setSelectedJob(null)} open={Boolean(selectedJob)}>
        <DialogContent className="max-w-4xl">
          <DialogHeader>
            <DialogTitle>{t("jobs.dialog.title", "Detalhes do trabalho")}</DialogTitle>
            <DialogDescription>{selected ? `${selected.job_name} · ${selected.id}` : ""}</DialogDescription>
          </DialogHeader>
          {selected ? <JobDialogContent job={selected} loading={detailQuery.isFetching} /> : null}
        </DialogContent>
      </Dialog>
    </Card>
  )
}

function JobDialogContent({ job, loading }: { job: AdminJob; loading?: boolean }) {
  const { t } = useTranslation()
  return (
    <div className="space-y-4">
      {loading ? <Skeleton className="h-24 w-full" /> : null}
      <div className="grid gap-3 md:grid-cols-2">
        <Stat label={t("jobs.card.startedAt", "Iniciado em")} value={formatTimestamp(job.started_at)} />
        <Stat label={t("jobs.card.finishedAt", "Finalizado em")} value={formatTimestamp(job.finished_at)} />
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <JsonPanel content={job.payload} title={t("jobs.dialog.payload", "Payload")}/>
        <JsonPanel content={job.result} title={t("jobs.dialog.result", "Resultado")}/>
      </div>
    </div>
  )
}

function JsonPanel({ title, content }: { title: string; content: unknown }) {
  return (
    <div className="space-y-2">
      <p className="text-xs uppercase tracking-[0.14em] text-muted-foreground">{title}</p>
      <pre className="max-h-[52vh] overflow-auto rounded-[calc(var(--radius)-0.25rem)] border border-border bg-muted/30 p-4 text-xs text-foreground">{content ? JSON.stringify(content, null, 2) : "{}"}</pre>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border/70 bg-card/80 px-3 py-3">
      <p className="text-xs uppercase tracking-[0.14em] text-muted-foreground">{label}</p>
      <p className="mt-1 text-sm text-foreground">{value || "-"}</p>
    </div>
  )
}

function JobsBoardSkeleton() {
  return (
    <div className="space-y-3">
      <Skeleton className="h-32 w-full" />
      <Skeleton className="h-32 w-full" />
    </div>
  )
}

function JobStatusBadge({ status }: { status: AdminJobStatus }) {
  const { t } = useTranslation()
  if (status === "queued" || status === "running" || status === "delayed") {
    return <Badge variant="warning"><Clock3 className="size-3" />{t(`jobs.status.${status}`, status)}</Badge>
  }
  if (status === "succeeded") {
    return <Badge variant="success">{t("jobs.status.succeeded", "succeeded")}</Badge>
  }
  if (status === "failed" || status === "dead" || status === "canceled") {
    return <Badge variant="danger">{t(`jobs.status.${status}`, status)}</Badge>
  }
  return <Badge variant="muted">{t("jobs.status.unknown", "unknown")}</Badge>
}

function getJobDetailSummary(job: AdminJob) {
  const filterSummary = summarizeJobFilter(job)
  if (job.job_name === "download.events") {
    const relayCount = Array.isArray(job.payload?.request?.relays) ? job.payload.request.relays.length : 0
    return {
      title: relayCount > 0 ? `${relayCount} relays preparados para importação` : "Download operacional em fila",
      meta: `queue=${job.queue} · status=${job.status}`,
      filterSummary,
    }
  }
  if (job.job_name === "sync.negentropy") {
    return {
      title: job.payload?.remote ? `Sync com ${job.payload.remote}` : "Sincronização Negentropy",
      meta: `direction=${job.payload?.direction ?? "both"} · queue=${job.queue}`,
      filterSummary,
    }
  }
  return {
    title: job.job_name,
    meta: `queue=${job.queue} · status=${job.status}`,
    filterSummary,
  }
}

function summarizeJobFilter(job: AdminJob) {
  const payload = job.payload as { filter?: unknown; filter_json?: string; request?: { filter?: unknown } } | undefined
  const rawFilter = payload?.filter ?? payload?.request?.filter ?? payload?.filter_json
  if (!rawFilter) {
    return ""
  }
  if (typeof rawFilter === "string") {
    return rawFilter
  }
  try {
    return JSON.stringify(rawFilter)
  } catch {
    return ""
  }
}

function formatTimestamp(value?: string) {
  if (!value) {
    return "-"
  }
  return new Date(value).toLocaleString()
}
