import React, { Suspense, useState } from "react"
import { QueryErrorResetBoundary } from "@tanstack/react-query"
import { Link, useParams } from "@tanstack/react-router"
import { Copy, ExternalLink, Plus, RefreshCcw, StepForward, X } from "lucide-react"
import { toast } from "sonner"

import { BanUserDialog } from "@/components/features/ban-user-dialog"
import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { useEventDetail, useEventDetailSuspense, useFetchEventFromRelaysMutation } from "@/hooks/use-admin-data"
import { ApiError } from "@/services/admin"
import { formatDateTime, shortenId } from "@/lib/utils"

type TagTuple = string[]

type EventDetailBoundaryProps = {
  children: React.ReactNode
  fallbackRender: (error: Error, reset: () => void) => React.ReactNode
  onReset?: () => void
}

type EventDetailBoundaryState = {
  error: Error | null
}

const commonRelays = [
  "wss://relay.damus.io",
  "wss://relay.primal.net",
  "wss://nos.lol",
  "wss://relay.nostr.band",
  "wss://nostr.mom",
]

const kindLabels: Record<number, string> = {
  1: "nota (NIP-10)",
  6: "repost de nota (NIP-18)",
  7: "reacao (NIP-25)",
  16: "repost generico (NIP-18)",
  20: "picture event (NIP-68)",
  21: "video event (NIP-71)",
  22: "short video event (NIP-71)",
  818: "merge request wiki (NIP-54)",
  1111: "comentario (NIP-22)",
  30023: "artigo long-form (NIP-23)",
  30024: "rascunho long-form (NIP-23)",
  30818: "wiki article (NIP-54)",
  30819: "wiki redirect (NIP-54)",
  31989: "recomendacao de handler (NIP-89)",
  31990: "handler metadata (NIP-89)",
  34235: "video enderecavel (NIP-71)",
  34236: "short video enderecavel (NIP-71)",
  34550: "comunidade (NIP-72)",
  4550: "aprovacao de post (NIP-72)",
}

class EventDetailErrorBoundary extends React.Component<EventDetailBoundaryProps, EventDetailBoundaryState> {
  state: EventDetailBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error) {
    return { error }
  }

  reset = () => {
    this.setState({ error: null })
    this.props.onReset?.()
  }

  render() {
    if (this.state.error) {
      return this.props.fallbackRender(this.state.error, this.reset)
    }
    return this.props.children
  }
}

function firstTagValue(tags: TagTuple[], key: string) {
  for (const tag of tags) {
    if (tag[0] === key && tag[1]) {
      return tag[1]
    }
  }
  return ""
}

function tagValues(tags: TagTuple[], key: string) {
  const values: string[] = []
  for (const tag of tags) {
    if (tag[0] === key && tag[1]) {
      values.push(tag[1])
    }
  }
  return values
}

function unique(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)))
}

function parseImetaResources(tags: TagTuple[]) {
  const imageURLs: string[] = []
  const mediaURLs: string[] = []
  const mimeTypes: string[] = []
  const altTexts: string[] = []

  for (const tag of tags) {
    if (tag[0] !== "imeta") {
      continue
    }
    for (const item of tag.slice(1)) {
      if (item.startsWith("image ")) {
        imageURLs.push(item.slice("image ".length).trim())
      }
      if (item.startsWith("url ")) {
        mediaURLs.push(item.slice("url ".length).trim())
      }
      if (item.startsWith("m ")) {
        mimeTypes.push(item.slice("m ".length).trim())
      }
      if (item.startsWith("alt ")) {
        altTexts.push(item.slice("alt ".length).trim())
      }
    }
  }

  return {
    imageURLs: unique(imageURLs),
    mediaURLs: unique(mediaURLs),
    mimeTypes: unique(mimeTypes),
    altTexts: unique(altTexts),
  }
}

function parseMediaURLsFromTags(tags: TagTuple[]) {
  const urls = unique([...tagValues(tags, "url"), ...tagValues(tags, "r")])
  return urls.filter((url) => /^https?:\/\//.test(url))
}

function pickVideoURL(urls: string[]) {
  const preferred = urls.find((url) => /\.(mp4|webm|ogg|mov)(\?|$)/i.test(url) || /video/i.test(url))
  return preferred ?? urls[0] ?? ""
}

function parseEmbeddedRepost(content: string, kind: number) {
  if (kind !== 6 && kind !== 16) {
    return null
  }

  const trimmed = content.trim()
  if (!trimmed.startsWith("{")) {
    return null
  }

  try {
    const parsed = JSON.parse(trimmed) as { id?: unknown; kind?: unknown; pubkey?: unknown; content?: unknown }
    if (typeof parsed.id !== "string") {
      return null
    }

    return {
      id: parsed.id,
      kind: typeof parsed.kind === "number" ? parsed.kind : -1,
      pubkey: typeof parsed.pubkey === "string" ? parsed.pubkey : "",
      content: typeof parsed.content === "string" ? parsed.content : "",
    }
  } catch {
    return null
  }
}

function isNotFoundError(error: Error) {
  return error instanceof ApiError && error.status === 404
}

export function EventDetailPage() {
  const { eventId } = useParams({ from: "/events/$eventId" })

  return (
    <QueryErrorResetBoundary>
      {({ reset }) => (
        <EventDetailErrorBoundary
          fallbackRender={(error, resetBoundary) => (
            <EventDetailErrorState error={error} eventID={eventId} onRetry={() => {
              reset()
              resetBoundary()
            }} />
          )}
          onReset={reset}
        >
          <Suspense fallback={<LoadingPanel label="Carregando detalhes do evento..." />}>
            <EventDetailPageContent eventID={eventId} />
          </Suspense>
        </EventDetailErrorBoundary>
      )}
    </QueryErrorResetBoundary>
  )
}

function EventDetailErrorState({ eventID, error, onRetry }: { eventID: string; error: Error; onRetry: () => void }) {
  const mutation = useFetchEventFromRelaysMutation()
  const [open, setOpen] = useState(false)
  const [relayInput, setRelayInput] = useState("")
  const [selectedRelays, setSelectedRelays] = useState<string[]>(commonRelays)
  const [relayFeedback, setRelayFeedback] = useState<Array<{ relay: string; status: string; error?: string }>>([])

  const addRelay = () => {
    const value = relayInput.trim()
    if (!value) {
      return
    }
    if (!/^wss?:\/\//.test(value)) {
      toast.error("Use URL com ws:// ou wss://")
      return
    }
    setSelectedRelays((current) => (current.includes(value) ? current : [...current, value]))
    setRelayInput("")
  }

  const toggleRelay = (relay: string) => {
    setSelectedRelays((current) => (current.includes(relay) ? current.filter((item) => item !== relay) : [...current, relay]))
  }

  if (!isNotFoundError(error)) {
    return <ErrorPanel description={error.message || "Falha ao carregar evento."} onRetry={onRetry} title="Falha ao carregar evento" />
  }

  return (
    <>
      <EmptyPanel
        action={(
          <Button onClick={() => setOpen(true)} type="button">
            <RefreshCcw className="size-4" />
            Buscar em outros Relays
          </Button>
        )}
        description="Este evento ainda nao esta indexado no relay atual. Voce pode consultar relays externos e importar o evento."
        title="Evento nao existe no relay"
      />

      <Dialog onOpenChange={setOpen} open={open}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>Buscar evento em outros relays</DialogTitle>
            <DialogDescription>Selecione relays comuns, adicione URLs extras e importe o evento para o banco local.</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Relays comuns</p>
              <div className="flex flex-wrap gap-2">
                {commonRelays.map((relay) => (
                  <Button key={relay} onClick={() => toggleRelay(relay)} size="sm" type="button" variant={selectedRelays.includes(relay) ? "default" : "outline"}>
                    {relay}
                  </Button>
                ))}
              </div>
            </div>

            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Adicionar relay</p>
              <div className="flex gap-2">
                <Input
                  onChange={(event) => setRelayInput(event.target.value)}
                  placeholder="wss://relay.exemplo.com"
                  value={relayInput}
                />
                <Button onClick={addRelay} type="button" variant="outline">
                  <Plus className="size-4" />
                  Incluir
                </Button>
              </div>
            </div>

            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Lista de busca</p>
              <div className="flex max-h-40 flex-wrap gap-2 overflow-auto rounded-md border border-border p-2">
                {selectedRelays.map((relay) => (
                  <Badge className="flex items-center gap-1" key={relay} variant="muted">
                    <span className="max-w-[220px] truncate">{relay}</span>
                    <button className="cursor-pointer" onClick={() => setSelectedRelays((current) => current.filter((item) => item !== relay))} type="button">
                      <X className="size-3" />
                    </button>
                  </Badge>
                ))}
                {selectedRelays.length === 0 ? <p className="text-xs text-muted-foreground">Nenhum relay selecionado.</p> : null}
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button onClick={() => setOpen(false)} type="button" variant="outline">Cancelar</Button>
            <Button
              disabled={mutation.isPending || selectedRelays.length === 0}
              onClick={async () => {
                try {
                  const response = await mutation.mutateAsync({ eventID, relays: selectedRelays })
                  setRelayFeedback(response.relay_results ?? [])
                  toast.success(`Evento encontrado em ${response.source_relay}.`)
                  setOpen(false)
                  onRetry()
                } catch (mutationError) {
                  if (mutationError instanceof ApiError && mutationError.details && typeof mutationError.details === "object") {
                    const details = mutationError.details as { relay_results?: Array<{ relay: string; status: string; error?: string }> }
                    setRelayFeedback(details.relay_results ?? [])
                  }
                  if (mutationError instanceof Error) {
                    toast.error(mutationError.message)
                  } else {
                    toast.error("Falha ao buscar evento em outros relays.")
                  }
                }
              }}
              type="button"
            >
              {mutation.isPending ? "Buscando..." : "Buscar evento"}
            </Button>
          </DialogFooter>

          {relayFeedback.length > 0 ? (
            <div className="space-y-2 rounded-md border border-border bg-muted/30 p-3">
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Resultado por relay</p>
              <div className="max-h-40 space-y-2 overflow-auto">
                {relayFeedback.map((entry) => (
                  <div className="flex items-center justify-between gap-2 text-xs" key={`${entry.relay}-${entry.status}`}>
                    <span className="truncate text-muted-foreground">{entry.relay}</span>
                    <Badge variant={entry.status === "found" ? "success" : "muted"}>{entry.status}</Badge>
                  </div>
                ))}
              </div>
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </>
  )
}

function EventDetailPageContent({ eventID }: { eventID: string }) {
  const query = useEventDetailSuspense(eventID)
  const detail = query.data
  const event = detail.event
  const tags = event.tags ?? []

  const title = firstTagValue(tags, "title")
  const summary = firstTagValue(tags, "summary")
  const contentWarning = firstTagValue(tags, "content-warning")
  const alt = firstTagValue(tags, "alt")
  const eventD = firstTagValue(tags, "d")
  const publishedAt = firstTagValue(tags, "published_at")

  const topicTags = unique([...detail.hashtags, ...tagValues(tags, "t")])
  const eRefs = unique(tagValues(tags, "e"))
  const pRefs = unique(tagValues(tags, "p"))
  const aRefs = unique(tagValues(tags, "a"))
  const qRefs = unique(tagValues(tags, "q"))
  const rRefs = unique(tagValues(tags, "r"))
  const kRefs = unique(tagValues(tags, "k"))

  const imeta = parseImetaResources(tags)
  const embeddedRepost = parseEmbeddedRepost(event.content, event.kind)

  const imageURLs = unique([...detail.image_urls, ...imeta.imageURLs])
  const mediaURLs = unique([...imeta.mediaURLs, ...parseMediaURLsFromTags(tags)])
  const videoURL = pickVideoURL(mediaURLs)
  const videoPoster = imageURLs[0] ?? ""

  const rootRef = tags.find((tag) => tag[0] === "e" && tag[3] === "root")?.[1] || ""
  const replyRef = tags.find((tag) => tag[0] === "e" && tag[3] === "reply")?.[1] || ""
  const targetEventID = eRefs[0] ?? ""
  const targetEventQuery = useEventDetail(targetEventID)
  const kindLabel = kindLabels[event.kind] ?? "kind especializado"

  return (
    <div className="space-y-6">
      <PageHeader
        actions={
          <>
            <Button
              onClick={async () => {
                await navigator.clipboard.writeText(event.id)
                toast.success("Event ID copiado.")
              }}
              variant="outline"
            >
              <Copy className="size-4" />
              Copiar ID
            </Button>
            <Button asChild variant="outline">
              <Link params={{ pubkey: event.pubkey }} to="/users/$pubkey">Ver Usuario</Link>
            </Button>
            {(event.kind === 7 || event.kind === 6 || event.kind === 16) && targetEventID ? (
              <Button asChild variant="outline">
                <Link params={{ eventId: targetEventID }} to="/events/$eventId">
                  <StepForward className="size-4" />
                  {event.kind === 7 ? "Seguir evento alvo" : "Evento original"}
                </Link>
              </Button>
            ) : null}
            <BanUserDialog contextEventId={event.id} defaultPubkey={event.pubkey} defaultReason={`acao originada do evento ${shortenId(event.id, 10, 4)}`} triggerLabel="Banir usuario" triggerVariant="warning" />
          </>
        }
        description="Renderizacao de evento com autor, identificadores Nostr e contexto moderativo."
        title="Visualizacao de evento"
      />

      <div className="grid gap-4 xl:grid-cols-[1.3fr_0.7fr]">
        <Card>
          <CardHeader>
            <CardTitle>Evento</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              <Badge variant="muted">kind {event.kind}</Badge>
              <Badge variant="muted">{kindLabel}</Badge>
              <Badge variant="muted">{formatDateTime(event.created_at)}</Badge>
              {detail.author?.nip05 ? (
                <TooltipProvider delayDuration={150}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Link className="underline decoration-dotted underline-offset-2" params={{ pubkey: event.pubkey }} to="/users/$pubkey">
                        {detail.author.display_name || `Autor ${shortenId(event.pubkey, 12, 4)}`}
                      </Link>
                    </TooltipTrigger>
                    <TooltipContent>{detail.author.nip05}</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              ) : (
                <Link className="underline decoration-dotted underline-offset-2" params={{ pubkey: event.pubkey }} to="/users/$pubkey">
                  {detail.author?.display_name || `Autor ${shortenId(event.pubkey, 12, 4)}`}
                </Link>
              )}
            </div>

            {contentWarning ? <Badge variant="warning">content-warning: {contentWarning}</Badge> : null}

            {title || summary || alt || publishedAt || eventD ? (
              <div className="space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/30 p-4 text-sm">
                {title ? <p><span className="font-semibold">Titulo:</span> {title}</p> : null}
                {summary ? <p><span className="font-semibold">Resumo:</span> {summary}</p> : null}
                {alt ? <p><span className="font-semibold">Alt:</span> {alt}</p> : null}
                {publishedAt ? <p><span className="font-semibold">Publicado em:</span> {formatDateTime(Number(publishedAt) || event.created_at)}</p> : null}
                {eventD ? <p className="break-all"><span className="font-semibold">d-tag:</span> {eventD}</p> : null}
              </div>
            ) : null}

            <p className="whitespace-pre-wrap break-words rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/50 p-4 text-sm leading-relaxed text-foreground">
              {event.content || "(evento sem conteudo textual)"}
            </p>

            {topicTags.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {topicTags.map((tag) => (
                  <Link key={tag} search={{ tags: tag }} to="/events/search">
                    <Badge variant="muted">#{tag}</Badge>
                  </Link>
                ))}
              </div>
            ) : null}

            {imageURLs.length > 0 ? (
              <div className="grid gap-3 sm:grid-cols-2">
                {imageURLs.map((url) => (
                  <a href={url} key={url} rel="noreferrer" target="_blank">
                    <img alt="Imagem do evento" className="h-52 w-full rounded-md border border-border object-cover" src={url} />
                  </a>
                ))}
              </div>
            ) : null}

            {(event.kind === 21 || event.kind === 22 || event.kind === 34235) && videoURL ? (
              <div className="space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/20 p-4">
                <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Player de video</p>
                <video className="max-h-[420px] w-full rounded-md border border-border bg-black" controls poster={videoPoster || undefined} preload="metadata" src={videoURL} />
              </div>
            ) : null}

            {mediaURLs.length > 0 ? (
              <div className="space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/20 p-4">
                <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Media URLs (imeta)</p>
                <div className="space-y-2">
                  {mediaURLs.map((url) => (
                    <a className="flex items-center gap-1 break-all text-sm text-primary underline decoration-dotted underline-offset-2" href={url} key={url} rel="noreferrer" target="_blank">
                      <ExternalLink className="size-3.5" />
                      {url}
                    </a>
                  ))}
                </div>
              </div>
            ) : null}

            {embeddedRepost ? (
              <div className="space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/20 p-4">
                <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Evento repostado</p>
                <p className="text-sm text-foreground">kind {embeddedRepost.kind} · {shortenId(embeddedRepost.id, 12, 4)}</p>
                {embeddedRepost.pubkey ? (
                  <p className="text-xs text-muted-foreground">Autor: {shortenId(embeddedRepost.pubkey, 12, 4)}</p>
                ) : null}
                {embeddedRepost.content ? <p className="line-clamp-4 text-sm text-foreground">{embeddedRepost.content}</p> : null}
                <Button asChild size="sm" variant="outline">
                  <Link params={{ eventId: embeddedRepost.id }} to="/events/$eventId">Ir para evento original</Link>
                </Button>
              </div>
            ) : null}

            {event.kind === 7 && targetEventID ? (
              <div className="space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/20 p-4">
                <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">Evento alvo da reacao</p>
                {targetEventQuery.isLoading ? <p className="text-sm text-muted-foreground">Carregando evento alvo...</p> : null}
                {targetEventQuery.data ? (
                  <>
                    <p className="text-sm text-foreground">kind {targetEventQuery.data.event.kind} · {shortenId(targetEventQuery.data.event.id, 12, 4)}</p>
                    <p className="line-clamp-3 text-sm text-muted-foreground">{targetEventQuery.data.event.content || "(sem conteudo textual)"}</p>
                    <Button asChild size="sm" variant="outline">
                      <Link params={{ eventId: targetEventID }} to="/events/$eventId">Ver evento alvo completo</Link>
                    </Button>
                  </>
                ) : null}
                {targetEventQuery.isError ? <p className="text-xs text-muted-foreground">Nao foi possivel carregar o evento alvo localmente.</p> : null}
              </div>
            ) : null}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Identificadores Nostr</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {Object.entries(detail.identifiers).map(([key, value]) => (
              value ? (
                <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3" key={key}>
                  <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{key}</p>
                  <p className="mt-1 break-all font-mono text-xs text-foreground">{value}</p>
                </div>
              ) : null
            ))}
            <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
              <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Event ID</p>
              <p className="mt-1 break-all font-mono text-xs text-foreground">{event.id}</p>
            </div>

            {rootRef || replyRef ? (
              <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Thread (NIP-10)</p>
                {rootRef ? (
                  <Link className="mt-1 block break-all text-xs text-primary underline decoration-dotted underline-offset-2" params={{ eventId: rootRef }} to="/events/$eventId">
                    root: {rootRef}
                  </Link>
                ) : null}
                {replyRef ? (
                  <Link className="mt-1 block break-all text-xs text-primary underline decoration-dotted underline-offset-2" params={{ eventId: replyRef }} to="/events/$eventId">
                    reply: {replyRef}
                  </Link>
                ) : null}
              </div>
            ) : null}

            {kRefs.length > 0 ? (
              <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">k tags</p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {kRefs.map((value) => <Badge key={value} variant="muted">{value}</Badge>)}
                </div>
              </div>
            ) : null}

            {aRefs.length > 0 ? (
              <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">a references</p>
                <div className="mt-2 space-y-1">
                  {aRefs.map((value) => <p className="break-all font-mono text-xs text-foreground" key={value}>{value}</p>)}
                </div>
              </div>
            ) : null}

            {eRefs.length > 0 ? (
              <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">e references</p>
                <div className="mt-2 space-y-1">
                  {eRefs.map((value) => (
                    <Link className="block break-all text-xs text-primary underline decoration-dotted underline-offset-2" key={value} params={{ eventId: value }} to="/events/$eventId">
                      {value}
                    </Link>
                  ))}
                </div>
              </div>
            ) : null}

            {pRefs.length > 0 ? (
              <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">p references</p>
                <div className="mt-2 space-y-1">
                  {pRefs.map((value) => (
                    <Link className="block break-all text-xs text-primary underline decoration-dotted underline-offset-2" key={value} params={{ pubkey: value }} to="/users/$pubkey">
                      {value}
                    </Link>
                  ))}
                </div>
              </div>
            ) : null}

            {qRefs.length > 0 || rRefs.length > 0 ? (
              <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Links e quotes</p>
                <div className="mt-2 space-y-1">
                  {qRefs.map((value) => <p className="break-all font-mono text-xs text-foreground" key={`q-${value}`}>q: {value}</p>)}
                  {rRefs.map((value) => (
                    <a className="block break-all text-xs text-primary underline decoration-dotted underline-offset-2" href={value} key={`r-${value}`} rel="noreferrer" target="_blank">
                      r: {value}
                    </a>
                  ))}
                </div>
              </div>
            ) : null}

            {imeta.mimeTypes.length > 0 || imeta.altTexts.length > 0 ? (
              <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">imeta</p>
                {imeta.mimeTypes.length > 0 ? <p className="mt-1 break-all text-xs text-foreground">mime: {imeta.mimeTypes.join(", ")}</p> : null}
                {imeta.altTexts.length > 0 ? <p className="mt-1 break-all text-xs text-foreground">alt: {imeta.altTexts.join(" | ")}</p> : null}
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
