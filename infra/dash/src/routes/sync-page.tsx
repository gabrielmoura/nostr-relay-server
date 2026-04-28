import { useState, Suspense } from "react"
import { useTranslation } from "react-i18next"
import { RefreshCcw } from "lucide-react"
import { toast } from "sonner"

import { JobsBoard } from "@/components/features/jobs/jobs-board"
import { PageHeader } from "@/components/shared/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useSyncMutation } from "@/hooks/use-admin-data"
import { NostrFilterBuilder, type NostrFilter } from "@/components/features/filters/nostr-filter-builder"
import { Skeleton } from "@/components/ui/skeleton"

export function SyncPage() {
  const { t } = useTranslation()
  const syncMutation = useSyncMutation()

  const [remote, setRemote] = useState("")
  const [filter, setFilter] = useState<NostrFilter>({ kinds: [1], limit: 100 })
  const [direction, setDirection] = useState<"both" | "up" | "down">("both")

  const handleSync = async () => {
    if (!remote.trim()) {
      toast.error(t("sync.errorRemoteRequired", "Remote relay URL is required"))
      return
    }

    try {
      const response = await syncMutation.mutateAsync({
        remote: remote.trim(),
        direction,
        filter: [filter],
      })
      toast.success(
        response.job_id
          ? t("sync.successStartedWithJob", { jobId: response.job_id, defaultValue: `Sync initiated. Job ${response.job_id}` })
          : t("sync.successStarted", "Sync process started in background"),
      )
    } catch (error) {
      console.error("Sync failed:", error)
      toast.error(t("common.error"))
    }
  }

  return (
    <Suspense fallback={<SyncSkeleton />}>
      <div className="space-y-6">
        <PageHeader
          description={t("sync.description")}
          title={t("sync.title")}
        />

        <div className="grid gap-6 lg:grid-cols-2">
          <Card className="panel-shadow">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <RefreshCcw className="size-5" />
                {t("sync.configTitle", "Configuration")}
              </CardTitle>
              <CardDescription>{t("sync.configDescription", "Configure the negentropy sync process.")}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>{t("sync.remoteRelay", "Remote Relay URL")}</Label>
                <Input
                  onChange={(e) => setRemote(e.target.value)}
                  placeholder="wss://relay.damus.io"
                  value={remote}
                />
              </div>

              <div className="space-y-2">
                <Label>{t("sync.direction", "Direction")}</Label>
                <select
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  onChange={(e) => setDirection(e.target.value as "both" | "up" | "down")}
                  value={direction}
                >
                  <option value="both">{t("sync.directionBoth", "Both (Bidirectional)")}</option>
                  <option value="up">{t("sync.directionUp", "Up (Local -> Remote)")}</option>
                  <option value="down">{t("sync.directionDown", "Down (Remote -> Local)")}</option>
                </select>
              </div>

              <div className="pt-4">
                <Button className="w-full" disabled={syncMutation.isPending} onClick={handleSync}>
                  {syncMutation.isPending ? <RefreshCcw className="mr-2 size-4 animate-spin" /> : <RefreshCcw className="mr-2 size-4" />}
                  {t("sync.actionStart", "Start Sync")}
                </Button>
              </div>
            </CardContent>
          </Card>

          <NostrFilterBuilder
            initialFilter={filter}
            onChange={setFilter}
            title={t("sync.filterTitle", "Sync Scope")}
          />
        </div>

        <JobsBoard
          description={t("jobs.sync.description", "Acompanhe sincronizações Negentropy com estado durável, tentativas e falhas terminais.")}
          emptyDescription={t("jobs.sync.emptyDescription", "Nenhuma sincronização foi enfileirada ainda.")}
          emptyTitle={t("jobs.sync.emptyTitle", "Sem sincronizações na fila")}
          filters={{ job_name: "sync.negentropy" }}
          title={t("jobs.sync.title", "Trabalhos de sincronização")}
        />
      </div>
    </Suspense>
  )
}

function SyncSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-24 w-full" />
      <div className="grid gap-6 lg:grid-cols-2">
        <Skeleton className="h-80 w-full" />
        <Skeleton className="h-80 w-full" />
      </div>
    </div>
  )
}
