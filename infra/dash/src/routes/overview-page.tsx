import { useMemo } from "react"
import { Link } from "@tanstack/react-router"
import { ArrowRight, Search } from "lucide-react"

import { BanUserDialog } from "@/components/features/ban-user-dialog"
import { MetricCard } from "@/components/shared/metric-card"
import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { UserAvatarChip } from "@/components/shared/user-avatar-chip"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { useInfiniteBannedUsers, useInfiniteConnections, useInfiniteLoggedUsers, useRelayOverview, useStreamStatus } from "@/hooks/use-admin-data"

export function OverviewPage() {
  const overview = useRelayOverview()
  const loggedUsers = useInfiniteLoggedUsers()
  const bannedUsers = useInfiniteBannedUsers("")
  const activeConnections = useInfiniteConnections("active")
  const stream = useStreamStatus()

  const topLoggedUsers = useMemo(() => loggedUsers.data?.pages.flatMap((page) => page.items).slice(0, 3) ?? [], [loggedUsers.data])
  const topBannedUsers = useMemo(() => bannedUsers.data?.pages.flatMap((page) => page.items).slice(0, 2) ?? [], [bannedUsers.data])
  const activeConnectionItems = activeConnections.data?.pages.flatMap((page) => page.items).slice(0, 4) ?? []

  if (overview.isLoading) {
    return <LoadingPanel label="Montando resumo operacional do relay..." />
  }

  if (overview.isError || !overview.data) {
    return (
      <ErrorPanel
        description="Nao foi possivel consolidar os indicadores de overview com os endpoints atuais."
        onRetry={() => void overview.refetch()}
        title="Falha ao montar overview"
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
                Buscar eventos
              </Link>
            </Button>
            <BanUserDialog triggerLabel="Banir usuario" />
          </>
        }
        description="Visao geral do relay com KPIs, moderacao imediata e contexto para navegacao operacional."
        title="Resumo do Relay"
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
              <CardTitle>Usuarios logados</CardTitle>
              <CardDescription>Snapshot de usuarios autenticados com maior atividade no relay.</CardDescription>
            </div>
            <Button asChild size="sm" variant="ghost">
              <Link to="/users/logged">
                Ver detalhes
                <ArrowRight className="size-4" />
              </Link>
            </Button>
          </CardHeader>
          <CardContent className="space-y-4">
            {loggedUsers.isLoading ? <LoadingPanel label="Buscando usuarios autenticados..." /> : null}
            {loggedUsers.isError ? (
              <ErrorPanel
                description="A listagem consolidada usa `/admin/connections/authed` e um enriquecimento local de perfis."
                onRetry={() => void loggedUsers.refetch()}
                title="Falha ao buscar usuarios logados"
              />
            ) : null}
            {!loggedUsers.isLoading && !loggedUsers.isError && topLoggedUsers.length === 0 ? (
              <EmptyPanel description="Nenhum usuario autenticado foi encontrado neste momento." title="Sem usuarios logados" />
            ) : null}
            {!loggedUsers.isLoading && !loggedUsers.isError && topLoggedUsers.length > 0 ? (
              <div className="space-y-3">
                {topLoggedUsers.map((user) => (
                  <div className="flex items-center justify-between rounded-[calc(var(--radius)-0.2rem)] border border-border px-3 py-3" key={user.pubkey}>
                    <UserAvatarChip subtitle={`${user.connectionCount} conexoes`} user={user} />
                    <BanUserDialog defaultPubkey={user.pubkey} defaultReason="atividade suspeita" triggerLabel="Banir" triggerVariant="warning" />
                  </div>
                ))}
              </div>
            ) : null}
          </CardContent>
        </Card>

        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Usuarios banidos</CardTitle>
              <CardDescription>Lista temporariamente sustentada por estado local da interface ate existir endpoint de listagem.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {topBannedUsers.map((user) => (
                <div className="rounded-[calc(var(--radius)-0.2rem)] border border-red-100 bg-red-50 px-3 py-3" key={user.pubkey}>
                  <UserAvatarChip subtitle={`${user.reason} · ${user.source}`} user={user} />
                </div>
              ))}
              {topBannedUsers.length === 0 ? <EmptyPanel description="Nenhum usuario banido foi registrado na camada de interface." title="Lista vazia" /> : null}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Conexoes ativas</CardTitle>
              <CardDescription>Origem: `/admin/connections/active`.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-muted-foreground">
              {activeConnectionItems.map((connection) => (
                <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border px-3 py-2" key={connection.ws_id}>
                  <p className="font-mono text-xs text-foreground">{connection.ws_id}</p>
                  <p>{connection.ip}</p>
                  <p>{connection.authed ? "autenticada" : "anonima"} · {connection.subscription_count} subscricoes</p>
                </div>
              ))}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <CardTitle>Streams</CardTitle>
                <CardDescription>Estado do dispatcher e relay pool.</CardDescription>
              </div>
              <Button asChild size="sm" variant="ghost">
                <Link to="/stream">
                  Ver stream
                  <ArrowRight className="size-4" />
                </Link>
              </Button>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-muted-foreground">
              {stream.isLoading ? <LoadingPanel label="Lendo status de stream..." /> : null}
              {stream.isError ? <ErrorPanel title="Falha ao ler stream" description="Endpoint `/admin/stream/status` indisponivel." onRetry={() => void stream.refetch()} /> : null}
              {stream.data ? (
                <>
                  <p>upstream: {stream.data.config.stream_up ? "ativo" : "desligado"} · downstream: {stream.data.config.stream_down ? "ativo" : "desligado"}</p>
                  <p>pool: {stream.data.pool.connected_relays}/{stream.data.pool.total_relays} relays conectados</p>
                  <p>fila eventos: {stream.data.dispatcher.event_queue_len}/{stream.data.dispatcher.event_queue_cap}</p>
                </>
              ) : null}
            </CardContent>
          </Card>
        </div>
      </section>
    </div>
  )
}
