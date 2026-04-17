import { useEffect, useMemo, useState } from "react"
import { useSearch } from "@tanstack/react-router"
import { Upload } from "lucide-react"
import { useTranslation } from "react-i18next"

import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { VirtualizedList } from "@/components/shared/virtualized-list"
import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"

import { EventSearchForm } from "@/components/features/event-search/event-search-form"
import { EventSearchItem } from "@/components/features/event-search/event-search-item"
import { EventSearchAggregates } from "@/components/features/event-search/event-search-aggregates"
import { EventSearchTimeline } from "@/components/features/event-search/event-search-timeline"
import { EventImportModal } from "@/components/features/event-search/event-import-modal"

import { useEventSearchAggregates, useEventSearchTimeline, useImportEventsMutation, useInfiniteEventSearch } from "@/hooks/use-admin-data"
import { parseSearchToFilters, filtersToSearch, type EventSearchRouteSearch } from "@/lib/event-search"
import { useNavigateFrom } from "@/lib/router"

import type { EventRecord } from "@/types/admin"

export function EventSearchPage() {
  const { t } = useTranslation()
  const navigate = useNavigateFrom()
  const routeSearch = useSearch({ from: "/events/search" }) as EventSearchRouteSearch
  const filters = useMemo(() => parseSearchToFilters(routeSearch), [routeSearch])
  const [draft, setDraft] = useState(filters)
  const [jsonEvent, setJsonEvent] = useState<EventRecord | null>(null)
  const [importOpen, setImportOpen] = useState(false)
  const [importResult, setImportResult] = useState<Array<{ filename: string; total: number; inserted: number; duplicates: number; invalid: number; error?: string }>>([])
  
  const importMutation = useImportEventsMutation()

  const query = useInfiniteEventSearch(filters)
  const aggregates = useEventSearchAggregates(filters)
  const timeline = useEventSearchTimeline(filters, "hour")

  const results = query.data?.pages.flatMap((page) => page.items) ?? []
  const total = query.data?.pages[0]?.total ?? 0

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    void navigate({ search: filtersToSearch(draft) })
  }

  const handleClear = () => {
    setDraft({ query: "", authors: [], kinds: [], tags: [], limit: 100 })
    void navigate({ search: {} })
  }

  useEffect(() => {
    setDraft(filters)
  }, [filters])

  const summary = useMemo(() => t("eventSearch.summary", { shown: results.length, total }), [results.length, t, total])

  const handleImport = async (files: File[]) => {
    const response = await importMutation.mutateAsync(files)
    setImportResult(response.files)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        actions={
          <>
            <Button onClick={() => setImportOpen(true)} type="button" variant="outline">
              <Upload className="size-4" />
              {t("eventSearch.import")}
            </Button>
          </>
        }
        description={t("eventSearch.description")}
        title={t("eventSearch.title")}
      />

      <Card>
        <CardContent className="space-y-4 p-4">
          <EventSearchForm draft={draft} onDraftChange={setDraft} onSubmit={handleSubmit} onClear={handleClear} />
        </CardContent>
      </Card>

      <Card>
        <CardContent className="space-y-4 p-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="font-heading text-lg font-semibold">{t("eventSearch.results")}</p>
              <p className="text-sm text-muted-foreground">{summary}</p>
            </div>
          </div>

          <Tabs defaultValue="events">
            <TabsList>
              <TabsTrigger value="events">{t("eventSearch.events")}</TabsTrigger>
              <TabsTrigger value="aggregates">{t("eventSearch.aggregates")}</TabsTrigger>
              <TabsTrigger value="timeline">Timeline</TabsTrigger>
            </TabsList>
            <TabsContent className="mt-4 space-y-4" value="events">
              {query.isLoading && results.length === 0 && <LoadingPanel label={t("eventSearch.eventsLoading")} />}
              {query.isError && <ErrorPanel description={t("eventSearch.eventsErrorDescription")} onRetry={() => void query.refetch()} title={t("eventSearch.eventsErrorTitle")} />}
              {!query.isLoading && !query.isError && results.length === 0 && <EmptyPanel description={t("eventSearch.eventsEmptyDescription")} title={t("eventSearch.eventsEmptyTitle")} />}
              {!query.isLoading && !query.isError && results.length > 0 && (
                <VirtualizedList
                  estimateSize={170}
                  fetchMore={() => void query.fetchNextPage()}
                  hasMore={query.hasNextPage}
                  isFetchingMore={query.isFetchingNextPage}
                  items={results}
                  renderItem={(eventItem, index) => <EventSearchItem eventItem={eventItem} index={index} onOpenJSON={() => setJsonEvent(eventItem)} />}
                  total={total}
                />
              )}
            </TabsContent>
            <TabsContent className="mt-4 space-y-4" value="aggregates">
              <EventSearchAggregates 
                data={aggregates.data} 
                isLoading={aggregates.isLoading} 
                isError={aggregates.isError} 
                onRetry={() => void aggregates.refetch()} 
              />
            </TabsContent>
            <TabsContent className="mt-4 space-y-4" value="timeline">
              <EventSearchTimeline 
                data={timeline.data} 
                isLoading={timeline.isLoading} 
                isError={timeline.isError} 
                onRetry={() => void timeline.refetch()} 
              />
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      <Dialog onOpenChange={(open) => !open && setJsonEvent(null)} open={Boolean(jsonEvent)}>
        <DialogContent className="max-w-4xl">
          <DialogHeader>
            <DialogTitle>{t("eventSearch.jsonTitle")}</DialogTitle>
            <DialogDescription>{t("eventSearch.jsonDescription")}</DialogDescription>
          </DialogHeader>
          <pre className="max-h-[70vh] overflow-auto rounded-md border border-border bg-muted p-4 text-xs leading-relaxed text-foreground">
            {jsonEvent ? JSON.stringify(jsonEvent, null, 2) : ""}
          </pre>
        </DialogContent>
      </Dialog>

      <EventImportModal
        importResult={importResult}
        isPending={importMutation.isPending}
        onImport={handleImport}
        onOpenChange={setImportOpen}
        open={importOpen}
      />
    </div>
  )
}