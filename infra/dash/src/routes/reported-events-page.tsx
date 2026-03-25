import { useState } from "react"
import { Link } from "@tanstack/react-router"

import { BanUserDialog } from "@/components/features/ban-user-dialog"
import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { VirtualizedList } from "@/components/shared/virtualized-list"
import { Avatar } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useEventReports, useInfiniteReportedEvents } from "@/hooks/use-admin-data"
import { formatDateTime, shortenId } from "@/lib/utils"

const reportTypeOptions = ["all", "spam", "nudity", "malware", "profanity", "illegal", "impersonation", "other"] as const

export function ReportedEventsPage() {
  const [query, setQuery] = useState("")
  const [reportType, setReportType] = useState<(typeof reportTypeOptions)[number]>("all")
  const [selectedEventID, setSelectedEventID] = useState<string | null>(null)

  const reportedQuery = useInfiniteReportedEvents(query, reportType === "all" ? "" : reportType)
  const pages = reportedQuery.data?.pages ?? []
  const items = pages.flatMap((page) => page.items)
  const total = pages[0]?.total ?? 0

  const reportsQuery = useEventReports(selectedEventID ?? "")
  const reports = reportsQuery.data?.pages.flatMap((page) => page.items) ?? []

  return (
    <div className="space-y-6">
      <PageHeader
        description="Moderacao de eventos reportados (NIP-56) com acesso rapido a reports, evento alvo e acao de banimento contextualizada."
        title="Eventos reportados"
      />

      <div className="grid gap-3 lg:grid-cols-[2fr_1fr]">
        <Input onChange={(event) => setQuery(event.target.value)} placeholder="Buscar por event id, npub/pubkey alvo ou texto do report" value={query} />
        <Select onValueChange={(value) => setReportType((reportTypeOptions as readonly string[]).includes(value) ? (value as (typeof reportTypeOptions)[number]) : "all")} value={reportType}>
          <SelectTrigger>
            <SelectValue placeholder="Tipo de report" />
          </SelectTrigger>
          <SelectContent>
            {reportTypeOptions.map((option) => (
              <SelectItem key={option} value={option}>{option === "all" ? "todos" : option}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {reportedQuery.isLoading && items.length === 0 ? <LoadingPanel label="Buscando eventos reportados..." /> : null}
      {reportedQuery.isError ? <ErrorPanel description="Nao foi possivel carregar eventos reportados." onRetry={() => void reportedQuery.refetch()} title="Falha ao carregar reports" /> : null}
      {!reportedQuery.isLoading && !reportedQuery.isError && items.length === 0 ? <EmptyPanel description="Nenhum evento reportado encontrado para os filtros atuais." title="Sem reports" /> : null}

      {items.length > 0 ? (
        <VirtualizedList
          estimateSize={152}
          fetchMore={() => void reportedQuery.fetchNextPage()}
          hasMore={reportedQuery.hasNextPage}
          isFetchingMore={reportedQuery.isFetchingNextPage}
          items={items}
          renderItem={(item) => (
            <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border bg-card p-4">
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-2">
                  <p className="text-sm font-semibold text-foreground">Evento alvo: {shortenId(item.target_event_id, 12, 4)}</p>
                  <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                    {item.target_author?.pubkey ? (
                      <Link className="font-medium text-foreground underline decoration-dotted underline-offset-2" params={{ pubkey: item.target_author.pubkey }} to="/users/$pubkey">
                        Autor: {item.target_author.display_name || shortenId(item.target_author.pubkey, 12, 4)}
                      </Link>
                    ) : (
                      <span>Autor: desconhecido</span>
                    )}
                    {item.target_author?.nip05 ? <Badge variant="muted">{item.target_author.nip05}</Badge> : null}
                  </div>
                  <p className="break-all text-xs text-muted-foreground">
                    nevent: {item.target_nevent || "nao disponivel"}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Criado em: {item.target_created_at ? formatDateTime(item.target_created_at) : "nao indexado"} · ultimo report: {formatDateTime(item.last_reported)}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <Badge variant="danger">{item.report_count} reports</Badge>
                    {item.report_types.map((type) => (
                      <Badge key={`${item.target_event_id}-${type}`} variant="warning">{type}</Badge>
                    ))}
                  </div>
                </div>

                <div className="flex flex-wrap gap-2">
                  <Button onClick={() => setSelectedEventID(item.target_event_id)} size="sm" variant="outline">Ver Reports</Button>
                  <Button asChild size="sm" variant="outline">
                    <Link params={{ eventId: item.target_event_id }} to="/events/$eventId">Ver Evento</Link>
                  </Button>
                  <BanUserDialog
                    contextEventId={item.target_event_id}
                    defaultPubkey={item.target_pubkey ?? ""}
                    defaultReason={`report associado ao evento ${shortenId(item.target_event_id, 10, 4)}`}
                    triggerLabel="Banir usuario"
                    triggerVariant="warning"
                  />
                </div>
              </div>
            </div>
          )}
          total={total}
        />
      ) : null}

      <Dialog onOpenChange={(open) => !open && setSelectedEventID(null)} open={Boolean(selectedEventID)}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>Reports do evento</DialogTitle>
            <DialogDescription>Lista de reports associados ao evento selecionado, com contexto de autor e tipo.</DialogDescription>
          </DialogHeader>
          {reportsQuery.isLoading && reports.length === 0 ? <LoadingPanel label="Carregando reports..." /> : null}
          {reportsQuery.isError ? <ErrorPanel description="Nao foi possivel carregar reports do evento." onRetry={() => void reportsQuery.refetch()} title="Falha nos reports" /> : null}
          <div className="max-h-[60vh] space-y-3 overflow-auto pr-1">
            {reports.map((report) => (
              <div className="rounded-md border border-border p-3" key={report.report_event_id}>
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 items-center gap-2">
                    <Avatar className="size-8" name={report.reporter_display_name || report.reporter_pubkey} src={report.reporter_picture} />
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-foreground">{report.reporter_display_name || "usuario sem nome"}</p>
                      <p className="truncate text-xs text-muted-foreground">
                        {shortenId(report.reporter_npub || report.reporter_pubkey, 14, 4)}
                      </p>
                    </div>
                  </div>
                  <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                    <Badge variant="muted">{report.report_type || "other"}</Badge>
                    <span>{formatDateTime(report.created_at)}</span>
                  </div>
                </div>
                <p className="mt-2 text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">Motivo</p>
                <p className="mt-1 text-sm text-foreground">{report.content || "(sem comentario adicional)"}</p>
                <p className="mt-2 text-xs text-muted-foreground">
                  Report event: {shortenId(report.report_event_id, 12, 4)}
                </p>
              </div>
            ))}
            {!reportsQuery.isLoading && reports.length === 0 ? <EmptyPanel description="Este evento ainda nao possui reports carregados." title="Sem reports" /> : null}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
