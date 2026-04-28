import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { RefreshCw, Download, Filter, Database, Calendar, Search, Plus, LayoutGrid, BarChart3 } from "lucide-react"
import { useSearch, useNavigate } from "@tanstack/react-router"

import { PageHeader } from "@/components/shared/page-header"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { EventSearchItem } from "@/components/features/event-search/event-search-item"
import { EventSearchAggregates } from "@/components/features/event-search/event-search-aggregates"
import { EventSearchTimeline } from "@/components/features/event-search/event-search-timeline"
import { EventImportModal } from "@/components/features/event-search/event-import-modal"
import { NostrFilterBuilder } from "@/components/features/filters/nostr-filter-builder"
import { VirtualizedList } from "@/components/shared/virtualized-list"

import { useEventSearchAggregates, useEventSearchTimeline, useImportEventsMutation, useInfiniteEventSearch } from "@/hooks/use-admin-data"
import { parseSearchToFilters, filtersToSearch, eventSearchToNostrFilter, nostrFilterToEventSearch, type NostrFilter } from "@/lib/event-search"

import type { EventRecord } from "@/types/admin"

export function EventSearchPage() {
  const { t } = useTranslation()
  const search = useSearch({ from: "/events/search" })
  const navigate = useNavigate({ from: "/events/search" })
  
  const [selectedEventJson, setSelectedEventJson] = useState<string | null>(null)
  const [isImportModalOpen, setIsImportModalOpen] = useState(false)
  
  const filters = useMemo(() => parseSearchToFilters(search), [search])
  
  const query = useInfiniteEventSearch(filters)
  const aggregatesQuery = useEventSearchAggregates(filters)
  const timelineQuery = useEventSearchTimeline(filters, "day")
  const importMutation = useImportEventsMutation()

  const results = query.data?.pages.flatMap((page) => page.items) ?? []
  const total = query.data?.pages[0]?.total ?? 0

  const currentNostrFilter = useMemo(() => eventSearchToNostrFilter(filters), [filters])

  const handleFilterChange = (newFilter: NostrFilter) => {
    const updatedFilters = nostrFilterToEventSearch(newFilter)
    void navigate({ search: (old) => ({ ...old, ...filtersToSearch(updatedFilters) }) })
  }

  const handleRefresh = () => {
    void query.refetch()
    void aggregatesQuery.refetch()
    void timelineQuery.refetch()
  }

  return (
    <div className="space-y-6 animate-in fade-in slide-in-from-top-2 duration-500">
      <PageHeader
        actions={
          <div className="flex items-center gap-2">
            <Button onClick={() => setIsImportModalOpen(true)} size="sm" variant="outline" className="h-9 rounded-md border-primary/20 hover:bg-primary/5 hover:text-primary transition-all">
              <Plus className="mr-2 size-4" />
              {t("eventSearch.import", "Importar")}
            </Button>
            <Button onClick={handleRefresh} size="sm" variant="default" className="h-9 px-4 shadow-sm hover:shadow-primary/20 transition-all">
              <RefreshCw className={`mr-2 size-4 ${query.isFetching ? "animate-spin" : ""}`} />
              {t("common.refresh")}
            </Button>
          </div>
        }
        description={t("eventSearch.description", "Explore e analise eventos armazenados no relay utilizando filtros avançados.")}
        title={t("eventSearch.title", "Busca de Eventos")}
      />

      <div className="flex flex-col gap-6 lg:flex-row items-start">
        {/* Sidebar: Filters */}
        <aside className="w-full lg:w-[380px] shrink-0 space-y-4 lg:sticky lg:top-4">
          <NostrFilterBuilder 
            initialFilter={currentNostrFilter} 
            onChange={handleFilterChange}
            description={t("eventSearch.builderDescription", "Refine sua busca no banco de dados do relay.")}
          />
          
          <Card className="panel-shadow border-primary/5 bg-secondary/5 overflow-hidden">
            <div className="h-1 bg-gradient-to-r from-primary/20 via-primary/40 to-primary/20" />
            <CardContent className="p-4 space-y-4">
               <div className="flex items-center justify-between">
                 <p className="text-xs font-bold uppercase tracking-wider text-muted-foreground">{t("eventSearch.stats", "Metricas do Resultado")}</p>
                 <Badge variant="muted" className="font-mono text-[10px]">{results.length} / {total.toLocaleString()}</Badge>
               </div>
               
               <div className="grid grid-cols-2 gap-3">
                 <div className="bg-background/40 p-3 rounded-lg border border-border/50 flex flex-col gap-1">
                    <span className="text-[10px] text-muted-foreground font-medium">Total Global</span>
                    <span className="text-lg font-mono font-bold tracking-tight text-primary leading-none">{total.toLocaleString()}</span>
                 </div>
                 <div className="bg-background/40 p-3 rounded-lg border border-border/50 flex flex-col gap-1">
                    <span className="text-[10px] text-muted-foreground font-medium">Em Memória</span>
                    <span className="text-lg font-mono font-bold tracking-tight text-primary leading-none">{results.length}</span>
                 </div>
               </div>

               <Separator className="opacity-30" />
               
               <Button variant="outline" size="sm" className="w-full h-9 text-xs gap-2 border-primary/10 hover:bg-primary/5 transition-colors" disabled={results.length === 0}>
                  <Download className="size-3.5" />
                  {t("eventSearch.exportJSON", "Exportar Resultado")}
               </Button>
            </CardContent>
          </Card>
        </aside>

        {/* Main Content: Results */}
        <main className="flex-1 min-w-0 w-full">
          <Tabs defaultValue="events" className="w-full">
            <div className="bg-secondary/20 p-1.5 rounded-xl border border-border/50 mb-4 inline-flex">
              <TabsList className="bg-transparent h-8 p-0 gap-1">
                <TabsTrigger value="events" className="data-[state=active]:bg-background data-[state=active]:shadow-sm rounded-md px-4 text-xs gap-2 transition-all">
                  <LayoutGrid className="size-3.5" />
                  {t("eventSearch.events", "Eventos")}
                </TabsTrigger>
                <TabsTrigger value="aggregates" className="data-[state=active]:bg-background data-[state=active]:shadow-sm rounded-md px-4 text-xs gap-2 transition-all">
                  <BarChart3 className="size-3.5" />
                  {t("eventSearch.aggregates", "Agregados")}
                </TabsTrigger>
                <TabsTrigger value="timeline" className="data-[state=active]:bg-background data-[state=active]:shadow-sm rounded-md px-4 text-xs gap-2 transition-all">
                  <Calendar className="size-3.5" />
                  Timeline
                </TabsTrigger>
              </TabsList>
            </div>

            <TabsContent value="events" className="mt-0 focus-visible:ring-0 outline-none space-y-4">
               {results.length > 0 ? (
                 <VirtualizedList
                    items={results}
                    total={total}
                    estimateSize={140}
                    hasMore={query.hasNextPage}
                    isFetchingMore={query.isFetchingNextPage}
                    fetchMore={() => void query.fetchNextPage()}
                    className="bg-transparent border-0 rounded-none"
                    heightClassName="max-h-[calc(100vh-280px)] min-h-[500px]"
                    renderItem={(event, index) => (
                      <div className="py-1">
                        <EventSearchItem 
                          key={event.id} 
                          eventItem={event} 
                          index={index}
                          onOpenJSON={() => setSelectedEventJson(JSON.stringify(event, null, 2))}
                        />
                      </div>
                    )}
                 />
               ) : !query.isLoading ? (
                 <div className="flex flex-col items-center justify-center py-32 rounded-2xl border border-dashed border-border bg-muted/10 animate-in fade-in zoom-in-95 duration-500">
                   <div className="size-16 rounded-full bg-secondary/30 flex items-center justify-center mb-4">
                     <Search className="size-8 text-muted-foreground/30" />
                   </div>
                   <h3 className="text-lg font-semibold text-foreground/80">{t("eventSearch.noResultsFound", "Nenhum evento encontrado")}</h3>
                   <p className="text-sm text-muted-foreground max-w-xs text-center mt-1">
                     Tente ajustar seus filtros ou utilize o NIP-50 para uma busca por texto completo.
                   </p>
                 </div>
               ) : (
                 <div className="space-y-3">
                   {[1, 2, 3, 4].map((i) => (
                     <div key={i} className="h-32 w-full rounded-lg bg-secondary/20 animate-pulse border border-border/50" />
                   ))}
                 </div>
               )}
            </TabsContent>

            <TabsContent value="aggregates" className="mt-0 focus-visible:ring-0">
              <Card className="panel-shadow border-primary/5 bg-card/50 backdrop-blur-sm overflow-hidden">
                <div className="h-1 bg-gradient-to-r from-emerald-500/20 via-emerald-500/40 to-emerald-500/20" />
                <CardContent className="p-6">
                  <EventSearchAggregates 
                    data={aggregatesQuery.data} 
                    isLoading={aggregatesQuery.isLoading} 
                    isError={aggregatesQuery.isError}
                    onRetry={() => void aggregatesQuery.refetch()}
                  />
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="timeline" className="mt-0 focus-visible:ring-0">
              <Card className="panel-shadow border-primary/5 bg-card/50 backdrop-blur-sm overflow-hidden">
                <div className="h-1 bg-gradient-to-r from-blue-500/20 via-blue-500/40 to-blue-500/20" />
                <CardContent className="p-6">
                  <EventSearchTimeline 
                    data={timelineQuery.data} 
                    isLoading={timelineQuery.isLoading} 
                    isError={timelineQuery.isError}
                    onRetry={() => void timelineQuery.refetch()}
                  />
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </main>
      </div>

      <EventImportModal 
        open={isImportModalOpen}
        onOpenChange={setIsImportModalOpen}
        isPending={importMutation.isPending} 
        importResult={importMutation.data?.files ?? []}
        onImport={async (files) => { await importMutation.mutateAsync(files) }} 
      />

      <Dialog open={!!selectedEventJson} onOpenChange={(open) => !open && setSelectedEventJson(null)}>
        <DialogContent className="max-w-3xl max-h-[85vh] overflow-hidden flex flex-col p-0 border-primary/20 bg-background/95 backdrop-blur-xl">
          <DialogHeader className="p-6 pb-2">
            <DialogTitle className="flex items-center gap-2 font-mono text-sm uppercase tracking-widest text-primary">
              <Database className="size-4" />
              Raw Event Object
            </DialogTitle>
          </DialogHeader>
          <div className="flex-1 overflow-auto p-6 pt-2">
             <div className="rounded-lg bg-zinc-950/90 dark:bg-zinc-900/50 p-4 border border-white/5 font-mono text-[11px] leading-relaxed text-emerald-400/90 whitespace-pre shadow-2xl">
               {selectedEventJson}
             </div>
          </div>
          <div className="p-4 border-t border-white/5 bg-secondary/30 flex justify-end">
             <Button variant="secondary" size="sm" onClick={() => setSelectedEventJson(null)}>
                {t("common.close")}
             </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}