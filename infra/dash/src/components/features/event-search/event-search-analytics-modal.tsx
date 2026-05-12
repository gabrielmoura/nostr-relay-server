import { BarChart3, Calendar, Clock3, Database, Hash, TrendingUp, Users } from "lucide-react"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import { EventSearchAnalyticsKpiStrip } from "@/components/features/event-search/event-search-analytics-kpi-strip"
import { EventSearchAggregates } from "@/components/features/event-search/event-search-aggregates"
import { EventSearchTimeline } from "@/components/features/event-search/event-search-timeline"
import { EventSearchTopAuthorsChart } from "@/components/features/event-search/event-search-top-authors-chart"
import { EventSearchTopTagsChart } from "@/components/features/event-search/event-search-top-tags-chart"
import { EventSearchTrendsPanel } from "@/components/features/event-search/event-search-trends-panel"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import type { EventAggregates, EventTimeline } from "@/types/admin"

type EventSearchAnalyticsTab = "overview" | "timeline" | "audience" | "trends"

interface EventSearchAnalyticsModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialTab?: EventSearchAnalyticsTab
  aggregates?: EventAggregates
  timeline?: EventTimeline
  isAggregatesLoading: boolean
  isAggregatesError: boolean
  isTimelineLoading: boolean
  isTimelineError: boolean
  onRetryAggregates: () => void
  onRetryTimeline: () => void
  onKindSelect?: (kind: number) => void
  onTagSelect?: (tag: string) => void
  onAuthorSelect?: (pubkey: string) => void
}

export function EventSearchAnalyticsModal({
  open,
  onOpenChange,
  initialTab = "overview",
  aggregates,
  timeline,
  isAggregatesLoading,
  isAggregatesError,
  isTimelineLoading,
  isTimelineError,
  onRetryAggregates,
  onRetryTimeline,
  onKindSelect,
  onTagSelect,
  onAuthorSelect,
}: EventSearchAnalyticsModalProps) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<EventSearchAnalyticsTab>(initialTab)

  useEffect(() => {
    if (open) {
      setActiveTab(initialTab)
    }
  }, [initialTab, open])

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-6xl max-h-[88vh] overflow-hidden border-primary/15 bg-background/95 backdrop-blur-xl p-0">
        <DialogHeader className="border-b border-border/60 px-6 py-4">
          <DialogTitle className="flex items-center gap-2 text-sm font-mono uppercase tracking-[0.18em] text-primary">
            <BarChart3 className="size-4" />
            {t("eventSearch.analyticsModalTitle", "Análises da busca")}
          </DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-auto px-6 py-5">
          <div className="space-y-6">
            <div className="rounded-xl border border-orange-200/70 bg-orange-50/80 px-4 py-3 text-sm text-orange-900 shadow-sm dark:border-orange-500/20 dark:bg-orange-500/10 dark:text-orange-100">
              <div className="flex items-start gap-3">
                <div className="mt-0.5 rounded-full bg-orange-500/15 p-1.5 text-orange-700 dark:text-orange-200">
                  <Clock3 className="size-4" />
                </div>
                <div className="space-y-1">
                  <p className="font-semibold">
                    {t("eventSearch.analyticsWarmupTitle", "Alguns dados podem demorar um pouco para aparecer")}
                  </p>
                  <p className="text-xs leading-relaxed text-orange-800/90 dark:text-orange-100/80">
                    {t(
                      "eventSearch.analyticsWarmupMessage",
                      "Os dados completos desta análise estão sendo tratados por um processo em segundo plano. Em cenários frios ou logo após mudanças no cache, parte das métricas pode levar alguns instantes para ser exibida.",
                    )}
                  </p>
                </div>
              </div>
            </div>

            <EventSearchAnalyticsKpiStrip aggregates={aggregates} />

            <Tabs onValueChange={(value) => setActiveTab(value as EventSearchAnalyticsTab)} value={activeTab} className="space-y-4">
              <TabsList className="grid w-full grid-cols-4 bg-secondary/20 p-1 rounded-md h-10">
                <TabsTrigger value="overview" className="gap-2 text-xs font-bold uppercase">
                  <Database className="size-3.5" />
                  {t("eventSearch.analyticsOverviewTab", "Visão geral")}
                </TabsTrigger>
                <TabsTrigger value="timeline" className="gap-2 text-xs font-bold uppercase">
                  <Calendar className="size-3.5" />
                  {t("eventSearch.analyticsTimeline", "Timeline")}
                </TabsTrigger>
                <TabsTrigger value="audience" className="gap-2 text-xs font-bold uppercase">
                  <Users className="size-3.5" />
                  {t("eventSearch.analyticsAudienceTab", "Autores e tags")}
                </TabsTrigger>
                <TabsTrigger value="trends" className="gap-2 text-xs font-bold uppercase">
                  <TrendingUp className="size-3.5" />
                  {t("eventSearch.analyticsTrendsTab", "Tendências")}
                </TabsTrigger>
              </TabsList>

              <TabsContent value="overview" className="space-y-6 mt-0">
                <section className="space-y-3">
                  <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                    <Database className="size-3.5" />
                    {t("eventSearch.analyticsAggregates", "Agregados")}
                  </div>
                  <EventSearchAggregates data={aggregates} isError={isAggregatesError} isLoading={isAggregatesLoading} onRetry={onRetryAggregates} onKindSelect={onKindSelect} onTagSelect={onTagSelect} />
                </section>
              </TabsContent>

              <TabsContent value="timeline" className="space-y-6 mt-0">
                <section className="space-y-3">
                  <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                    <Calendar className="size-3.5" />
                    {t("eventSearch.analyticsTimeline", "Timeline")}
                  </div>
                  <EventSearchTimeline data={timeline} isError={isTimelineError} isLoading={isTimelineLoading} onRetry={onRetryTimeline} />
                </section>
              </TabsContent>

              <TabsContent value="audience" className="space-y-6 mt-0">
                <div className="grid gap-4 xl:grid-cols-2">
                  <section className="space-y-3">
                    <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                      <Users className="size-3.5" />
                      {t("eventSearch.activeAuthors")}
                    </div>
                    <EventSearchTopAuthorsChart items={aggregates?.top_authors ?? []} onAuthorSelect={onAuthorSelect} />
                  </section>

                  <section className="space-y-3">
                    <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                      <Hash className="size-3.5" />
                      {t("eventSearch.commonTags")}
                    </div>
                    <EventSearchTopTagsChart items={aggregates?.top_tags ?? []} onTagSelect={onTagSelect} />
                  </section>
                </div>
              </TabsContent>

              <TabsContent value="trends" className="space-y-6 mt-0">
                <EventSearchTrendsPanel aggregates={aggregates} />
              </TabsContent>
            </Tabs>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
