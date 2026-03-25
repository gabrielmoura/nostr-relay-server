import { useMemo, useState } from "react"
import { Link, useSearch } from "@tanstack/react-router"
import { Copy, Eye, Search } from "lucide-react"
import { toast } from "sonner"

import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { VirtualizedList } from "@/components/shared/virtualized-list"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useEventSearchAggregates, useEventSearchTimeline, useInfiniteEventSearch } from "@/hooks/use-admin-data"
import { getEventTags } from "@/services/admin"
import { formatDateTime, shortenId } from "@/lib/utils"
import type { EventRecord, EventSearchFilters } from "@/types/admin"

const defaultFilters: EventSearchFilters = {
  query: "",
  authors: [],
  kinds: [],
  tags: [],
  limit: 100,
}

export function EventSearchPage() {
  const routeSearch = useSearch({ from: "/events/search" }) as { q?: string; tags?: string }
  const [draft, setDraft] = useState<EventSearchFilters>({
    ...defaultFilters,
    query: routeSearch.q ?? "",
    tags: routeSearch.tags ? routeSearch.tags.split(",").map((tag) => `t:${tag.trim()}`).filter(Boolean) : [],
  })
  const [filters, setFilters] = useState<EventSearchFilters>({
    ...defaultFilters,
    query: routeSearch.q ?? "",
    tags: routeSearch.tags ? routeSearch.tags.split(",").map((tag) => `t:${tag.trim()}`).filter(Boolean) : [],
  })
  const [jsonEvent, setJsonEvent] = useState<EventRecord | null>(null)
  const [bucket, setBucket] = useState<"hour" | "day">("hour")

  const query = useInfiniteEventSearch(filters)
  const aggregates = useEventSearchAggregates(filters)
  const timeline = useEventSearchTimeline(filters, bucket)

  const results = query.data?.pages.flatMap((page) => page.items) ?? []
  const total = query.data?.pages[0]?.total ?? 0

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setFilters(draft)
  }

  const summary = useMemo(() => `${results.length} de ${total} eventos carregados`, [results.length, total])

  return (
    <div className="space-y-6">
      <PageHeader description="Busca por texto completo, autores, kinds e tags com exploracao por lista, agregados e timeline." title="Busca de eventos" />

      <Card>
        <CardContent className="space-y-4 p-4">
          <form className="space-y-3" onSubmit={handleSubmit}>
            <div className="grid gap-3 lg:grid-cols-[2fr_1fr_1fr_auto]">
              <Input placeholder='kind:1 content:"relay" author:npub1...' value={draft.query} onChange={(event) => setDraft((current) => ({ ...current, query: event.target.value }))} />
              <Input placeholder="Kinds (1,7,30023)" value={draft.kinds.join(",")} onChange={(event) => setDraft((current) => ({ ...current, kinds: event.target.value.split(",").map((value) => Number(value.trim())).filter((value) => !Number.isNaN(value)) }))} />
              <Input placeholder="Tags (#t:relay,#t:ops)" value={draft.tags.join(",")} onChange={(event) => setDraft((current) => ({ ...current, tags: event.target.value.split(",").map((value) => value.trim()).filter(Boolean) }))} />
              <Button type="submit">
                <Search className="size-4" />
                Buscar
              </Button>
            </div>
            <div className="grid gap-3 lg:grid-cols-[2fr_1fr_auto]">
              <Input placeholder="Autores (pubkeys separados por virgula)" value={draft.authors.join(",")} onChange={(event) => setDraft((current) => ({ ...current, authors: event.target.value.split(",").map((value) => value.trim()).filter(Boolean) }))} />
              <Input placeholder="Limite" type="number" value={draft.limit} onChange={(event) => setDraft((current) => ({ ...current, limit: Number(event.target.value) || 100 }))} />
              <Button type="button" variant="outline" onClick={() => setDraft(defaultFilters)}>
                Limpar
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="space-y-4 p-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="font-heading text-lg font-semibold">Resultados</p>
              <p className="text-sm text-muted-foreground">{summary}</p>
            </div>
          </div>

          <Tabs defaultValue="events">
            <TabsList>
              <TabsTrigger value="events">Eventos</TabsTrigger>
              <TabsTrigger value="aggregates">Agregados</TabsTrigger>
              <TabsTrigger value="timeline">Timeline</TabsTrigger>
            </TabsList>
            <TabsContent className="mt-4 space-y-4" value="events">
              {query.isLoading && results.length === 0 ? <LoadingPanel label="Consultando `/admin/events/search`..." /> : null}
              {query.isError ? <ErrorPanel description="A busca de eventos falhou. Verifique os filtros e a conexao com o backend." onRetry={() => void query.refetch()} title="Falha de consulta" /> : null}
              {!query.isLoading && !query.isError && results.length === 0 ? <EmptyPanel description="Ajuste período, query ou remova filtros muito restritivos." title="Nenhum evento para este filtro" /> : null}
              {!query.isLoading && !query.isError && results.length > 0 ? (
                <VirtualizedList
                  estimateSize={170}
                  fetchMore={() => void query.fetchNextPage()}
                  hasMore={query.hasNextPage}
                  isFetchingMore={query.isFetchingNextPage}
                  items={results}
                  renderItem={(eventItem, index) => (
                    <div className="rounded-[calc(var(--radius)-0.15rem)] border border-border bg-card p-4">
                      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                        <div className="flex min-w-0 flex-1 gap-3">
                          <div className="flex size-10 shrink-0 items-center justify-center rounded-md bg-destructive text-sm font-heading font-semibold text-white">{index + 1}</div>
                          <div className="min-w-0 space-y-3">
                            <div>
                              <p className="text-sm font-semibold text-foreground">[kind:{eventItem.kind}] {eventItem.content || "(sem conteudo textual)"}</p>
                              <p className="mt-1 text-xs text-muted-foreground">
                                <Link className="font-medium text-foreground underline decoration-dotted underline-offset-2 hover:text-primary" params={{ pubkey: eventItem.pubkey }} to="/users/$pubkey">
                                  Autor: {shortenId(eventItem.pubkey, 12, 4)}
                                </Link>
                                {" · "}
                                {formatDateTime(eventItem.created_at)}
                                {" · event_id: "}
                                {shortenId(eventItem.id, 10, 4)}
                              </p>
                            </div>
                            <div className="flex flex-wrap gap-2">
                              {getEventTags(eventItem).map((tag) => (
                                <Link key={`${eventItem.id}-${tag}`} search={{ tags: tag }} to="/events/search">
                                  <Badge variant="muted">#{tag}</Badge>
                                </Link>
                              ))}
                            </div>
                          </div>
                        </div>
                        <div className="flex shrink-0 flex-wrap gap-2">
                          <Button onClick={() => setJsonEvent(eventItem)} size="sm" variant="outline">
                            <Eye className="size-4" />
                            Ver JSON
                          </Button>
                          <Button asChild size="sm" variant="outline">
                            <Link params={{ eventId: eventItem.id }} to="/events/$eventId">
                              Ver Evento
                            </Link>
                          </Button>
                          <Button
                            onClick={async () => {
                              await navigator.clipboard.writeText(eventItem.id)
                              toast.success("Event ID copiado.")
                            }}
                            size="sm"
                            variant="outline"
                          >
                            <Copy className="size-4" />
                            Copiar ID
                          </Button>
                        </div>
                      </div>
                    </div>
                  )}
                  total={total}
                />
              ) : null}
            </TabsContent>
            <TabsContent className="mt-4 space-y-4" value="aggregates">
              {aggregates.isLoading ? <LoadingPanel label="Calculando agregados da busca..." /> : null}
              {aggregates.isError ? <ErrorPanel description="Nao foi possivel calcular agregados para os filtros atuais." onRetry={() => void aggregates.refetch()} title="Falha de agregacao" /> : null}
              {aggregates.data ? (
                <div className="grid gap-4 lg:grid-cols-3">
                  <Card>
                    <CardContent className="space-y-2 p-4">
                      <p className="font-heading text-sm font-semibold">Kinds mais frequentes</p>
                      {aggregates.data.kinds.map((item) => (
                        <div className="flex items-center justify-between text-sm" key={`kind-${item.kind}`}>
                          <span>kind {item.kind}</span>
                          <Badge variant="muted">{item.count}</Badge>
                        </div>
                      ))}
                    </CardContent>
                  </Card>
                  <Card>
                    <CardContent className="space-y-2 p-4">
                      <p className="font-heading text-sm font-semibold">Autores mais ativos</p>
                      {aggregates.data.top_authors.map((item) => (
                        <div className="flex items-center justify-between text-sm" key={`author-${item.pubkey}`}>
                          <Link className="underline decoration-dotted underline-offset-2" params={{ pubkey: item.pubkey }} to="/users/$pubkey">
                            {shortenId(item.pubkey, 10, 4)}
                          </Link>
                          <Badge variant="muted">{item.count}</Badge>
                        </div>
                      ))}
                    </CardContent>
                  </Card>
                  <Card>
                    <CardContent className="space-y-2 p-4">
                      <p className="font-heading text-sm font-semibold">Tags mais comuns</p>
                      {aggregates.data.top_tags.map((item) => (
                        <div className="flex items-center justify-between text-sm" key={`tag-${item.tag}`}>
                          <Link search={{ tags: item.tag }} to="/events/search">
                            #{item.tag}
                          </Link>
                          <Badge variant="muted">{item.count}</Badge>
                        </div>
                      ))}
                    </CardContent>
                  </Card>
                </div>
              ) : null}
            </TabsContent>
            <TabsContent className="mt-4 space-y-4" value="timeline">
              <div className="flex gap-2">
                <Button onClick={() => setBucket("hour")} size="sm" variant={bucket === "hour" ? "default" : "outline"}>Hora</Button>
                <Button onClick={() => setBucket("day")} size="sm" variant={bucket === "day" ? "default" : "outline"}>Dia</Button>
              </div>
              {timeline.isLoading ? <LoadingPanel label="Montando timeline da busca..." /> : null}
              {timeline.isError ? <ErrorPanel description="Nao foi possivel montar a timeline." onRetry={() => void timeline.refetch()} title="Falha de timeline" /> : null}
              {timeline.data ? (
                <Card>
                  <CardContent className="space-y-3 p-4">
                    {timeline.data.points.length === 0 ? <EmptyPanel description="Sem pontos para os filtros atuais." title="Timeline vazia" /> : null}
                    {timeline.data.points.map((point) => (
                      <div className="space-y-1" key={point.ts}>
                        <div className="flex items-center justify-between text-xs text-muted-foreground">
                          <span>{formatDateTime(point.ts)}</span>
                          <span>{point.count} eventos</span>
                        </div>
                        <div className="h-2 rounded bg-muted">
                          <div className="h-full rounded bg-primary" style={{ width: `${Math.max(3, (point.count / Math.max(1, timeline.data.points[0]?.count ?? 1)) * 100)}%` }} />
                        </div>
                      </div>
                    ))}
                  </CardContent>
                </Card>
              ) : null}
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      <Dialog onOpenChange={(open) => !open && setJsonEvent(null)} open={Boolean(jsonEvent)}>
        <DialogContent className="max-w-4xl">
          <DialogHeader>
            <DialogTitle>Evento JSON</DialogTitle>
            <DialogDescription>Inspecao tecnica do envelope do evento em formato formatado.</DialogDescription>
          </DialogHeader>
          <pre className="max-h-[70vh] overflow-auto rounded-md border border-border bg-muted p-4 text-xs leading-relaxed text-foreground">
            {jsonEvent ? JSON.stringify(jsonEvent, null, 2) : ""}
          </pre>
        </DialogContent>
      </Dialog>
    </div>
  )
}
