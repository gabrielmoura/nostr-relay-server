import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { Ban, ShieldX, WifiOff } from "lucide-react"

import { NIP86ActionToolbar } from "@/components/features/nip86/nip86-action-toolbar"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { useBlockNIP86IPMutation, useInfiniteNIP86BlockedIPs, useUnblockNIP86IPMutation } from "@/hooks/use-admin-data"

export function BlockedIPsPanel() {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const [ip, setIP] = useState("")
  const [reason, setReason] = useState("")
  const recordsQuery = useInfiniteNIP86BlockedIPs(query)
  const blockMutation = useBlockNIP86IPMutation()
  const unblockMutation = useUnblockNIP86IPMutation()

  const items = useMemo(() => recordsQuery.data?.pages.flatMap((page) => page.items) ?? [], [recordsQuery.data])

  return (
    <Card className="overflow-hidden border-destructive/15 bg-card/95">
      <CardHeader>
        <CardTitle>{t("nip86.blockedIps.title")}</CardTitle>
        <CardDescription>{t("nip86.blockedIps.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <NIP86ActionToolbar
          action={
            <Button
              disabled={!ip.trim() || blockMutation.isPending}
              onClick={() => blockMutation.mutate({ ip, payload: { reason } })}
              type="button"
              variant="destructive"
            >
              <Ban className="size-4" />
              {blockMutation.isPending ? t("common.saving") : t("nip86.blockedIps.blockAction")}
            </Button>
          }
          onChange={setQuery}
          placeholder={t("nip86.blockedIps.searchPlaceholder")}
          value={query}
        />

        <div className="grid gap-3 lg:grid-cols-[1fr_1.6fr]">
          <Input onChange={(event) => setIP(event.target.value)} placeholder={t("nip86.blockedIps.ipPlaceholder")} value={ip} />
          <Input onChange={(event) => setReason(event.target.value)} placeholder={t("nip86.blockedIps.reasonPlaceholder")} value={reason} />
        </div>

        <div className="flex items-start gap-3 rounded-[calc(var(--radius)-0.15rem)] border border-orange-200 bg-orange-50 px-4 py-3 text-sm text-orange-800">
          <WifiOff className="mt-0.5 size-4 shrink-0" />
          <p>{t("nip86.blockedIps.disconnectHint")}</p>
        </div>

        {recordsQuery.isLoading ? <LoadingPanel label={t("nip86.blockedIps.loading")} /> : null}
        {recordsQuery.isError ? <ErrorPanel description={t("nip86.blockedIps.errorDescription")} onRetry={() => void recordsQuery.refetch()} title={t("nip86.blockedIps.errorTitle")} /> : null}
        {!recordsQuery.isLoading && !recordsQuery.isError && items.length === 0 ? <EmptyPanel description={t("nip86.blockedIps.emptyDescription")} title={t("nip86.blockedIps.emptyTitle")} /> : null}

        {items.length > 0 ? (
          <div className="space-y-3">
            {items.map((item) => (
              <div className="flex flex-col gap-3 rounded-[calc(var(--radius)-0.15rem)] border border-border/80 bg-background/70 px-4 py-4 md:flex-row md:items-center md:justify-between" key={item.ip}>
                <div className="space-y-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="font-mono text-sm font-semibold text-foreground">{item.ip}</p>
                    <Badge variant="danger">BLOCKED</Badge>
                  </div>
                  <p className="text-sm text-muted-foreground">{item.reason || t("nip86.shared.noReason")}</p>
                </div>
                <Button onClick={() => unblockMutation.mutate(item.ip)} size="sm" type="button" variant="outline">
                  <ShieldX className="size-4" />
                  {t("nip86.blockedIps.unblockAction")}
                </Button>
              </div>
            ))}
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}
