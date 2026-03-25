import { useMemo, useState } from "react"
import { Link, useSearch } from "@tanstack/react-router"

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

  return (
    <div className="space-y-6">
      <PageHeader description="Pesquisa por npub, nome, dominio NIP-05 e estado moderativo com visoes dedicadas por tarefa." title="Busca de usuarios" />

      <Card>
        <CardContent className="space-y-4 p-4">
          <div className="grid gap-3 lg:grid-cols-[1fr_auto]">
            <Input placeholder="npub1..., @handle ou nome completo" value={query} onChange={(event) => setQuery(event.target.value)} />
            <Button>Buscar perfil</Button>
          </div>
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="font-heading text-base font-semibold">Usuarios encontrados</p>
              <p className="text-sm text-muted-foreground">{users.length} de {mode === "suspects" ? users.length : total} resultados</p>
            </div>
            <Tabs onValueChange={(value) => setMode(value as typeof mode)} value={mode}>
              <TabsList>
                <TabsTrigger value="cards">Cards</TabsTrigger>
                <TabsTrigger value="list">Lista</TabsTrigger>
                <TabsTrigger value="suspects">Suspeitos</TabsTrigger>
              </TabsList>
            </Tabs>
          </div>

          {usersQuery.isLoading && users.length === 0 ? <LoadingPanel label="Consultando diretorio de perfis..." /> : null}
          {usersQuery.isError ? <ErrorPanel description="A busca de usuarios nao retornou dados do backend administrativo." onRetry={() => void usersQuery.refetch()} title="Falha ao buscar usuarios" /> : null}
          {!usersQuery.isLoading && !usersQuery.isError && users.length === 0 ? <EmptyPanel description="Tente buscar por nome, npub, handle ou dominio NIP-05." title="Nenhum usuario encontrado" /> : null}

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
                      <div className="text-muted-foreground">{user.status ?? "active"}</div>
                      <Button asChild size="sm" variant="outline">
                        <Link params={{ pubkey: user.pubkey }} to="/users/$pubkey">Ver</Link>
                      </Button>
                    </div>
                  )
                }

                if (mode === "suspects") {
                  return (
                    <div className="rounded-[calc(var(--radius)-0.25rem)] border border-orange-200 bg-orange-50/60 p-4">
                      <div className="flex items-start justify-between gap-3">
                        <UserAvatarChip subtitle={user.reason ?? user.metadata ?? "sinalizacao moderativa"} user={user} />
                        <Badge variant={user.status === "banned" ? "danger" : "warning"}>{user.status === "banned" ? "banido" : "investigar"}</Badge>
                      </div>
                      <div className="mt-3 flex flex-wrap gap-2 text-xs">
                        {user.reason ? <Badge variant="danger">motivo: {user.reason}</Badge> : null}
                        {user.nip05 ? <Badge variant="muted">nip05 {user.nip05}</Badge> : null}
                      </div>
                      <div className="mt-3 flex gap-2">
                        <Button asChild size="sm" variant="outline">
                          <Link params={{ pubkey: user.pubkey }} to="/users/$pubkey">Abrir perfil</Link>
                        </Button>
                        <Button asChild size="sm" variant="warning">
                          <Link search={{ q: user.pubkey }} to="/events/search">Monitorar eventos</Link>
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
                        <Link params={{ pubkey: user.pubkey }} to="/users/$pubkey">Ver perfil</Link>
                      </Button>
                      <Button asChild size="sm" variant={user.status === "suspect" || user.status === "banned" ? "warning" : "secondary"}>
                        <Link search={{ q: user.pubkey }} to="/events/search">Monitorar</Link>
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
