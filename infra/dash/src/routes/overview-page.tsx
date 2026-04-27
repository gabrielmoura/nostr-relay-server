import { useMemo } from "react"
import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"
import { ArrowRight, Search, ShieldCheck } from "lucide-react"

import { BanUserDialog } from "@/components/features/ban-user-dialog"
import { MetricCard } from "@/components/shared/metric-card"
import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { UserAvatarChip } from "@/components/shared/user-avatar-chip"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { useInfiniteBannedUsers, useInfiniteConnections, useInfiniteLoggedUsers, useRelayOverview, useStreamStatus } from "@/hooks/use-admin-data"

export function OverviewPage() {
  const { t } = useTranslation()
  const overview = useRelayOverview()
  const loggedUsers = useInfiniteLoggedUsers()
  const bannedUsers = useInfiniteBannedUsers("")
  const activeConnections = useInfiniteConnections("active")
  const stream = useStreamStatus()

  const topLoggedUsers = useMemo(() => loggedUsers.data?.pages.flatMap((page) => page.items).slice(0, 3) ?? [], [loggedUsers.data])
  const topBannedUsers = useMemo(() => bannedUsers.data?.pages.flatMap((page) => page.items).slice(0, 2) ?? [], [bannedUsers.data])
  const activeConnectionItems = activeConnections.data?.pages.flatMap((page) => page.items).slice(0, 4) ?? []

  if (overview.isLoading) {
    return <LoadingPanel label={t("overview.loading")} />
  }

  if (overview.isError || !overview.data) {
    return (
      <ErrorPanel
        description={t("overview.errorDescription")}
        onRetry={() => void overview.refetch()}
        title={t("overview.errorTitle")}
      />
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        actions={
          <>
            <Button asChild variant="outline">
              <Link to="/events/search">
                <Search className="size-4" />
                {t("overview.searchEvents")}
              </Link>
            </Button>
            <BanUserDialog triggerLabel={t("moderation.ban.trigger")} />
          </>
        }
        description={t("overview.description")}
        title={t("overview.title")}
      />

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {overview.data.cards.map((card) => (
          <MetricCard card={card} key={card.label} />
        ))}
      </section>

      <section className="grid gap-4 xl:grid-cols-[1.2fr_1fr]">
        <Card>
          <CardHeader className="gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle>{t("overview.loggedUsersTitle")}</CardTitle>
              <CardDescription>{t("overview.loggedUsersDescription")}</CardDescription>
            </div>
            <Button asChild size="sm" variant="ghost">
              <Link to="/users/logged">
                {t("overview.viewDetails")}
                <ArrowRight className="size-4" />
              </Link>
            </Button>
          </CardHeader>
          <CardContent className="space-y-4">
            {loggedUsers.isLoading ? <LoadingPanel label={t("overview.loadingLoggedUsers")} /> : null}
            {loggedUsers.isError ? (
              <ErrorPanel
                description="A listagem consolidada usa `/admin/connections/authed` e um enriquecimento local de perfis."
                onRetry={() => void loggedUsers.refetch()}
                title={t("overview.loggedUsersErrorTitle")}
              />
            ) : null}
            {!loggedUsers.isLoading && !loggedUsers.isError && topLoggedUsers.length === 0 ? (
              <EmptyPanel description={t("overview.loggedUsersEmptyDescription")} title={t("overview.loggedUsersEmptyTitle")} />
            ) : null}
            {!loggedUsers.isLoading && !loggedUsers.isError && topLoggedUsers.length > 0 ? (
              <div className="space-y-3">
                {topLoggedUsers.map((user) => (
                  <div className="flex items-center justify-between rounded-[calc(var(--radius)-0.2rem)] border border-border px-3 py-3" key={user.pubkey}>
                    <UserAvatarChip subtitle={t("overview.connectionsCount", { count: user.connectionCount })} user={user} />
                    <BanUserDialog defaultPubkey={user.pubkey} defaultReason="atividade suspeita" triggerLabel={t("moderation.ban.shortTrigger")} triggerVariant="warning" />
                  </div>
                ))}
              </div>
            ) : null}
          </CardContent>
        </Card>

        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>{t("overview.bannedUsersTitle")}</CardTitle>
              <CardDescription>{t("overview.bannedUsersDescription")}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {topBannedUsers.map((user) => (
                <div className="rounded-[calc(var(--radius)-0.2rem)] border border-red-100 bg-red-50 px-3 py-3" key={user.pubkey}>
                  <UserAvatarChip subtitle={`${user.reason} · ${user.source}`} user={user} />
                </div>
              ))}
              {topBannedUsers.length === 0 ? <EmptyPanel description={t("overview.bannedUsersEmptyDescription")} title={t("overview.emptyList")} /> : null}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("overview.activeConnectionsTitle")}</CardTitle>
              <CardDescription>Origem: `/admin/connections/active`.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-muted-foreground">
              {activeConnectionItems.map((connection) => (
                <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border px-3 py-2" key={connection.ws_id}>
                  <p className="font-mono text-xs text-foreground">{connection.ws_id}</p>
                  <p>{connection.ip}</p>
                  <p>{connection.authed ? t("overview.authenticated") : t("overview.anonymous")} · {t("overview.subscriptionsCount", { count: connection.subscription_count })}</p>
                </div>
              ))}
            </CardContent>
          </Card>

          <Card className="border-primary/15 bg-[linear-gradient(135deg,rgba(14,165,233,0.08),rgba(255,255,255,0.98))]">
            <CardHeader className="gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <CardTitle>{t("overview.nip86Title")}</CardTitle>
                <CardDescription>{t("overview.nip86Description")}</CardDescription>
              </div>
              <Badge variant="success">NIP-86</Badge>
            </CardHeader>
            <CardContent className="space-y-3 text-sm text-muted-foreground">
              <div className="flex items-start gap-3 rounded-[calc(var(--radius)-0.2rem)] border border-primary/10 bg-card/80 px-3 py-3">
                <ShieldCheck className="mt-0.5 size-4 text-primary" />
                <p>{t("overview.nip86Body")}</p>
              </div>
              <Button asChild className="w-full sm:w-auto" variant="outline">
                <Link to="/nip86">
                  {t("overview.nip86Action")}
                  <ArrowRight className="size-4" />
                </Link>
              </Button>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <CardTitle>Streams</CardTitle>
                <CardDescription>{t("overview.streamDescription")}</CardDescription>
              </div>
              <Button asChild size="sm" variant="ghost">
                <Link to="/stream">
                  {t("overview.viewStream")}
                  <ArrowRight className="size-4" />
                </Link>
              </Button>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-muted-foreground">
              {stream.isLoading ? <LoadingPanel label={t("overview.loadingStream")} /> : null}
              {stream.isError ? <ErrorPanel title={t("overview.streamErrorTitle")} description={t("overview.streamErrorDescription")} onRetry={() => void stream.refetch()} /> : null}
              {stream.data ? (
                <>
                  <p>{t("overview.upstream")}: {stream.data.config.stream_up ? t("overview.active") : t("overview.off")} · {t("overview.downstream")}: {stream.data.config.stream_down ? t("overview.active") : t("overview.off")}</p>
                  <p>{t("overview.pool")}: {stream.data.pool.connected_relays}/{stream.data.pool.total_relays} {t("overview.connectedRelays")}</p>
                  <p>{t("overview.eventQueue")}: {stream.data.dispatcher.event_queue_len}/{stream.data.dispatcher.event_queue_cap}</p>
                </>
              ) : null}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("overview.clusterTopologyTitle")}</CardTitle>
              <CardDescription>{t("overview.clusterTopologyDescription")}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-muted-foreground">
              <p><strong>LB</strong> (Nginx/HAProxy) {"->"} <strong>Relay Node A</strong> + <strong>Relay Node B</strong></p>
              <p><strong>Redis</strong> {t("overview.redisRole")}</p>
              <p><strong>PostgreSQL</strong> {t("overview.postgresRole")}</p>
              <p>{t("overview.currentStreamState")}: {stream.data?.pool.connected_relays ?? 0} {t("overview.connectedRelays")}</p>
            </CardContent>
          </Card>
        </div>
      </section>
    </div>
  )
}
