import { useParams } from "@tanstack/react-router"
import { Copy } from "lucide-react"
import { toast } from "sonner"

import { BanUserDialog } from "@/components/features/ban-user-dialog"
import { UnbanUserAlert } from "@/components/features/unban-user-alert"
import { PageHeader } from "@/components/shared/page-header"
import { ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { UserAvatarChip } from "@/components/shared/user-avatar-chip"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useBanStatus, useUser } from "@/hooks/use-admin-data"
import { shortenId } from "@/lib/utils"

export function UserDetailPage() {
  const { pubkey } = useParams({ from: "/users/$pubkey" })
  const userQuery = useUser(pubkey)
  const banStatus = useBanStatus(pubkey)

  if (userQuery.isLoading) {
    return <LoadingPanel label="Carregando perfil e status de moderacao..." />
  }

  if (userQuery.isError || !userQuery.data) {
    return <ErrorPanel description="Nao foi possivel montar a visao detalhada do usuario selecionado." onRetry={() => void userQuery.refetch()} title="Falha ao carregar usuario" />
  }

  const user = userQuery.data

  return (
    <div className="space-y-6">
      <PageHeader
        actions={
          <>
            <Button asChild variant="outline">
              <a href={`/panel/events/search?q=${encodeURIComponent(pubkey)}`}>Monitorar</a>
            </Button>
            <Button
              onClick={async () => {
                await navigator.clipboard.writeText(user.npub)
                toast.success("npub copiado.")
              }}
              variant="outline"
            >
              <Copy className="size-4" />
              Copiar npub
            </Button>
            {banStatus.data?.banned ? <UnbanUserAlert pubkey={pubkey} /> : <BanUserDialog defaultPubkey={pubkey} triggerLabel="Banir usuario" triggerVariant="warning" />}
          </>
        }
        description="Detalhe do usuario com identidade resumida, metadata, status e situacao de moderacao."
        title="Detalhe de usuario"
      />

      <div className="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
        <Card>
          <CardContent className="space-y-4 p-5">
            <UserAvatarChip subtitle={user.nip05 ?? user.metadata} user={user} />
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Pubkey</p>
                <p className="mt-2 break-all font-mono text-sm text-foreground">{pubkey}</p>
              </div>
              <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">NPUB</p>
                <p className="mt-2 break-all font-mono text-sm text-foreground">{user.npub || shortenId(pubkey, 18, 4)}</p>
              </div>
              <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Confianca</p>
                <p className="mt-2 text-sm text-foreground">{user.trustScore != null ? user.trustScore.toFixed(2) : "N/D"}</p>
              </div>
              <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Metadata</p>
                <p className="mt-2 text-sm text-foreground">{user.metadata ?? "Sem metadata adicional"}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Status de moderacao</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-wrap gap-2">
              <Badge variant={banStatus.data?.banned ? "danger" : "success"}>{banStatus.data?.banned ? "banido" : "liberado"}</Badge>
              {user.status ? <Badge variant="muted">{user.status}</Badge> : null}
            </div>
            <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border p-3 text-sm text-muted-foreground">
              <p className="font-medium text-foreground">Motivo registrado</p>
                <p className="mt-1">{banStatus.data?.reason ?? user.reason ?? "Nenhum motivo registrado no backend ou no fallback local."}</p>
              </div>
            </CardContent>
          </Card>
      </div>
    </div>
  )
}
