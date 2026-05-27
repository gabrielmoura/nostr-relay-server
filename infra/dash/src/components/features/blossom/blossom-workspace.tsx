import { useEffect, useMemo, useState } from "react"
import { Link } from "@tanstack/react-router"
import { AlertTriangle, BarChart3, CopyPlus, ImageIcon, List, RefreshCw, ShieldCheck, Workflow } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { BlossomAnalyticsDialog } from "@/components/features/blossom/blossom-analytics-dialog"
import { BlossomKpiStrip } from "@/components/features/blossom/blossom-kpi-strip"
import { BlossomObjectSheet } from "@/components/features/blossom/blossom-object-sheet"
import { BlossomUsersTable } from "@/components/features/blossom/blossom-users-table"
import { BlossomWorkersDialog } from "@/components/features/blossom/blossom-workers-dialog"
import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  useBlossomAnalytics,
  useBlossomBulkReviewMutation,
  useBlossomOverview,
  useBlossomPolicy,
  useBlossomWorkers,
  useCreateBlossomMirrorMutation,
  useInfiniteBlossomObjects,
  useInfiniteBlossomReports,
  useInfiniteBlossomUsers,
  usePurgeBlossomUserMutation,
  useUpsertBlossomWhitelistMutation,
} from "@/hooks/use-admin-data"
import { blossomExifVariant, blossomReviewVariant, type BlossomRouteSearch } from "@/lib/blossom"
import { normalizeFilterIdentifier } from "@/lib/nostr"
import { formatBytes, formatDateTime, shortenId } from "@/lib/utils"
import { useBlossomOperatorStore } from "@/stores/blossom-operator-store"
import type { BlossomLibraryView, BlossomReviewState, BlossomTab, BlossomWorkerRecord } from "@/types/admin"

const allValue = "__all__"

type BlossomWorkspaceProps = {
  search: BlossomRouteSearch
  onSearchChange: (patch: Partial<BlossomRouteSearch>) => void
}

export function BlossomWorkspace({ search, onSearchChange }: BlossomWorkspaceProps) {
  const { t } = useTranslation()
  const [selectedHash, setSelectedHash] = useState("")
  const [selectedHashes, setSelectedHashes] = useState<string[]>([])
  const [workersModalOpen, setWorkersModalOpen] = useState(false)
  const [analyticsOpen, setAnalyticsOpen] = useState(false)
  const recentMimeFilters = useBlossomOperatorStore((state) => state.recentMimeFilters)
  const mirrorDraft = useBlossomOperatorStore((state) => state.mirrorDraft)
  const mirrorHistory = useBlossomOperatorStore((state) => state.mirrorHistory)
  const mirrorHistoryLoaded = useBlossomOperatorStore((state) => state.mirrorHistoryLoaded)
  const addRecentMimeFilter = useBlossomOperatorStore((state) => state.addRecentMimeFilter)
  const setMirrorDraft = useBlossomOperatorStore((state) => state.setMirrorDraft)
  const resetMirrorDraft = useBlossomOperatorStore((state) => state.resetMirrorDraft)
  const hydrateMirrorHistory = useBlossomOperatorStore((state) => state.hydrateMirrorHistory)
  const addMirrorHistory = useBlossomOperatorStore((state) => state.addMirrorHistory)
  const clearMirrorHistory = useBlossomOperatorStore((state) => state.clearMirrorHistory)

  const activeTab = search.tab ?? "overview"
  const view = search.view ?? "table"

  const overviewQuery = useBlossomOverview()
  const policyQuery = useBlossomPolicy()
  const analyticsQuery = useBlossomAnalytics(analyticsOpen)
  const objectsQuery = useInfiniteBlossomObjects({
    q: search.q,
    sha256: search.sha256,
    mime_type: search.mimeType,
    extension: search.extension,
    review_state: search.reviewState,
    pubkey: search.pubkey,
    uploader_q: search.uploaderQuery,
  })
  const usersQuery = useInfiniteBlossomUsers({ q: search.userQuery, sort_by: search.userSortBy, sort_dir: search.userSortDir })
  const workersQuery = useBlossomWorkers({ status: search.workerStatus })
  const reportsQuery = useInfiniteBlossomReports({ q: search.reportQuery, report_type: search.reportType, status: search.reportStatus })

  const reviewMutation = useBlossomBulkReviewMutation()
  const whitelistMutation = useUpsertBlossomWhitelistMutation()
  const purgeMutation = usePurgeBlossomUserMutation()
  const mirrorMutation = useCreateBlossomMirrorMutation()

  const alerts = overviewQuery.data?.alerts ?? []
  const objects = useMemo(() => objectsQuery.data?.pages.flatMap((page) => page.items) ?? [], [objectsQuery.data])
  const users = useMemo(() => usersQuery.data?.pages.flatMap((page) => page.items) ?? [], [usersQuery.data])
  const reports = useMemo(() => reportsQuery.data?.pages.flatMap((page) => page.items) ?? [], [reportsQuery.data])
  const reviewItems = useMemo(() => objects.filter((item) => item.review_state === "flagged" || item.review_state === "pending_review"), [objects])
  const mimeOptions = useMemo(() => [...new Set([...objects.map((item) => item.mime_type).filter(Boolean), ...recentMimeFilters])], [objects, recentMimeFilters])
  const reviewEnabled = (policyQuery.data?.mode ?? overviewQuery.data?.policy?.mode) === "mandatory_review"

  useEffect(() => {
    if (!mirrorHistoryLoaded) {
      void hydrateMirrorHistory()
    }
  }, [hydrateMirrorHistory, mirrorHistoryLoaded])

  const loading = overviewQuery.isLoading && objects.length === 0
  const error = overviewQuery.error ?? objectsQuery.error

  const runBulkAction = async (hashes: string[], action: "approve" | "hard_delete" | "requeue_optimization") => {
    if (hashes.length === 0) {
      return
    }
    try {
      await reviewMutation.mutateAsync({ hashes, action, reason: "dashboard action" })
      setSelectedHashes((current) => current.filter((item) => !hashes.includes(item)))
      toast.success(t("blossom.bulk.success", "Ação aplicada com sucesso."))
    } catch (mutationError) {
      toast.error(mutationError instanceof Error ? mutationError.message : t("common.error"))
    }
  }

  if (loading) {
    return <LoadingPanel label={t("blossom.loading", "Carregando biblioteca Blossom...")} />
  }

  if (error || !overviewQuery.data) {
    return <ErrorPanel description={t("blossom.errorDescription", "Não foi possível carregar o workspace Blossom.")} onRetry={() => { void overviewQuery.refetch(); void objectsQuery.refetch() }} title={t("blossom.errorTitle", "Falha ao carregar Blossom")} />
  }

  return (
    <div className="space-y-6">
      <PageHeader
        actions={
          <>
            <Button onClick={() => void overviewQuery.refetch()} type="button" variant="outline">
              <RefreshCw className="size-4" />
              {t("jobs.actions.refresh", "Atualizar")}
            </Button>
            <Button onClick={() => setAnalyticsOpen(true)} type="button" variant="outline">
              <BarChart3 className="size-4" />
              {t("blossom.header.analytics", "Análises")}
            </Button>
            <Button onClick={() => setWorkersModalOpen(true)} type="button" variant="outline">
              <Workflow className="size-4" />
              {t("blossom.header.workers", "Workers")}
            </Button>
            <Button asChild type="button" variant="outline">
              <Link to="/blossom/plans">{t("blossom.header.plans", "Planos e cotas")}</Link>
            </Button>
          </>
        }
        className="rounded-[var(--radius)] border border-primary/15 bg-[linear-gradient(135deg,rgba(14,165,233,0.08),rgba(34,197,94,0.08))] p-5 panel-shadow"
        description={t("blossom.description", "Gerencie arquivos Blossom com revisão, cotas, mirroring, workers de mídia e trilha de auditoria.")}
        title={t("blossom.title", "Servidor Blossom")}
      />

      <BlossomKpiStrip overview={overviewQuery.data} />

      {alerts.length > 0 ? (
        <div className="grid gap-3 xl:grid-cols-2">
          {alerts.map((alert) => (
            <Card className="border-border/70 bg-card/95" key={alert.code}>
              <CardContent className="flex items-start gap-3 p-4">
                <div className="rounded-full bg-orange-100 p-2 text-orange-700">
                  <AlertTriangle className="size-4" />
                </div>
                <div>
                  <p className="font-medium text-foreground">{alert.message}</p>
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">{alert.code}</p>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : null}

      <Card>
        <CardHeader className="pb-3">
          <CardTitle>{t("blossom.filters.title", "Filtros operacionais")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid gap-3 lg:grid-cols-2 xl:grid-cols-5">
            <Input onChange={(event) => onSearchChange({ q: event.target.value || undefined })} placeholder={t("blossom.filters.search", "Buscar hash, pubkey, MIME ou estado")} value={search.q ?? ""} />
            <Input onChange={(event) => onSearchChange({ sha256: event.target.value || undefined })} placeholder={t("blossom.filters.sha256", "Busca exata por SHA-256")} value={search.sha256 ?? ""} />
            <Input list="blossom-mime-options" onBlur={(event) => addRecentMimeFilter(event.target.value)} onChange={(event) => onSearchChange({ mimeType: event.target.value || undefined })} placeholder={t("blossom.filters.mime", "Tipo MIME")} value={search.mimeType ?? ""} />
            <datalist id="blossom-mime-options">
              {mimeOptions.map((item) => <option key={item} value={item} />)}
            </datalist>
            <Select onValueChange={(value) => onSearchChange({ reviewState: value === allValue ? undefined : (value as BlossomReviewState) })} value={search.reviewState ?? allValue}>
              <SelectTrigger><SelectValue placeholder={t("blossom.filters.reviewState", "Estado de revisão")} /></SelectTrigger>
              <SelectContent>
                <SelectItem value={allValue}>{t("blossom.filters.allStates", "Todos os estados")}</SelectItem>
                {(["ready", "flagged", "pending_review", "approved", "deleted"] as BlossomReviewState[]).map((item) => <SelectItem key={item} value={item}>{item}</SelectItem>)}
              </SelectContent>
            </Select>
            <Input onChange={(event) => onSearchChange({ uploaderQuery: normalizeFilterIdentifier("generic", event.target.value) || undefined })} placeholder={t("blossom.filters.uploader", "Uploader: nome, npub ou hex")} value={search.uploaderQuery ?? ""} />
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Button onClick={() => onSearchChange({ view: view === "table" ? "grid" : "table", tab: "library" })} type="button" variant="outline">
              {view === "table" ? <ImageIcon className="size-4" /> : <List className="size-4" />}
              {view === "table" ? t("blossom.views.grid", "Grade") : t("blossom.views.table", "Tabela")}
            </Button>
            {selectedHashes.length > 0 ? (
              <>
                <Badge variant="warning">{selectedHashes.length} {t("blossom.bulk.selected", "selecionado(s)")}</Badge>
                <Button onClick={() => void runBulkAction(selectedHashes, "approve")} size="sm" type="button"><ShieldCheck className="size-4" />{t("blossom.bulk.approve", "Aprovar")}</Button>
                <Button onClick={() => void runBulkAction(selectedHashes, "hard_delete")} size="sm" type="button" variant="destructive">{t("blossom.bulk.delete", "Excluir")}</Button>
              </>
            ) : null}
          </div>
        </CardContent>
      </Card>

      <Tabs onValueChange={(value) => onSearchChange({ tab: value as BlossomTab })} value={activeTab}>
        <TabsList className="flex h-auto flex-wrap">
          <TabsTrigger value="overview">{t("blossom.tabs.overview", "Visão geral")}</TabsTrigger>
          <TabsTrigger value="library">{t("blossom.tabs.library", "Biblioteca")}</TabsTrigger>
          <TabsTrigger value="users">{t("blossom.tabs.users", "Usuários")}</TabsTrigger>
          <TabsTrigger value="workers">{t("blossom.tabs.workers", "Workers")}</TabsTrigger>
        </TabsList>

        <TabsContent className="mt-4 space-y-4" value="overview">
          <PolicySummaryCard
            mode={policyQuery.data?.mode ?? overviewQuery.data.policy?.mode ?? "free"}
          />
          <OverviewCards flaggedCount={reviewItems.length} reportCount={reports.length} usersCount={users.length} workersCount={workersQuery.data?.length ?? 0} />
          <SubscreenLinks reportCount={reports.length} reviewCount={reviewItems.length} reviewEnabled={reviewEnabled} />
          <LibraryTable items={objects.slice(0, 4)} onOpen={setSelectedHash} onToggleSelection={toggleHashSelection(setSelectedHashes)} selectedHashes={selectedHashes} />
        </TabsContent>

        <TabsContent className="mt-4 space-y-4" value="library">
          {objects.length === 0 ? <EmptyPanel description={t("blossom.emptyDescription", "Nenhum arquivo corresponde aos filtros atuais.")} title={t("blossom.emptyTitle", "Biblioteca vazia")} /> : null}
          {objects.length > 0 && view === "table" ? <LibraryTable items={objects} onOpen={setSelectedHash} onToggleSelection={toggleHashSelection(setSelectedHashes)} selectedHashes={selectedHashes} /> : null}
          {objects.length > 0 && view === "grid" ? <LibraryGrid items={objects} onOpen={setSelectedHash} onToggleSelection={toggleHashSelection(setSelectedHashes)} selectedHashes={selectedHashes} /> : null}
          {objectsQuery.hasNextPage ? <Button onClick={() => void objectsQuery.fetchNextPage()} type="button" variant="outline">{t("blossom.loadMore", "Carregar mais")}</Button> : null}
        </TabsContent>

        <TabsContent className="mt-4 space-y-4" value="users">
          <Input onChange={(event) => onSearchChange({ userQuery: normalizeFilterIdentifier("generic", event.target.value) || undefined })} placeholder={t("blossom.users.search", "Filtrar nome, nip05, npub, pubkey ou notas")} value={search.userQuery ?? ""} />
          <BlossomUsersTable
            fetchNextPage={() => void usersQuery.fetchNextPage()}
            hasNextPage={usersQuery.hasNextPage}
            isFetchingNextPage={usersQuery.isFetchingNextPage}
            items={users}
            onPurge={async (pubkey) => {
              try {
                await purgeMutation.mutateAsync(pubkey)
                toast.success(t("blossom.users.purgeSuccess", "Purge enfileirado com sucesso."))
              } catch (mutationError) {
                toast.error(mutationError instanceof Error ? mutationError.message : t("common.error"))
              }
            }}
            onSave={async (pubkey, enabled) => {
              try {
                await whitelistMutation.mutateAsync({ pubkey, enabled })
                toast.success(t("blossom.users.quotaSaved", "Permissões atualizadas."))
              } catch (mutationError) {
                toast.error(mutationError instanceof Error ? mutationError.message : t("common.error"))
              }
            }}
            onSortChange={(sortBy, sortDir) => {
              onSearchChange({ userSortBy: sortBy, userSortDir: sortDir })
            }}
            policyMode={policyQuery.data?.mode ?? "free"}
            sortBy={search.userSortBy}
            sortDir={search.userSortDir}
          />
        </TabsContent>

        <TabsContent className="mt-4 space-y-4" value="workers">
          <Card>
            <CardHeader><CardTitle>{t("blossom.workers.mirrorTitle", "Gerenciador de mirroring")}</CardTitle></CardHeader>
            <CardContent className="grid gap-3 lg:grid-cols-[1.4fr_1fr_auto]">
              <Input onChange={(event) => setMirrorDraft({ sourceURL: event.target.value })} placeholder={t("blossom.workers.mirrorUrl", "URL remota do arquivo")} value={mirrorDraft.sourceURL} />
              <Input onChange={(event) => setMirrorDraft({ expectedSHA256: event.target.value })} placeholder={t("blossom.workers.mirrorHash", "SHA-256 esperado")} value={mirrorDraft.expectedSHA256} />
              <Button onClick={async () => {
                try {
                  const response = await mirrorMutation.mutateAsync({ source_url: mirrorDraft.sourceURL, expected_sha256: mirrorDraft.expectedSHA256 })
                  toast.success(t("blossom.workers.mirrorQueued", "Mirror enfileirado") + ` ${response.job_id}`)
                  await addMirrorHistory({
                    id: response.job_id,
                    source_url: mirrorDraft.sourceURL,
                    expected_sha256: mirrorDraft.expectedSHA256,
                    job_id: response.job_id,
                    status: response.status,
                    created_at: new Date().toISOString(),
                  })
                  resetMirrorDraft()
                } catch (mutationError) {
                  toast.error(mutationError instanceof Error ? mutationError.message : t("common.error"))
                }
              }} type="button"><CopyPlus className="size-4" />{t("blossom.workers.mirrorAction", "Espelhar")}</Button>
            </CardContent>
          </Card>
          <MirrorHistoryCard history={mirrorHistory} onClear={() => { void clearMirrorHistory() }} />
          <WorkersTable items={workersQuery.data ?? []} />
        </TabsContent>

      </Tabs>

      <BlossomObjectSheet hash={selectedHash} onApprove={(hash) => void runBulkAction([hash], "approve")} onDelete={(hash) => void runBulkAction([hash], "hard_delete")} onOpenChange={(open) => { if (!open) setSelectedHash("") }} onRequeue={(hash) => void runBulkAction([hash], "requeue_optimization")} open={Boolean(selectedHash)} />
      <BlossomWorkersDialog items={workersQuery.data ?? []} onOpenChange={setWorkersModalOpen} onRefresh={() => { void workersQuery.refetch() }} open={workersModalOpen} />
      <BlossomAnalyticsDialog analytics={analyticsQuery.data} onOpenChange={setAnalyticsOpen} open={analyticsOpen} />
    </div>
  )
}

function toggleHashSelection(setter: React.Dispatch<React.SetStateAction<string[]>>) {
  return (hash: string) => {
    setter((current) => (current.includes(hash) ? current.filter((item) => item !== hash) : [...current, hash]))
  }
}

function OverviewCards({ flaggedCount, usersCount, workersCount, reportCount }: { flaggedCount: number; usersCount: number; workersCount: number; reportCount: number }) {
  const { t } = useTranslation()
  const cards = [
    { key: "review", label: t("blossom.overview.review", "Itens críticos"), value: String(flaggedCount) },
    { key: "reports", label: t("blossom.overview.reports", "Reports ativos"), value: String(reportCount) },
    { key: "users", label: t("blossom.overview.users", "Usuários monitorados"), value: String(usersCount) },
    { key: "workers", label: t("blossom.overview.workers", "Jobs visíveis"), value: String(workersCount) },
  ]
  return <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">{cards.map((card) => <Card key={card.key}><CardContent className="p-4"><p className="text-sm text-muted-foreground">{card.label}</p><p className="mt-2 font-heading text-3xl font-semibold">{card.value}</p></CardContent></Card>)}</div>
}

function PolicySummaryCard({ mode }: { mode: string }) {
  const { t } = useTranslation()
  const label =
    mode === "mandatory_review"
      ? t("blossom.policy.modeMandatoryLabel", "Revisao obrigatoria")
      : mode === "enabled_users"
        ? t("blossom.policy.modeEnabledLabel", "Somente habilitados")
        : t("blossom.policy.modeFreeLabel", "Livre")

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-3">
        <CardTitle>{t("blossom.policy.title", "Política de uploads")}</CardTitle>
        <Button asChild type="button" variant="outline">
          <Link to="/blossom/policy">{t("blossom.policy.configure", "Configurar politica")}</Link>
        </Button>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="rounded-[var(--radius)] border border-primary/20 bg-primary/5 p-4">
          <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">{t("blossom.policy.activeMode", "Modo ativo")}</p>
          <p className="mt-2 text-lg font-semibold text-foreground">{label}</p>
          <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
            {mode === "mandatory_review"
              ? t("blossom.policy.modeMandatoryDesc", "Uploads ficam retidos ate aprovacao manual do operador.")
              : mode === "enabled_users"
                ? t("blossom.policy.modeEnabledDesc", "Apenas usuarios marcados como habilitados podem enviar arquivos.")
                : t("blossom.policy.modeFreeDesc", "Qualquer usuario pode fazer upload respeitando as cotas aplicadas.")}
          </p>
        </div>
      </CardContent>
    </Card>
  )
}

function LibraryTable({ items, selectedHashes, onToggleSelection, onOpen }: { items: Array<any>; selectedHashes: string[]; onToggleSelection: (hash: string) => void; onOpen: (hash: string) => void }) {
  const { t } = useTranslation()
  return (
    <Card>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("common.actions", "Ações")}</TableHead>
            <TableHead>SHA-256</TableHead>
            <TableHead>{t("blossom.table.mime", "MIME")}</TableHead>
            <TableHead>{t("blossom.table.size", "Tamanho")}</TableHead>
            <TableHead>{t("blossom.table.createdAt", "Data")}</TableHead>
            <TableHead>{t("blossom.table.review", "Revisão")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((item) => (
            <TableRow key={item.hash}>
              <TableCell><Button onClick={() => onToggleSelection(item.hash)} size="sm" type="button" variant={selectedHashes.includes(item.hash) ? "default" : "outline"}>{selectedHashes.includes(item.hash) ? "OK" : "+"}</Button></TableCell>
              <TableCell><button className="cursor-pointer text-left font-mono text-xs text-primary" onClick={() => onOpen(item.hash)} type="button">{shortenId(item.hash, 14, 8)}</button></TableCell>
              <TableCell>{item.mime_type}</TableCell>
              <TableCell>{formatBytes(item.size)}</TableCell>
              <TableCell>{formatDateTime(item.created_at)}</TableCell>
              <TableCell><div className="flex flex-wrap gap-2"><Badge variant={blossomReviewVariant(item.review_state)}>{item.review_state}</Badge><Badge variant={blossomExifVariant(item.exif_status)}>{item.exif_status}</Badge></div></TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Card>
  )
}

function LibraryGrid({ items, selectedHashes, onToggleSelection, onOpen }: { items: Array<any>; selectedHashes: string[]; onToggleSelection: (hash: string) => void; onOpen: (hash: string) => void }) {
  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      {items.map((item) => (
        <Card className="overflow-hidden" key={item.hash}>
          {item.thumbnail_url ? <img alt={item.hash} className="aspect-video w-full object-cover" src={item.thumbnail_url} /> : null}
          <CardContent className="space-y-3 p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0"><p className="font-mono text-xs break-all">{shortenId(item.hash, 14, 8)}</p><p className="text-sm text-muted-foreground">{item.mime_type}</p></div>
              <Button onClick={() => onToggleSelection(item.hash)} size="sm" type="button" variant={selectedHashes.includes(item.hash) ? "default" : "outline"}>{selectedHashes.includes(item.hash) ? "OK" : "+"}</Button>
            </div>
            <div className="flex flex-wrap gap-2"><Badge variant={blossomReviewVariant(item.review_state)}>{item.review_state}</Badge><Badge variant={blossomExifVariant(item.exif_status)}>{item.exif_status}</Badge></div>
            <div className="flex items-center justify-between text-sm"><span>{formatBytes(item.size)}</span><Button onClick={() => onOpen(item.hash)} size="sm" type="button" variant="outline">Abrir</Button></div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

function WorkersTable({ items }: { items: BlossomWorkerRecord[] }) {
  return <Card><Table><TableHeader><TableRow><TableHead>Job</TableHead><TableHead>Status</TableHead><TableHead>Detalhe</TableHead><TableHead>Atualizado</TableHead></TableRow></TableHeader><TableBody>{items.map((item) => <TableRow key={item.job_id}><TableCell><div><p className="font-medium">{item.job_type}</p><p className="font-mono text-xs text-muted-foreground">{item.job_id}</p></div></TableCell><TableCell><Badge variant={item.status === "failed" ? "danger" : item.status === "running" ? "warning" : "default"}>{item.status}</Badge></TableCell><TableCell>{item.detail}</TableCell><TableCell>{formatDateTime(item.updated_at)}</TableCell></TableRow>)}</TableBody></Table></Card>
}

function MirrorHistoryCard({ history, onClear }: { history: Array<{ id: string; source_url: string; expected_sha256: string; status: string; created_at: string }>; onClear: () => void }) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-3">
        <CardTitle>{t("blossom.workers.historyTitle", "Histórico de mirroring")}</CardTitle>
        <Button disabled={history.length === 0} onClick={onClear} size="sm" type="button" variant="outline">{t("common.clear", "Limpar")}</Button>
      </CardHeader>
      <CardContent className="space-y-3">
        {history.length === 0 ? <p className="text-sm text-muted-foreground">{t("blossom.workers.historyEmpty", "Nenhum mirror recente salvo localmente.")}</p> : history.map((item) => (
          <div className="rounded-[var(--radius)] border border-border p-3" key={item.id}>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="font-mono text-xs text-primary">{item.id}</p>
              <Badge variant={item.status === "failed" ? "danger" : item.status === "running" ? "warning" : "default"}>{item.status}</Badge>
            </div>
            <p className="mt-2 break-all text-sm">{item.source_url}</p>
            <p className="mt-1 font-mono text-xs text-muted-foreground">{shortenId(item.expected_sha256, 14, 8)} · {formatDateTime(item.created_at)}</p>
          </div>
        ))}
      </CardContent>
    </Card>
  )
}

function SubscreenLinks({ reviewEnabled, reviewCount, reportCount }: { reviewEnabled: boolean; reviewCount: number; reportCount: number }) {
  const { t } = useTranslation()
  const links = [
    reviewEnabled ? { to: "/blossom/review", title: t("blossom.review.title", "Revisão"), description: t("blossom.review.subtitle", "Fila de arquivos aguardando decisão manual."), badge: reviewCount > 0 ? String(reviewCount) : undefined } : null,
    { to: "/blossom/reports", title: t("blossom.reports.title", "Reports"), description: t("blossom.reports.subtitle", "Relatos BUD-09 e resolução operacional."), badge: reportCount > 0 ? String(reportCount) : undefined },
    { to: "/blossom/audit", title: t("blossom.audit.title", "Auditoria"), description: t("blossom.audit.subtitle", "Linha do tempo imutável das ações críticas."), badge: undefined },
  ].filter(Boolean) as Array<{ to: string; title: string; description: string; badge?: string }>

  return (
    <div className="grid gap-4 lg:grid-cols-3">
      {links.map((item) => (
        <Link className="rounded-[var(--radius)] border border-border bg-card p-4 transition-colors hover:border-primary/40 hover:bg-primary/5" key={item.to} to={item.to}>
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="font-semibold text-foreground">{item.title}</p>
              <p className="mt-1 text-sm text-muted-foreground">{item.description}</p>
            </div>
            {item.badge ? <Badge variant="warning">{item.badge}</Badge> : null}
          </div>
        </Link>
      ))}
    </div>
  )
}
