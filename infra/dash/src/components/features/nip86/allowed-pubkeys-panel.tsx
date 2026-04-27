import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { ShieldPlus, Trash2 } from "lucide-react"

import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { NIP86ActionToolbar } from "@/components/features/nip86/nip86-action-toolbar"
import { useAllowNIP86PubKeyMutation, useInfiniteNIP86AllowedPubKeys, useUnallowNIP86PubKeyMutation } from "@/hooks/use-admin-data"

export function AllowedPubkeysPanel() {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const [pubkey, setPubkey] = useState("")
  const [reason, setReason] = useState("")
  const recordsQuery = useInfiniteNIP86AllowedPubKeys(query)
  const allowMutation = useAllowNIP86PubKeyMutation()
  const unallowMutation = useUnallowNIP86PubKeyMutation()

  const items = useMemo(() => recordsQuery.data?.pages.flatMap((page) => page.items) ?? [], [recordsQuery.data])

  return (
    <Card className="overflow-hidden border-primary/15 bg-card/95">
      <CardHeader>
        <CardTitle>{t("nip86.allowed.title")}</CardTitle>
        <CardDescription>{t("nip86.allowed.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <NIP86ActionToolbar
          action={
            <Button
              onClick={() => allowMutation.mutate({ pubkey, payload: { reason } })}
              disabled={!pubkey.trim() || allowMutation.isPending}
              type="button"
            >
              <ShieldPlus className="size-4" />
              {allowMutation.isPending ? t("common.saving") : t("nip86.allowed.addAction")}
            </Button>
          }
          onChange={setQuery}
          placeholder={t("nip86.allowed.searchPlaceholder")}
          value={query}
        />

        <div className="grid gap-3 lg:grid-cols-[1.6fr_1fr]">
          <Input onChange={(event) => setPubkey(event.target.value)} placeholder={t("nip86.allowed.pubkeyPlaceholder")} value={pubkey} />
          <Input onChange={(event) => setReason(event.target.value)} placeholder={t("nip86.allowed.reasonPlaceholder")} value={reason} />
        </div>

        {recordsQuery.isLoading ? <LoadingPanel label={t("nip86.allowed.loading")} /> : null}
        {recordsQuery.isError ? <ErrorPanel description={t("nip86.allowed.errorDescription")} onRetry={() => void recordsQuery.refetch()} title={t("nip86.allowed.errorTitle")} /> : null}
        {!recordsQuery.isLoading && !recordsQuery.isError && items.length === 0 ? <EmptyPanel description={t("nip86.allowed.emptyDescription")} title={t("nip86.allowed.emptyTitle")} /> : null}

        {items.length > 0 ? (
          <div className="space-y-3">
            {items.map((item) => (
              <div className="flex flex-col gap-3 rounded-[calc(var(--radius)-0.15rem)] border border-border/80 bg-background/70 px-4 py-4 md:flex-row md:items-center md:justify-between" key={item.pubkey}>
                <div className="space-y-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="font-mono text-sm font-semibold text-foreground">{item.pubkey}</p>
                    <Badge variant="success">ALLOW</Badge>
                  </div>
                  <p className="text-sm text-muted-foreground">{item.reason || t("nip86.shared.noReason")}</p>
                </div>
                <Button onClick={() => unallowMutation.mutate(item.pubkey)} size="sm" type="button" variant="outline">
                  <Trash2 className="size-4" />
                  {t("nip86.allowed.removeAction")}
                </Button>
              </div>
            ))}
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}
