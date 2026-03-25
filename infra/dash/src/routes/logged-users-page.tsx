import { Link } from "@tanstack/react-router"

import { BanUserDialog } from "@/components/features/ban-user-dialog"
import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { UserAvatarChip } from "@/components/shared/user-avatar-chip"
import { VirtualizedList } from "@/components/shared/virtualized-list"
import { Badge } from "@/components/ui/badge"
import { useInfiniteLoggedUsers } from "@/hooks/use-admin-data"

export function LoggedUsersPage() {
  const query = useInfiniteLoggedUsers()
  const pages = query.data?.pages ?? []
  const users = pages.flatMap((page) => page.items)
  const total = pages[0]?.total ?? 0

  if (query.isLoading && users.length === 0) {
    return <LoadingPanel label="Consolidando usuarios autenticados por pubkey..." />
  }

  if (query.isError) {
    return <ErrorPanel description="A lista depende de `/admin/connections/authed` e enriquecimento local de perfis." onRetry={() => void query.refetch()} title="Falha ao listar usuarios logados" />
  }
  return (
    <div className="space-y-6">
      <PageHeader description="Usuarios autenticados agrupados por pubkey, com avatar, identificacao curta e status operacional." title="Usuarios logados" />

      {users.length === 0 ? (
        <EmptyPanel description="Nenhum usuario autenticado foi encontrado no relay." title="Sem usuarios logados" />
      ) : (
        <VirtualizedList
          estimateSize={140}
          fetchMore={() => void query.fetchNextPage()}
          hasMore={query.hasNextPage}
          isFetchingMore={query.isFetchingNextPage}
          items={users}
          renderItem={(user) => (
            <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-card p-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <UserAvatarChip subtitle={`${user.connectionCount} conexoes`} user={user} />
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant={user.connectionState === "stable" ? "success" : "warning"}>{user.connectionState === "stable" ? "estavel" : "atencao"}</Badge>
                  {user.lastSeenAt ? <Badge variant="muted">ultimo sinal {new Date(user.lastSeenAt).toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit", timeZone: "UTC" })} UTC</Badge> : null}
                </div>
              </div>
              <div className="mt-4 flex flex-wrap gap-2">
                <Link className="inline-flex items-center rounded-[calc(var(--radius)-0.3rem)] border border-border px-3 py-2 text-sm hover:bg-muted" params={{ pubkey: user.pubkey }} to="/users/$pubkey">
                  Ver detalhes
                </Link>
                <Link className="inline-flex items-center rounded-[calc(var(--radius)-0.3rem)] border border-border px-3 py-2 text-sm hover:bg-muted" search={{ q: user.pubkey }} to="/events/search">
                  Buscar eventos
                </Link>
                <BanUserDialog defaultPubkey={user.pubkey} defaultReason="moderacao manual" triggerLabel="Banir" triggerVariant="warning" />
              </div>
            </div>
          )}
          total={total}
        />
      )}
    </div>
  )
}
