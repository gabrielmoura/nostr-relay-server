import { useEffect, useMemo, useState } from "react"
import { Link, useNavigate, useSearch } from "@tanstack/react-router"
import { Copy, Eye, Search, Upload } from "lucide-react"
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
import { useEventSearchAggregates, useEventSearchTimeline, useImportEventsMutation, useInfiniteEventSearch } from "@/hooks/use-admin-data"
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

type EventSearchRouteSearch = {
  q?: string
  authors?: string
  kinds?: string
  tags?: string
  limit?: number
}

function parseCSV(value?: string) {
  if (!value) {
    return []
  }
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
}

function parseSearchToFilters(search: EventSearchRouteSearch): EventSearchFilters {
  const parsedKinds = parseCSV(search.kinds)
    .map((value) => Number(value))
    .filter((value) => !Number.isNaN(value))

  const parsedTags = parseCSV(search.tags).map((value) => {
    if (value.includes(":")) {
      return value
    }
    return `t:${value}`
  })

  return {
    ...defaultFilters,
    query: search.q ?? "",
    authors: parseCSV(search.authors),
    kinds: parsedKinds,
    tags: parsedTags,
    limit: typeof search.limit === "number" && search.limit > 0 ? search.limit : defaultFilters.limit,
  }
}

function filtersToSearch(filters: EventSearchFilters): EventSearchRouteSearch {
  return {
    q: filters.query || undefined,
    authors: filters.authors.length > 0 ? filters.authors.join(",") : undefined,
    kinds: filters.kinds.length > 0 ? filters.kinds.join(",") : undefined,
    tags: filters.tags.length > 0 ? filters.tags.join(",") : undefined,
    limit: filters.limit !== defaultFilters.limit ? filters.limit : undefined,
  }
}

type EventRef = { id: string; relay?: string }

function parseEventRefs(eventItem: EventRecord): EventRef[] {
  const refs: EventRef[] = []
  for (const tag of eventItem.tags) {
    if (tag[0] === "e" && tag[1]) {
      refs.push({ id: tag[1], relay: tag[2] })
    }
  }
  return refs
}

function parseServers(eventItem: EventRecord): string[] {
  const servers: string[] = []
  for (const tag of eventItem.tags) {
    if (tag[0] === "server" && tag[1]) {
      servers.push(tag[1])
    }
  }
  return servers
}

function parseProfileContent(content: string) {
  try {
    const parsed = JSON.parse(content) as { name?: string; display_name?: string; about?: string; picture?: string; nip05?: string }
    return parsed
  } catch {
    return null
  }
}

function tagValue(eventItem: EventRecord, key: string): string {
  const tag = eventItem.tags.find((entry) => entry[0] === key && entry[1])
  return tag?.[1] ?? ""
}

function eventHeadline(eventItem: EventRecord): string {
  if (eventItem.kind === 30003) {
    const title = tagValue(eventItem, "title")
    const dTag = tagValue(eventItem, "d")
    return title || dTag || "(lista sem titulo)"
  }
  return eventItem.content || "(sem conteudo textual)"
}

export function EventSearchPage() {
  const routeSearch = useSearch({ from: "/events/search" }) as EventSearchRouteSearch
  const navigate = useNavigate({ from: "/events/search" })
  const filters = useMemo(() => parseSearchToFilters(routeSearch), [routeSearch])
  const [draft, setDraft] = useState<EventSearchFilters>(filters)
  const [jsonEvent, setJsonEvent] = useState<EventRecord | null>(null)
  const [importOpen, setImportOpen] = useState(false)
  const [selectedFiles, setSelectedFiles] = useState<File[]>([])
  const [importResult, setImportResult] = useState<Array<{ filename: string; total: number; inserted: number; duplicates: number; invalid: number; error?: string }>>([])
  const [bucket, setBucket] = useState<"hour" | "day">("hour")
  const importMutation = useImportEventsMutation()

  const query = useInfiniteEventSearch(filters)
  const aggregates = useEventSearchAggregates(filters)
  const timeline = useEventSearchTimeline(filters, bucket)

  const results = query.data?.pages.flatMap((page) => page.items) ?? []
  const total = query.data?.pages[0]?.total ?? 0

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    void navigate({ search: filtersToSearch(draft) })
  }

  useEffect(() => {
    setDraft(filters)
  }, [filters])

  const summary = useMemo(() => `${results.length} de ${total} eventos carregados`, [results.length, total])

  return (
    <div className="space-y-6">
      <PageHeader
        actions={(
          <Button onClick={() => setImportOpen(true)} type="button" variant="outline">
            <Upload className="size-4" />
            Importar
          </Button>
        )}
        description="Busca por texto completo, autores, kinds e tags com exploracao por lista, agregados e timeline."
        title="Busca de eventos"
      />

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
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setDraft(defaultFilters)
                  void navigate({ search: {} })
                }}
              >
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
                    <EventSearchItem eventItem={eventItem} index={index} onOpenJSON={() => setJsonEvent(eventItem)} />
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

      <Dialog onOpenChange={setImportOpen} open={importOpen}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>Importar eventos JSONL</DialogTitle>
            <DialogDescription>Envie um ou mais arquivos JSONL para importacao temporaria e persistencia no relay.</DialogDescription>
          </DialogHeader>

          <div className="space-y-3">
            <Input
              multiple
              onChange={(event) => setSelectedFiles(Array.from(event.target.files ?? []))}
              type="file"
            />

            {selectedFiles.length > 0 ? (
              <div className="rounded-md border border-border p-3">
                <p className="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Arquivos selecionados</p>
                <div className="space-y-1 text-sm">
                  {selectedFiles.map((file) => <p key={file.name}>{file.name}</p>)}
                </div>
              </div>
            ) : null}

            <div className="flex justify-end">
              <Button
                disabled={importMutation.isPending || selectedFiles.length === 0}
                onClick={async () => {
                  try {
                    const response = await importMutation.mutateAsync(selectedFiles)
                    setImportResult(response.files)
                    toast.success("Importacao concluida.")
                    void query.refetch()
                  } catch (error) {
                    if (error instanceof Error) {
                      toast.error(error.message)
                    }
                  }
                }}
                type="button"
              >
                {importMutation.isPending ? "Importando..." : "Importar arquivos"}
              </Button>
            </div>

            {importResult.length > 0 ? (
              <div className="rounded-md border border-border p-3">
                <p className="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Resultado por arquivo</p>
                <div className="space-y-2 text-sm">
                  {importResult.map((file) => (
                    <div className="rounded border border-border px-3 py-2" key={file.filename}>
                      <p className="font-medium text-foreground">{file.filename}</p>
                      <p className="text-xs text-muted-foreground">total {file.total} · inseridos {file.inserted} · duplicados {file.duplicates} · invalidos {file.invalid}</p>
                      {file.error ? <p className="mt-1 text-xs text-destructive">erro: {file.error}</p> : null}
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function EventSearchItem({ eventItem, index, onOpenJSON }: { eventItem: EventRecord; index: number; onOpenJSON: () => void }) {
  const refs = parseEventRefs(eventItem)
  const servers = parseServers(eventItem)
  const profile = eventItem.kind === 0 ? parseProfileContent(eventItem.content) : null

  return (
    <div className="rounded-[calc(var(--radius)-0.15rem)] border border-border bg-card p-4">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="flex min-w-0 flex-1 gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-md bg-destructive text-sm font-heading font-semibold text-white">{index + 1}</div>
          <div className="min-w-0 space-y-3">
            <div>
              <p className="text-sm font-semibold text-foreground">[kind:{eventItem.kind}] {eventHeadline(eventItem)}</p>
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

            {eventItem.kind === 30003 ? <Badge variant="muted">Evento de lista (kind 30003)</Badge> : null}

            {eventItem.kind === 1010 && refs.length > 0 ? (
              <div className="space-y-1 rounded-md border border-warning/40 bg-warning/5 p-3 text-xs">
                <p className="font-semibold text-foreground">Content Change Event</p>
                <p className="text-muted-foreground">Este evento altera o conteudo de:</p>
                {refs.map((ref) => (
                  <Link className="block font-mono underline decoration-dotted underline-offset-2" key={`cc-${ref.id}`} params={{ eventId: ref.id }} to="/events/$eventId">{ref.id}</Link>
                ))}
              </div>
            ) : null}

            {eventItem.kind === 10063 && servers.length > 0 ? (
              <div className="space-y-2 rounded-md border border-border bg-muted/20 p-3">
                <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Servidores Blossom</p>
                <div className="space-y-1">
                  {servers.map((server) => (
                    <a className="block break-all text-xs text-primary underline decoration-dotted underline-offset-2" href={server} key={server} rel="noreferrer" target="_blank">{server}</a>
                  ))}
                </div>
              </div>
            ) : null}

            {eventItem.kind === 0 && profile ? (
              <div className="rounded-md border border-border bg-muted/20 p-3 text-xs">
                <p className="font-semibold text-foreground">Evento de perfil</p>
                <p className="text-muted-foreground">Nome: {profile.display_name || profile.name || "-"}</p>
                {profile.nip05 ? <p className="text-muted-foreground">NIP-05: {profile.nip05}</p> : null}
                {profile.about ? <p className="mt-1 line-clamp-2 text-muted-foreground">{profile.about}</p> : null}
              </div>
            ) : null}

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
          <Button onClick={onOpenJSON} size="sm" variant="outline">
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
  )
}
