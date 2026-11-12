import { useTranslation } from "react-i18next"
import { Link } from "@tanstack/react-router"
import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { VirtualizedList } from "@/components/shared/virtualized-list"
import { Badge } from "@/components/ui/badge"
import { useInfiniteConnections } from "@/hooks/use-admin-data"
import { formatDateTime, shortenId } from "@/lib/utils"

export function LoggedConnectionsPage() {
  const { t } = useTranslation()
  const query = useInfiniteConnections("authed")
  const pages = query.data?.pages ?? []
  const rows = pages.flatMap((page) => page.items)
  const total = pages[0]?.total ?? 0

  if (query.isLoading && rows.length === 0) {
    return <LoadingPanel label={t("loggedConnections.loading")} />
  }

  if (query.isError) {
    return <ErrorPanel description={t("loggedConnections.errorDescription")} onRetry={() => void query.refetch()} title={t("loggedConnections.errorTitle")} />
  }

  return (
    <div className="space-y-6">
      <PageHeader description={t("loggedConnections.description")} title={t("loggedConnections.title")} />

      {rows.length === 0 ? (
        <EmptyPanel description={t("loggedConnections.emptyDescription")} title={t("loggedConnections.emptyTitle")} />
      ) : (
        <VirtualizedList
          estimateSize={82}
          fetchMore={() => void query.fetchNextPage()}
          hasMore={query.hasNextPage}
          isFetchingMore={query.isFetchingNextPage}
          items={rows}
          renderItem={(connection) => (
            <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-card px-4 py-3">
              <div className="flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
                <div>
                  <p className="font-mono text-xs text-foreground">{connection.ws_id}</p>
                  <p className="text-sm text-foreground">{connection.authed ? <Link className="underline decoration-dotted underline-offset-2" params={{ pubkey: connection.authed }} to="/users/$pubkey">{shortenId(connection.authed, 14, 4)}</Link> : "Usuário não identificado"} · {connection.ip}</p>
                  <p className="text-xs text-muted-foreground">Origem de rede: não identificada</p>
                  <p className="text-xs text-muted-foreground">Conectado: {connection.connected_at ? formatDateTime(connection.connected_at) : "não informado"} · Última atividade: {connection.last_seen_at ? formatDateTime(connection.last_seen_at) : "não informada"}</p>
                  {connection.user_agent ? <p className="break-all text-xs text-muted-foreground">Cliente: {connection.user_agent}</p> : null}
                </div>
                <div className="flex flex-wrap gap-2">
                  <Badge variant="success">{t("activeConnections.authenticatedShort")}</Badge>
                  <Badge variant="muted">{t("activeConnections.subscriptionsCount", { count: connection.subscription_count })}</Badge>
                </div>
              </div>
            </div>
          )}
          total={total}
        />
      )}
    </div>
  )
}
