import { ErrorBoundary } from "react-error-boundary"
import type { ReactNode } from "react"
import { Link, useParams } from "@tanstack/react-router"

import { Avatar } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeader } from "@/components/shared/page-header"
import { useGroup, useGroupMembers, useGroupStats } from "@/hooks/use-admin-data"

export function GroupDetailPage() {
  const { groupId } = useParams({ from: "/groups/$groupId" })
  return <div className="space-y-4"><PageHeader title="Detalhe do grupo" description={groupId} /><GroupBoundary><MetadataBlock groupId={groupId} /></GroupBoundary><div className="grid gap-4 lg:grid-cols-2"><GroupBoundary><StatsBlock groupId={groupId} /></GroupBoundary><GroupBoundary><MembersBlock groupId={groupId} /></GroupBoundary></div></div>
}

function GroupBoundary({ children }: { children: ReactNode }) { return <ErrorBoundary fallbackRender={({ resetErrorBoundary }) => <Card><CardContent className="p-4 text-sm">Não foi possível carregar este bloco. <button className="underline" onClick={resetErrorBoundary}>Tentar novamente</button></CardContent></Card>}>{children}</ErrorBoundary> }
function MetadataBlock({ groupId }: { groupId: string }) { const query = useGroup(groupId); if (query.isLoading) return <Skeleton className="h-36 w-full" />; if (!query.data) throw query.error; const group = query.data; return <Card><CardContent className="flex min-w-0 gap-4 p-4"><Avatar name={group.name || group.group_id} src={group.picture} className="size-20 shrink-0" /><div className="min-w-0"><h2 className="truncate font-heading text-xl">{group.name}</h2><p className="break-words text-sm text-muted-foreground">{group.description || "Sem descrição"}</p><div className="mt-3 flex flex-wrap gap-2"><Badge variant="muted">~ criado {group.created_at || "indisponível"}</Badge><Badge variant="muted">~ alterado {group.updated_at || "indisponível"}</Badge></div></div></CardContent></Card> }
function StatsBlock({ groupId }: { groupId: string }) { const query = useGroupStats(groupId); if (query.isLoading) return <Skeleton className="h-28 w-full" />; if (!query.data) throw query.error; return <Card><CardContent className="p-4"><p className="text-sm text-muted-foreground">Mensagens</p><p className="font-heading text-3xl">~ {query.data.message_count}</p><p className="text-xs text-muted-foreground">Valor aproximado; calculado em {query.data.computed_at}</p></CardContent></Card> }
function MembersBlock({ groupId }: { groupId: string }) { const query = useGroupMembers(groupId); if (query.isLoading) return <Skeleton className="h-48 w-full" />; if (!query.data) throw query.error; return <Card><CardContent className="p-4"><h2 className="mb-3 font-heading">Administradores e membros</h2><div className="space-y-3">{query.data.items.map((member: any) => <div key={member.pubkey} className="flex min-w-0 items-center gap-3"><Avatar name={member.display_name} src={member.picture} className="size-8 shrink-0" /><Link to="/users/$pubkey" params={{ pubkey: member.pubkey }} className="min-w-0 flex-1 truncate text-sm underline">{member.display_name}</Link>{member.admin && <Badge>Admin</Badge>}</div>)}</div></CardContent></Card> }
