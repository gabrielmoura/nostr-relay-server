import { useMemo } from "react"
import { useTranslation } from "react-i18next"

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
  const { t } = useTranslation()
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
    return <LoadingPanel label={t("stream.loading")} />
  }

  if (query.isError || !query.data) {
    return <ErrorPanel title={t("stream.errorTitle")} description={t("stream.errorDescription")} onRetry={() => void query.refetch()} />
  }

  const stream = query.data

  return (
    <div className="space-y-6">
      <PageHeader title={t("stream.title")} description={t("stream.description")} />

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Card>
          <CardHeader>
            <CardDescription>{t("stream.upstream")}</CardDescription>
            <CardTitle>{stream.config.stream_up ? t("stream.active") : t("stream.off")}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>{t("stream.downstream")}</CardDescription>
            <CardTitle>{stream.config.stream_down ? t("stream.active") : t("stream.off")}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>{t("stream.connectedRelays")}</CardDescription>
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
              <CardTitle>{t("stream.queues")}</CardTitle>
              <CardDescription>{t("stream.queuesDescription")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div>
              <p className="mb-1 flex justify-between"><span>{t("stream.events")}</span><span>{stream.dispatcher.event_queue_len}/{stream.dispatcher.event_queue_cap} ({pressure.event}%)</span></p>
              <div className="h-2 rounded bg-muted"><div className="h-full rounded bg-primary" style={{ width: `${Math.max(2, pressure.event)}%` }} /></div>
            </div>
            <div>
              <p className="mb-1 flex justify-between"><span>{t("stream.requests")}</span><span>{stream.dispatcher.request_queue_len}/{stream.dispatcher.request_queue_cap} ({pressure.req}%)</span></p>
              <div className="h-2 rounded bg-muted"><div className="h-full rounded bg-primary" style={{ width: `${Math.max(2, pressure.req)}%` }} /></div>
            </div>
            <div className="flex flex-wrap gap-2 pt-2 text-xs text-muted-foreground">
              <Badge variant="muted">{t("stream.droppedEvents")}: {stream.dispatcher.dropped_event_jobs}</Badge>
              <Badge variant="muted">{t("stream.droppedRequests")}: {stream.dispatcher.dropped_request_jobs}</Badge>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("stream.throughput")}</CardTitle>
            <CardDescription>{t("stream.throughputDescription")}</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3 text-sm">
            <div className="rounded border border-border px-3 py-2">{t("stream.forwardedEvents")}: <strong>{stream.counters.forwarded_events}</strong></div>
            <div className="rounded border border-border px-3 py-2">{t("stream.forwardedRequests")}: <strong>{stream.counters.forwarded_requests}</strong></div>
            <div className="rounded border border-border px-3 py-2">{t("stream.forwardFailures")}: <strong>{stream.counters.forward_failures}</strong></div>
          </CardContent>
        </Card>
      </section>

      <Card>
        <CardHeader>
          <CardTitle>{t("stream.poolRelays")}</CardTitle>
          <CardDescription>{t("stream.poolRelaysDescription")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          {stream.pool.relays.map((relay) => (
            <div key={relay.url} className="rounded border border-border px-3 py-2 text-sm">
              <div className="flex items-center justify-between gap-2">
                <p className="truncate font-mono text-xs">{relay.url}</p>
                <Badge variant={relay.connected ? "success" : "warning"}>{relay.connected ? t("stream.connected") : t("stream.offline")}</Badge>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">{t("stream.failures")}: {relay.failure_count}{relay.last_error ? ` · ${relay.last_error}` : ""}</p>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
