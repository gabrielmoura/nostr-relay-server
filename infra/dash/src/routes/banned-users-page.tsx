import { useState } from "react"

import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { UserAvatarChip } from "@/components/shared/user-avatar-chip"
import { VirtualizedList } from "@/components/shared/virtualized-list"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { useInfiniteBannedUsers } from "@/hooks/use-admin-data"
import { formatDateTime } from "@/lib/utils"
import { UnbanUserAlert } from "@/components/features/unban-user-alert"

export function BannedUsersPage() {
  const [search, setSearch] = useState("")
  const query = useInfiniteBannedUsers(search)
  const pages = query.data?.pages ?? []
  const users = pages.flatMap((page) => page.items)
  const total = pages[0]?.total ?? 0

  if (query.isLoading && users.length === 0) {
    return <LoadingPanel label="Carregando lista consolidada de banimentos..." />
  }

  if (query.isError) {
    return <ErrorPanel description="A UI esta usando uma camada local isolada enquanto a API nao expõe listagem de banidos." onRetry={() => void query.refetch()} title="Falha ao carregar usuarios banidos" />
  }
  return (
    <div className="space-y-6">
      <PageHeader description="Painel de moderacao com estado persistido na interface enquanto a API nao fornece listagem nativa de banidos." title="Usuarios banidos" />

      <Input onChange={(event) => setSearch(event.target.value)} placeholder="Buscar display, pubkey, npub ou motivo" value={search} />

      {users.length === 0 ? (
        <EmptyPanel description="Nenhum banimento foi registrado localmente nesta sessao." title="Sem banimentos" />
      ) : (
        <VirtualizedList
          estimateSize={122}
          fetchMore={() => void query.fetchNextPage()}
          hasMore={query.hasNextPage}
          isFetchingMore={query.isFetchingNextPage}
          items={users}
          renderItem={(user) => (
            <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-card p-4">
              <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                <div className="space-y-3">
                  <UserAvatarChip subtitle={user.reason} user={user} />
                  <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                    <Badge variant="danger">{user.reason}</Badge>
                    {user.created_at ? <Badge variant="muted">{formatDateTime(user.created_at)}</Badge> : null}
                  </div>
                </div>
                <UnbanUserAlert pubkey={user.pubkey} />
              </div>
            </div>
          )}
          total={total}
        />
      )}
    </div>
  )
}
