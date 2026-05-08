import { Suspense, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { Download, Radio, Settings2 } from "lucide-react"
import { toast } from "sonner"

import { NostrFilterBuilder, type NostrFilter } from "@/components/features/filters/nostr-filter-builder"
import { JobsBoard } from "@/components/features/jobs/jobs-board"
import { PageHeader } from "@/components/shared/page-header"
import { RelayListModal } from "@/components/shared/relay-list-modal"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { useDownloadMutation, useJobsQuery } from "@/hooks/use-admin-data"
import { readRelayStorage } from "@/lib/relay-presets"

export function DownloadPage() {
  const { t } = useTranslation()
  const downloadMutation = useDownloadMutation()
  const jobsQuery = useJobsQuery({ job_name: "download.events", limit: 30 })

  const [relays, setRelays] = useState<string[]>(() => readRelayStorage("external-relays"))
  const [filter, setFilter] = useState<NostrFilter>({ limit: 100 })
  const [timeout, setTimeoutVal] = useState("60")
  const [relayModalOpen, setRelayModalOpen] = useState(false)

  const activeJobs = useMemo(
    () => (jobsQuery.data?.items ?? []).filter((job) => job.status === "queued" || job.status === "running"),
    [jobsQuery.data],
  )

  const handleStartDownload = async () => {
    if (relays.length === 0) {
      toast.error(t("download.relaysRequired", "Pelo menos um relay é necessário"))
      return
    }

    try {
      const response = await downloadMutation.mutateAsync({
        relays,
        filter,
        timeout: parseInt(timeout, 10),
      })

      toast.success(
        t("download.startedWithJob", {
          jobId: response.job_id,
          defaultValue: `Download iniciado. Job ${response.job_id}`,
        }),
      )
    } catch (error) {
      console.error("Download failed:", error)
      toast.error(t("common.error"))
    }
  }

  return (
    <Suspense fallback={<DownloadSkeleton />}>
      <div className="space-y-6">
        <PageHeader description={t("download.description")} title={t("download.title")} />

        {downloadMutation.isPending || activeJobs.length > 0 ? (
          <div className="flex items-start gap-3 rounded-[var(--radius)] border border-primary/20 bg-primary/5 px-4 py-4 text-sm text-foreground panel-shadow">
            <Radio className="mt-0.5 size-4 shrink-0 animate-pulse text-primary" />
            <div>
              <p className="font-medium">{t("download.activityTitle", "Trabalho em progresso")}</p>
              <p className="text-muted-foreground">
                {activeJobs.length > 0
                  ? t("download.activityDescription", {
                      count: activeJobs.length,
                      defaultValue: `${activeJobs.length} trabalho(s) de download em andamento.`,
                    })
                  : t("download.activityStarting", "Preparando o envio do novo trabalho para o backend.")}
              </p>
            </div>
          </div>
        ) : null}

        <div className="grid gap-6 lg:grid-cols-2">
          <Card className="panel-shadow">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Download className="size-5" />
                {t("download.configTitle", "Configuração")}
              </CardTitle>
              <CardDescription>
                {t("download.configDescription", "Baixe eventos de outros relays para o seu banco de dados local.")}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="relays">{t("download.relaysLabel", "Relays Remotos")}</Label>
                <div className="flex gap-2">
                  <Input id="relays" placeholder="wss://relay.damus.io, wss://nos.lol" readOnly value={relays.join(", ")} />
                  <Button onClick={() => setRelayModalOpen(true)} type="button" variant="outline">
                    <Settings2 className="size-4" />
                    {t("download.manageRelays", "Gerir relays")}
                  </Button>
                </div>
                <div className="flex flex-wrap gap-2">
                  {relays.map((relay) => (
                    <Badge key={relay} variant="muted">
                      {relay}
                    </Badge>
                  ))}
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="timeout">{t("download.timeoutLabel", "Timeout (segundos)")}</Label>
                <Input id="timeout" max={600} min={5} onChange={(e) => setTimeoutVal(e.target.value)} type="number" value={timeout} />
              </div>

              <div className="pt-4">
                <Button className="w-full" disabled={downloadMutation.isPending} onClick={handleStartDownload}>
                  <Download className={`mr-2 size-4 ${downloadMutation.isPending ? "animate-spin" : ""}`} />
                  {t("download.startAction", "Iniciar Download")}
                </Button>
              </div>
            </CardContent>
          </Card>

          <NostrFilterBuilder initialFilter={filter} onChange={setFilter} title={t("download.filterTitle", "Escopo do Download")} />
        </div>

        <JobsBoard
          description={t("jobs.download.description", "Downloads enfileirados, em andamento, concluídos ou falhos com estado real do backend.")}
          emptyDescription={t("jobs.download.emptyDescription", "Nenhum download operacional foi disparado ainda.")}
          emptyTitle={t("jobs.download.emptyTitle", "Sem downloads na fila")}
          filters={{ job_name: "download.events" }}
          title={t("jobs.download.title", "Trabalhos de download")}
        />

        <RelayListModal
          description={t(
            "download.relaysModalDescription",
            "Adicione relays individualmente ou importe uma lista. A seleção é salva localmente para reutilização em outras telas.",
          )}
          onApply={setRelays}
          onOpenChange={setRelayModalOpen}
          open={relayModalOpen}
          storageScope="external-relays"
          submitLabel={t("download.saveRelays", "Salvar relays")}
          title={t("download.relaysModalTitle", "Relays para download")}
          value={relays}
        />
      </div>
    </Suspense>
  )
}

function DownloadSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-24 w-full" />
      <div className="grid gap-6 lg:grid-cols-2">
        <Skeleton className="h-80 w-full" />
        <Skeleton className="h-80 w-full" />
      </div>
      <Skeleton className="h-72 w-full" />
    </div>
  )
}
