import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { VirtualizedList } from "@/components/shared/virtualized-list"
import { Badge } from "@/components/ui/badge"
import { useInfiniteConnections } from "@/hooks/use-admin-data"
import { shortenId } from "@/lib/utils"

export function LoggedConnectionsPage() {
  const query = useInfiniteConnections("authed")
  const pages = query.data?.pages ?? []
  const rows = pages.flatMap((page) => page.items)
  const total = pages[0]?.total ?? 0

  if (query.isLoading && rows.length === 0) {
    return <LoadingPanel label="Carregando conexoes autenticadas..." />
  }

  if (query.isError) {
    return <ErrorPanel description="Nao foi possivel ler `/admin/connections/authed`." onRetry={() => void query.refetch()} title="Falha ao listar conexoes logadas" />
  }

  return (
    <div className="space-y-6">
      <PageHeader description="Vista dedicada para conexoes autenticadas, util para rastrear pubkeys conectadas e densidade de subscricoes." title="Conexoes logadas" />

      {rows.length === 0 ? (
        <EmptyPanel description="Nenhuma conexao autenticada esta ativa neste momento." title="Sem conexoes logadas" />
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
                  <p className="text-sm text-foreground">{shortenId(connection.authed ?? "", 14, 4)} · {connection.ip}</p>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Badge variant="success">autenticada</Badge>
                  <Badge variant="muted">{connection.subscription_count} subscricoes</Badge>
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
