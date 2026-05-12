import { useMemo, useState } from "react"
import { BarChart3, CircleHelp, Plus } from "lucide-react"
import { useTranslation } from "react-i18next"

import { LabelsAnalyticsModal } from "@/components/features/labels/labels-analytics-modal"
import { CreateLabelDialog } from "@/components/features/labels/create-label-dialog"
import { LabelsHelpDialog } from "@/components/features/labels/labels-help-dialog"
import { LabelsFilterBar } from "@/components/features/labels/labels-filter-bar"
import { LabelsStatsStrip } from "@/components/features/labels/labels-stats-strip"
import { LabelsTargetsTable } from "@/components/features/labels/labels-targets-table"
import { LabelsTimeline } from "@/components/features/labels/labels-timeline"
import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useInfiniteLabels, useLabelsSummary } from "@/hooks/use-admin-data"
import { groupLabelsByTarget, type LabelsRouteView } from "@/lib/labels"
import { ApiError } from "@/services/admin"
import type { AdminLabelsFilters } from "@/types/admin"

type LabelsWorkspaceProps = {
  filters: AdminLabelsFilters
  view: LabelsRouteView
  onFiltersChange: (patch: Partial<AdminLabelsFilters>) => void
  onResetFilters: () => void
  onViewChange: (view: LabelsRouteView) => void
}

export function LabelsWorkspace({ filters, view, onFiltersChange, onResetFilters, onViewChange }: LabelsWorkspaceProps) {
  const { t } = useTranslation()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [helpOpen, setHelpOpen] = useState(false)
  const [analyticsOpen, setAnalyticsOpen] = useState(false)

  const labelsQuery = useInfiniteLabels(filters)
  const summaryQuery = useLabelsSummary(filters)

  const items = useMemo(() => labelsQuery.data?.pages.flatMap((page) => page.items) ?? [], [labelsQuery.data])
  const groupedTargets = useMemo(() => groupLabelsByTarget(items), [items])

  const loading = (labelsQuery.isLoading && items.length === 0) || summaryQuery.isLoading
  const error = labelsQuery.error ?? summaryQuery.error
  const errorDescription = error instanceof ApiError && error.requestId
    ? `${error.message} (request-id: ${error.requestId})`
    : error instanceof Error
      ? error.message
      : t("labels.errorDescription")

  return (
    <div className="space-y-6">
      <PageHeader
        actions={
          <>
            <Button onClick={() => setHelpOpen(true)} type="button" variant="outline">
              <CircleHelp className="size-4" />
              {t("labels.help.trigger", "Ajuda")}
            </Button>
            <Button onClick={() => setAnalyticsOpen(true)} type="button" variant="outline">
              <BarChart3 className="size-4" />
              {t("labels.analytics.trigger", "Análises")}
            </Button>
            <Button onClick={() => setDialogOpen(true)}>
              <Plus className="size-4" />
              {t("labels.create.trigger", "Criar label")}
            </Button>
          </>
        }
        className="rounded-[var(--radius)] border border-primary/15 bg-[linear-gradient(135deg,rgba(14,165,233,0.08),rgba(34,197,94,0.08))] p-5 panel-shadow"
        description={t("labels.description", "Gerencie eventos kind 1985, agrupe por alvo e publique novos labels NIP-32 a partir do painel interno.")}
        title={t("labels.title", "Labels NIP-32")}
      />

      {summaryQuery.data ? <LabelsStatsStrip summary={summaryQuery.data} /> : null}

      <LabelsFilterBar filters={filters} onChange={onFiltersChange} onReset={onResetFilters} summary={summaryQuery.data} />

      {loading ? <LoadingPanel label={t("labels.loading", "Carregando labels administrativos...")} /> : null}
      {error ? (
        <ErrorPanel
          description={errorDescription}
          onRetry={() => {
            void labelsQuery.refetch()
            void summaryQuery.refetch()
          }}
          title={t("labels.errorTitle", "Falha ao carregar labels")}
        />
      ) : null}

      {!loading && !error && items.length === 0 ? (
        <EmptyPanel
          action={<Button onClick={() => setDialogOpen(true)}>{t("labels.create.trigger", "Criar label")}</Button>}
          description={t("labels.emptyDescription", "Nenhum label NIP-32 corresponde ao filtro atual.")}
          title={t("labels.emptyTitle", "Nenhum label encontrado")}
        />
      ) : null}

      {!loading && !error && items.length > 0 ? (
        <Tabs onValueChange={(next) => onViewChange(next as LabelsRouteView)} value={view}>
          <TabsList>
            <TabsTrigger value="timeline">{t("labels.views.timeline", "Timeline")}</TabsTrigger>
            <TabsTrigger value="targets">{t("labels.views.targets", "By Target")}</TabsTrigger>
          </TabsList>

          <TabsContent className="mt-4" value="timeline">
            <LabelsTimeline
              hasMore={labelsQuery.hasNextPage}
              isFetchingMore={labelsQuery.isFetchingNextPage}
              items={items}
              onLoadMore={() => void labelsQuery.fetchNextPage()}
            />
          </TabsContent>

          <TabsContent className="mt-4" value="targets">
            <LabelsTargetsTable
              hasMore={labelsQuery.hasNextPage}
              isFetchingMore={labelsQuery.isFetchingNextPage}
              items={groupedTargets}
              onLoadMore={() => void labelsQuery.fetchNextPage()}
            />
          </TabsContent>
        </Tabs>
      ) : null}

      <CreateLabelDialog onOpenChange={setDialogOpen} open={dialogOpen} />
      <LabelsHelpDialog onOpenChange={setHelpOpen} open={helpOpen} />
      <LabelsAnalyticsModal onOpenChange={setAnalyticsOpen} open={analyticsOpen} summary={summaryQuery.data} />
    </div>
  )
}
