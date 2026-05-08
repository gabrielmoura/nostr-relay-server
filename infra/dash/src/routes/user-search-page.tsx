import { useMemo, useState } from "react"
import { Link, useSearch } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { UserAvatarChip } from "@/components/shared/user-avatar-chip"
import { VirtualizedList } from "@/components/shared/virtualized-list"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useInfiniteUserSearch } from "@/hooks/use-admin-data"
import { shortenId } from "@/lib/utils"

export function UserSearchPage() {
  const { t } = useTranslation()
  const routeSearch = useSearch({ from: "/users/search" }) as { q?: string }
  const [query, setQuery] = useState(routeSearch.q ?? "")
  const [mode, setMode] = useState<"cards" | "list" | "suspects">("cards")
  const usersQuery = useInfiniteUserSearch(query)

  const allUsers = usersQuery.data?.pages.flatMap((page) => page.items) ?? []
  const total = usersQuery.data?.pages[0]?.total ?? 0

  const users = useMemo(() => {
    if (mode === "suspects") {
      return allUsers.filter((user) => user.status === "suspect" || user.status === "banned" || Boolean(user.reason))
    }
    return allUsers
  }, [allUsers, mode])

  const bannedCount = allUsers.filter((user) => user.status === "banned").length
  const suspectOrBannedCount = allUsers.filter((user) => user.status === "suspect" || user.status === "banned" || Boolean(user.reason)).length
  const nip05Count = allUsers.filter((user) => Boolean(user.nip05)).length

  return (
    <div className="space-y-6">
      <PageHeader description={t("userSearch.description")} title={t("userSearch.title")} />


      <Card>
        <CardContent className="space-y-4 p-4">
          <div className="grid gap-3 md:grid-cols-3">
            <KpiCard label={t("userSearch.kpis.total", "Resultados totais")} value={String(total)} />
            <KpiCard label={t("userSearch.kpis.suspects", "Suspeitos / banidos")} value={String(suspectOrBannedCount)} helper={t("userSearch.kpis.bannedOnly", { count: bannedCount, defaultValue: `${bannedCount} banidos` })} />
            <KpiCard label={t("userSearch.kpis.nip05", "Com NIP-05")} value={String(nip05Count)} />
          </div>

          <div className="grid gap-3 lg:grid-cols-[1fr_auto]">
            <Input placeholder={t("userSearch.inputPlaceholder")} value={query} onChange={(event) => setQuery(event.target.value)} />
            <Button>{t("userSearch.searchProfile")}</Button>
          </div>
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="font-heading text-base font-semibold">{t("userSearch.foundUsers")}</p>
              <p className="text-sm text-muted-foreground">{t("userSearch.resultsCount", { shown: users.length, total: mode === "suspects" ? users.length : total })}</p>
            </div>
            <Tabs onValueChange={(value) => setMode(value as typeof mode)} value={mode}>
              <TabsList>
                <TabsTrigger value="cards">Cards</TabsTrigger>
                <TabsTrigger value="list">{t("userSearch.list")}</TabsTrigger>
                <TabsTrigger value="suspects">{t("userSearch.suspects")}</TabsTrigger>
              </TabsList>
            </Tabs>
          </div>

          {usersQuery.isLoading && users.length === 0 ? <LoadingPanel label={t("userSearch.loading")} /> : null}
          {usersQuery.isError ? <ErrorPanel description={t("userSearch.errorDescription")} onRetry={() => void usersQuery.refetch()} title={t("userSearch.errorTitle")} /> : null}
          {!usersQuery.isLoading && !usersQuery.isError && users.length === 0 ? <EmptyPanel description={t("userSearch.emptyDescription")} title={t("userSearch.emptyTitle")} /> : null}

          {!usersQuery.isLoading && !usersQuery.isError && users.length > 0 ? (
            <VirtualizedList
              estimateSize={mode === "cards" ? 150 : mode === "list" ? 88 : 132}
              fetchMore={() => void usersQuery.fetchNextPage()}
              hasMore={mode === "suspects" ? false : usersQuery.hasNextPage}
              isFetchingMore={usersQuery.isFetchingNextPage}
              items={users}
              renderItem={(user) => {
                if (mode === "list") {
                  return (
                    <div className="grid grid-cols-[2fr_1fr_1fr_auto] items-center gap-3 rounded-[calc(var(--radius)-0.25rem)] border border-border px-4 py-3 text-sm">
                      <UserAvatarChip compact subtitle={user.nip05 ?? user.metadata} user={user} />
                      <div className="text-muted-foreground">{shortenId(user.pubkey, 12, 4)}</div>
                      <div className="text-muted-foreground">{user.status ?? t("userSearch.active")}</div>
                      <Button asChild size="sm" variant="outline">
                        <Link params={{ pubkey: user.pubkey }} to="/users/$pubkey">{t("userSearch.view")}</Link>
                      </Button>
                    </div>
                  )
                }

                if (mode === "suspects") {
                  return (
                    <div className="rounded-[calc(var(--radius)-0.25rem)] border border-orange-200 bg-orange-50/60 p-4">
                      <div className="flex items-start justify-between gap-3">
                        <UserAvatarChip subtitle={user.reason ?? user.metadata ?? t("userSearch.moderationSignal")} user={user} />
                        <Badge variant={user.status === "banned" ? "danger" : "warning"}>{user.status === "banned" ? t("userSearch.banned") : t("userSearch.investigate")}</Badge>
                      </div>
                      <div className="mt-3 flex flex-wrap gap-2 text-xs">
                        {user.reason ? <Badge variant="danger">{t("userSearch.reason")}: {user.reason}</Badge> : null}
                        {user.nip05 ? <Badge variant="muted">nip05 {user.nip05}</Badge> : null}
                      </div>
                      <div className="mt-3 flex gap-2">
                        <Button asChild size="sm" variant="outline">
                          <Link params={{ pubkey: user.pubkey }} to="/users/$pubkey">{t("userSearch.openProfile")}</Link>
                        </Button>
                        <Button asChild size="sm" variant="warning">
                          <Link search={{ q: user.pubkey }} to="/events/search">{t("userSearch.monitorEvents")}</Link>
                        </Button>
                      </div>
                    </div>
                  )
                }

                return (
                  <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-card p-4">
                    <UserAvatarChip subtitle={user.nip05 ?? user.metadata} user={user} />
                    <div className="mt-3 flex flex-wrap gap-2 text-xs">
                      {user.trustScore != null ? <Badge variant="muted">trust {user.trustScore.toFixed(2)}</Badge> : null}
                      {user.relayCount != null ? <Badge variant="muted">relays {user.relayCount}</Badge> : null}
                      {user.followers != null ? <Badge variant="muted">follows {user.followers}</Badge> : null}
                    </div>
                    <div className="mt-4 flex flex-wrap gap-2">
                      <Button asChild size="sm" variant="outline">
                        <Link params={{ pubkey: user.pubkey }} to="/users/$pubkey">{t("userSearch.viewProfile")}</Link>
                      </Button>
                      <Button asChild size="sm" variant={user.status === "suspect" || user.status === "banned" ? "warning" : "secondary"}>
                        <Link search={{ q: user.pubkey }} to="/events/search">{t("userSearch.monitor")}</Link>
                      </Button>
                    </div>
                  </div>
                )
              }}
              total={mode === "suspects" ? users.length : total}
            />
          ) : null}
        </CardContent>
      </Card>
    </div>
  )
}

function KpiCard({ label, value, helper }: { label: string; value: string; helper?: string }) {
  return (
    <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-background/80 px-4 py-4">
      <p className="text-xs uppercase tracking-[0.14em] text-muted-foreground">{label}</p>
      <p className="mt-2 font-heading text-2xl text-foreground">{value}</p>
      {helper ? <p className="mt-1 text-xs text-muted-foreground">{helper}</p> : null}
    </div>
  )
}
