import { useState } from "react"
import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

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
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const [reportType, setReportType] = useState<(typeof reportTypeOptions)[number]>("all")
  const [selectedEventID, setSelectedEventID] = useState<string | null>(null)

  const reportedQuery = useInfiniteReportedEvents(query, reportType === "all" ? "" : reportType)
  const pages = reportedQuery.data?.pages ?? []
  const items = pages.flatMap((page) => page.items)
  const total = pages[0]?.total ?? 0
  const totalReports = items.reduce((acc, item) => acc + item.report_count, 0)
  const uniqueAuthors = new Set(items.map((item) => item.target_author?.pubkey).filter(Boolean)).size
  const topType = mostCommon(items.flatMap((item) => item.report_types))

  const reportsQuery = useEventReports(selectedEventID ?? "")
  const reports = reportsQuery.data?.pages.flatMap((page) => page.items) ?? []

  return (
    <div className="space-y-6">
      <PageHeader
        description={t("reported.description")}
        title={t("reported.title")}
      />

      <div className="grid gap-3 lg:grid-cols-[2fr_1fr]">
        <Input onChange={(event) => setQuery(event.target.value)} placeholder={t("reported.searchPlaceholder")} value={query} />
        <Select onValueChange={(value) => setReportType((reportTypeOptions as readonly string[]).includes(value) ? (value as (typeof reportTypeOptions)[number]) : "all")} value={reportType}>
          <SelectTrigger>
            <SelectValue placeholder={t("reported.typePlaceholder")} />
          </SelectTrigger>
          <SelectContent>
            {reportTypeOptions.map((option) => (
              <SelectItem key={option} value={option}>{option === "all" ? t("reported.all") : option}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="grid gap-3 md:grid-cols-3">
        <KpiCard label={t("reported.kpis.events", "Eventos reportados")} value={String(total)} />
        <KpiCard label={t("reported.kpis.reports", "Reports acumulados")} value={String(totalReports)} />
        <KpiCard label={t("reported.kpis.topType", "Tipo dominante")} value={topType || t("reported.notAvailable")} helper={t("reported.kpis.uniqueAuthors", { count: uniqueAuthors, defaultValue: `${uniqueAuthors} autores distintos` })} />
      </div>

      {reportedQuery.isLoading && items.length === 0 ? <LoadingPanel label={t("reported.loading")} /> : null}
      {reportedQuery.isError ? <ErrorPanel description={t("reported.errorDescription")} onRetry={() => void reportedQuery.refetch()} title={t("reported.errorTitle")} /> : null}
      {!reportedQuery.isLoading && !reportedQuery.isError && items.length === 0 ? <EmptyPanel description={t("reported.emptyDescription")} title={t("reported.emptyTitle")} /> : null}

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
                  <p className="text-sm font-semibold text-foreground">{t("reported.targetEvent")}: {shortenId(item.target_event_id, 12, 4)}</p>
                  <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                    {item.target_author?.pubkey ? (
                      <Link className="font-medium text-foreground underline decoration-dotted underline-offset-2" params={{ pubkey: item.target_author.pubkey }} to="/users/$pubkey">
                        {t("reported.author")}: {item.target_author.display_name || shortenId(item.target_author.pubkey, 12, 4)}
                      </Link>
                    ) : (
                      <span>{t("reported.author")}: {t("reported.unknown")}</span>
                    )}
                    {item.target_author?.nip05 ? <Badge variant="muted">{item.target_author.nip05}</Badge> : null}
                  </div>
                  <p className="break-all text-xs text-muted-foreground">
                    nevent: {item.target_nevent || t("reported.notAvailable")}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {t("reported.createdAt")}: {item.target_created_at ? formatDateTime(item.target_created_at) : t("reported.notIndexed")} · {t("reported.lastReport")}: {formatDateTime(item.last_reported)}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <Badge variant="danger">{item.report_count} reports</Badge>
                    {item.report_types.map((type) => (
                      <Badge key={`${item.target_event_id}-${type}`} variant="warning">{type}</Badge>
                    ))}
                  </div>
                </div>

                <div className="flex flex-wrap gap-2">
                  <Button onClick={() => setSelectedEventID(item.target_event_id)} size="sm" variant="outline">{t("reported.viewReports")}</Button>
                  <Button asChild size="sm" variant="outline">
                    <Link params={{ eventId: item.target_event_id }} to="/events/$eventId">{t("reported.viewEvent")}</Link>
                  </Button>
                  <BanUserDialog
                    contextEventId={item.target_event_id}
                    defaultPubkey={item.target_pubkey ?? ""}
                    defaultReason={`report associado ao evento ${shortenId(item.target_event_id, 10, 4)}`}
                    triggerLabel={t("moderation.ban.trigger")}
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
            <DialogTitle>{t("reported.modalTitle")}</DialogTitle>
            <DialogDescription>{t("reported.modalDescription")}</DialogDescription>
          </DialogHeader>
          {reportsQuery.isLoading && reports.length === 0 ? <LoadingPanel label={t("reported.modalLoading")} /> : null}
          {reportsQuery.isError ? <ErrorPanel description={t("reported.modalErrorDescription")} onRetry={() => void reportsQuery.refetch()} title={t("reported.modalErrorTitle")} /> : null}
          <div className="max-h-[60vh] space-y-3 overflow-auto pr-1">
            {reports.map((report) => (
              <div className="rounded-md border border-border p-3" key={report.report_event_id}>
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 items-center gap-2">
                    <Avatar className="size-8" name={report.reporter_display_name || report.reporter_pubkey} src={report.reporter_picture} />
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-foreground">{report.reporter_display_name || t("reported.userNoName")}</p>
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
                <p className="mt-2 text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{t("reported.reason")}</p>
                <p className="mt-1 text-sm text-foreground">{report.content || t("reported.noComment")}</p>
                <p className="mt-2 text-xs text-muted-foreground">
                  Report event: {shortenId(report.report_event_id, 12, 4)}
                </p>
              </div>
            ))}
            {!reportsQuery.isLoading && reports.length === 0 ? <EmptyPanel description={t("reported.modalEmptyDescription")} title={t("reported.emptyTitle")} /> : null}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function KpiCard({ label, value, helper }: { label: string; value: string; helper?: string }) {
  return (
    <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-card px-4 py-4 panel-shadow">
      <p className="text-xs uppercase tracking-[0.14em] text-muted-foreground">{label}</p>
      <p className="mt-2 font-heading text-2xl text-foreground">{value}</p>
      {helper ? <p className="mt-1 text-xs text-muted-foreground">{helper}</p> : null}
    </div>
  )
}

function mostCommon(values: string[]) {
  const counts = new Map<string, number>()
  for (const value of values) {
    counts.set(value, (counts.get(value) ?? 0) + 1)
  }
  return [...counts.entries()].sort((a, b) => b[1] - a[1])[0]?.[0] ?? ""
}
