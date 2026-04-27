import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { ShieldBan, ShieldCheck, Trash2 } from "lucide-react"

import { NIP86ActionToolbar } from "@/components/features/nip86/nip86-action-toolbar"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { useBanNIP86EventMutation, useInfiniteNIP86BannedEvents, useUnbanNIP86EventMutation } from "@/hooks/use-admin-data"

export function BannedEventsPanel() {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const [eventID, setEventID] = useState("")
  const [reason, setReason] = useState("")
  const recordsQuery = useInfiniteNIP86BannedEvents(query)
  const banMutation = useBanNIP86EventMutation()
  const unbanMutation = useUnbanNIP86EventMutation()

  const items = useMemo(() => recordsQuery.data?.pages.flatMap((page) => page.items) ?? [], [recordsQuery.data])

  return (
    <Card className="overflow-hidden border-warning/20 bg-card/95">
      <CardHeader>
        <CardTitle>{t("nip86.bannedEvents.title")}</CardTitle>
        <CardDescription>{t("nip86.bannedEvents.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <NIP86ActionToolbar
          action={
            <Button
              disabled={!eventID.trim() || banMutation.isPending}
              onClick={() => banMutation.mutate({ eventID, payload: { reason } })}
              type="button"
              variant="warning"
            >
              <ShieldBan className="size-4" />
              {banMutation.isPending ? t("common.saving") : t("nip86.bannedEvents.banAction")}
            </Button>
          }
          onChange={setQuery}
          placeholder={t("nip86.bannedEvents.searchPlaceholder")}
          value={query}
        />

        <div className="grid gap-3 lg:grid-cols-[1.6fr_1fr]">
          <Input onChange={(event) => setEventID(event.target.value)} placeholder={t("nip86.bannedEvents.idPlaceholder")} value={eventID} />
          <Input onChange={(event) => setReason(event.target.value)} placeholder={t("nip86.bannedEvents.reasonPlaceholder")} value={reason} />
        </div>

        {recordsQuery.isLoading ? <LoadingPanel label={t("nip86.bannedEvents.loading")} /> : null}
        {recordsQuery.isError ? <ErrorPanel description={t("nip86.bannedEvents.errorDescription")} onRetry={() => void recordsQuery.refetch()} title={t("nip86.bannedEvents.errorTitle")} /> : null}
        {!recordsQuery.isLoading && !recordsQuery.isError && items.length === 0 ? <EmptyPanel description={t("nip86.bannedEvents.emptyDescription")} title={t("nip86.bannedEvents.emptyTitle")} /> : null}

        {items.length > 0 ? (
          <div className="space-y-3">
            {items.map((item) => (
              <div className="flex flex-col gap-3 rounded-[calc(var(--radius)-0.15rem)] border border-border/80 bg-background/70 px-4 py-4 md:flex-row md:items-center md:justify-between" key={item.event_id}>
                <div className="space-y-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="font-mono text-sm font-semibold text-foreground">{item.event_id}</p>
                    <Badge variant="warning">EVENT BLOCK</Badge>
                  </div>
                  <p className="text-sm text-muted-foreground">{item.reason || t("nip86.shared.noReason")}</p>
                </div>
                <div className="flex items-center gap-2">
                  <Button onClick={() => unbanMutation.mutate(item.event_id)} size="sm" type="button" variant="outline">
                    <ShieldCheck className="size-4" />
                    {t("nip86.bannedEvents.allowAction")}
                  </Button>
                  <Button onClick={() => setEventID(item.event_id)} size="sm" type="button" variant="ghost">
                    <Trash2 className="size-4" />
                    {t("nip86.bannedEvents.reuseAction")}
                  </Button>
                </div>
              </div>
            ))}
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}
