import { useMemo } from "react"

import { PageHeader } from "@/components/shared/page-header"
import { ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { useStreamStatus } from "@/hooks/use-admin-data"

function pct(current: number, total: number) {
  if (total <= 0) {
    return 0
  }
  return Math.round((current / total) * 100)
}

export function StreamStatusPage() {
  const query = useStreamStatus()

  const pressure = useMemo(() => {
    const d = query.data?.dispatcher
    if (!d) {
      return { event: 0, req: 0 }
    }
    return {
      event: pct(d.event_queue_len, d.event_queue_cap),
      req: pct(d.request_queue_len, d.request_queue_cap),
    }
  }, [query.data])

  if (query.isLoading) {
    return <LoadingPanel label="Carregando status de stream..." />
  }

  if (query.isError || !query.data) {
    return <ErrorPanel title="Falha ao carregar stream" description="Nao foi possivel obter o endpoint `/admin/stream/status`." onRetry={() => void query.refetch()} />
  }

  const stream = query.data

  return (
    <div className="space-y-6">
      <PageHeader title="Streams" description="Visibilidade operacional do fluxo upstream/downstream, filas e estado do relay pool." />

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Card>
          <CardHeader>
            <CardDescription>Upstream</CardDescription>
            <CardTitle>{stream.config.stream_up ? "ativo" : "desligado"}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>Downstream</CardDescription>
            <CardTitle>{stream.config.stream_down ? "ativo" : "desligado"}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>Relays conectados</CardDescription>
            <CardTitle>{stream.pool.connected_relays}/{stream.pool.total_relays}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>Workers</CardDescription>
            <CardTitle>{stream.dispatcher.worker_count}</CardTitle>
          </CardHeader>
        </Card>
      </section>

      <section className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Filas</CardTitle>
            <CardDescription>Pressao atual das filas do dispatcher</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div>
              <p className="mb-1 flex justify-between"><span>Eventos</span><span>{stream.dispatcher.event_queue_len}/{stream.dispatcher.event_queue_cap} ({pressure.event}%)</span></p>
              <div className="h-2 rounded bg-muted"><div className="h-full rounded bg-primary" style={{ width: `${Math.max(2, pressure.event)}%` }} /></div>
            </div>
            <div>
              <p className="mb-1 flex justify-between"><span>Requests</span><span>{stream.dispatcher.request_queue_len}/{stream.dispatcher.request_queue_cap} ({pressure.req}%)</span></p>
              <div className="h-2 rounded bg-muted"><div className="h-full rounded bg-primary" style={{ width: `${Math.max(2, pressure.req)}%` }} /></div>
            </div>
            <div className="flex flex-wrap gap-2 pt-2 text-xs text-muted-foreground">
              <Badge variant="muted">drops eventos: {stream.dispatcher.dropped_event_jobs}</Badge>
              <Badge variant="muted">drops requests: {stream.dispatcher.dropped_request_jobs}</Badge>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Throughput</CardTitle>
            <CardDescription>Counters acumulados de encaminhamento</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3 text-sm">
            <div className="rounded border border-border px-3 py-2">Eventos encaminhados: <strong>{stream.counters.forwarded_events}</strong></div>
            <div className="rounded border border-border px-3 py-2">Requests encaminhadas: <strong>{stream.counters.forwarded_requests}</strong></div>
            <div className="rounded border border-border px-3 py-2">Falhas de encaminhamento: <strong>{stream.counters.forward_failures}</strong></div>
          </CardContent>
        </Card>
      </section>

      <Card>
        <CardHeader>
          <CardTitle>Relays do pool</CardTitle>
          <CardDescription>Estado por relay com contagem de falhas</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          {stream.pool.relays.map((relay) => (
            <div key={relay.url} className="rounded border border-border px-3 py-2 text-sm">
              <div className="flex items-center justify-between gap-2">
                <p className="truncate font-mono text-xs">{relay.url}</p>
                <Badge variant={relay.connected ? "success" : "warning"}>{relay.connected ? "conectado" : "offline"}</Badge>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">falhas: {relay.failure_count}{relay.last_error ? ` · ${relay.last_error}` : ""}</p>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
