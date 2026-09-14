import { Suspense, useState } from "react"
import { useTranslation } from "react-i18next"
import { ExternalLink, Trash2, Users } from "lucide-react"
import { ErrorBoundary } from "react-error-boundary"
import { Link } from "@tanstack/react-router"

import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel } from "@/components/shared/state-panels"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useInfiniteGroups, isFeatureDisabledError } from "@/hooks/use-admin-data"
import { FeatureDisabledPanel } from "@/components/shared/feature-disabled-panel"
import { Skeleton } from "@/components/ui/skeleton"
import { Avatar } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { useDeleteGroupMutation } from "@/hooks/use-admin-data"

export function GroupsPage() {
  return (
    <ErrorBoundary
      fallbackRender={({ error, resetErrorBoundary }: { error: any; resetErrorBoundary: any }) => {
        if (isFeatureDisabledError(error)) {
          return (
            <FeatureDisabledPanel
              configKey="nip29.enabled"
              description="NIP-29 (Relay-based Groups) permite criar e gerenciar grupos moderados e canais de chat diretamente no relay."
              howToEnable="nip29:
  enabled: true"
              title="Módulo de Grupos Desabilitado"
            />
          )
        }
        return (
          <div className="p-6 text-center">
            <h2 className="text-lg font-bold text-destructive">Algo deu errado</h2>
            <p className="text-muted-foreground">{error.message}</p>
            <button onClick={resetErrorBoundary} className="mt-4 text-primary underline">Tentar novamente</button>
          </div>
        )
      }}
    >
      <Suspense fallback={<GroupsSkeleton />}>
        <GroupsContent />
      </Suspense>
    </ErrorBoundary>
  )
}

function GroupsContent() {
  const { t } = useTranslation()
  const [pageSize, setPageSize] = useState("20")
  const [pendingDelete, setPendingDelete] = useState<string | null>(null)
  const [reason, setReason] = useState<"spam" | "illegal_content" | "abuse" | "other">("spam")
  const listQuery = useInfiniteGroups(Number(pageSize))
  const deleteMutation = useDeleteGroupMutation()

  const items = listQuery.data?.pages.flatMap((page) => page.items) ?? []
  const hasItems = items.length > 0

  if (!hasItems && !listQuery.isLoading) {
    return (
      <div className="space-y-6">
        <PageHeader
          description={t("groups.description")}
          title={t("groups.title")}
        />
        <EmptyPanel
          description={t("groups.empty")}
          title={t("groups.empty")}
        />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        description={t("groups.description")}
        title={t("groups.title")}
      />

      <Card>
        <CardContent className="p-4">
          <div className="mb-4 flex items-center justify-between gap-3">
            <span className="text-sm text-muted-foreground">{items.length} grupos carregados</span>
            <Select value={pageSize} onValueChange={setPageSize}>
              <SelectTrigger aria-label="Itens por página" className="w-24"><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="20">20</SelectItem><SelectItem value="50">50</SelectItem><SelectItem value="100">100</SelectItem></SelectContent>
            </Select>
          </div>
          <div className="grid gap-3 md:hidden">
            {items.map((item) => <GroupCard key={item.group_id} item={item} onDelete={setPendingDelete} />)}
          </div>
          <Table className="hidden md:table">
            <TableHeader>
              <TableRow>
                <TableHead>Imagem</TableHead>
                <TableHead>{t("groups.table.id")}</TableHead>
                <TableHead>{t("groups.table.name")}</TableHead>
                <TableHead>{t("groups.table.members")}</TableHead>
                <TableHead>{t("groups.table.privacy")}</TableHead>
                <TableHead>{t("groups.table.status")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((item) => (
                <TableRow key={item.group_id}>
                  <TableCell><Avatar name={item.name || item.group_id} src={item.picture} className="size-9" /></TableCell>
                  <TableCell className="max-w-32 truncate font-mono text-xs" title={item.group_id}>{item.group_id}</TableCell>
                  <TableCell className="max-w-48 truncate font-medium" title={item.name}>{item.name}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Users className="size-3" />
                      {item.member_count}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={item.private ? "muted" : "default"}>
                      {item.private ? t("groups.table.private") : t("groups.table.public")}
                    </Badge>
                    {item.hidden && (
                      <Badge className="ml-1" variant="danger">
                        {t("groups.table.hidden")}
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant={item.closed ? "warning" : "success"}>
                      {item.closed ? t("groups.table.closed") : t("groups.table.open")}
                    </Badge>
                  </TableCell>
                  <TableCell><div className="flex gap-1"><Button asChild size="icon" variant="ghost"><Link to="/groups/$groupId" params={{ groupId: item.group_id }} aria-label={`Abrir ${item.name}`}><ExternalLink className="size-4" /></Link></Button><Button size="icon" variant="ghost" className="text-destructive" onClick={() => setPendingDelete(item.group_id)} aria-label={`Excluir ${item.name}`}><Trash2 className="size-4" /></Button></div></TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {listQuery.hasNextPage && <Button className="mt-4 w-full" variant="outline" onClick={() => listQuery.fetchNextPage()} disabled={listQuery.isFetchingNextPage}>{listQuery.isFetchingNextPage ? "Carregando…" : "Carregar mais"}</Button>}
        </CardContent>
      </Card>
      <AlertDialog open={Boolean(pendingDelete)} onOpenChange={(open) => !open && setPendingDelete(null)}><AlertDialogContent><AlertDialogHeader><AlertDialogTitle>Excluir grupo por violação de política?</AlertDialogTitle><AlertDialogDescription>A moderação ocultará o grupo e emitirá um evento 9008 assinado pelo relay. O histórico de mensagens será preservado.</AlertDialogDescription></AlertDialogHeader><Select value={reason} onValueChange={(value) => setReason(value as typeof reason)}><SelectTrigger aria-label="Motivo"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="spam">Spam</SelectItem><SelectItem value="illegal_content">Conteúdo ilegal</SelectItem><SelectItem value="abuse">Abuso</SelectItem><SelectItem value="other">Outro</SelectItem></SelectContent></Select><AlertDialogFooter><AlertDialogCancel>Cancelar</AlertDialogCancel><AlertDialogAction disabled={!pendingDelete || deleteMutation.isPending} onClick={() => pendingDelete && void deleteMutation.mutateAsync({ groupID: pendingDelete, reason }).then(() => setPendingDelete(null), () => undefined)}>Excluir grupo</AlertDialogAction></AlertDialogFooter></AlertDialogContent></AlertDialog>
    </div>
  )
}

function GroupCard({ item, onDelete }: { item: { group_id: string; name: string; picture?: string; member_count: number; private: boolean; closed: boolean }; onDelete: (groupID: string) => void }) {
  return <div className="rounded-lg border border-border p-3"><div className="flex min-w-0 items-center gap-3"><Avatar name={item.name || item.group_id} src={item.picture} className="size-10 shrink-0" /><div className="min-w-0 flex-1"><p className="truncate font-medium">{item.name}</p><p className="truncate font-mono text-xs text-muted-foreground">{item.group_id}</p></div><Button asChild size="icon" variant="ghost"><Link to="/groups/$groupId" params={{ groupId: item.group_id }} aria-label={`Abrir ${item.name}`}><ExternalLink className="size-4" /></Link></Button><Button size="icon" variant="ghost" className="text-destructive" onClick={() => onDelete(item.group_id)} aria-label={`Excluir ${item.name}`}><Trash2 className="size-4" /></Button></div><div className="mt-3 flex gap-2 text-sm text-muted-foreground"><span className="flex items-center gap-1"><Users className="size-3" />{item.member_count}</span><Badge variant={item.private ? "muted" : "default"}>{item.private ? "Privado" : "Público"}</Badge><Badge variant={item.closed ? "warning" : "success"}>{item.closed ? "Fechado" : "Aberto"}</Badge></div></div>
}

function GroupsSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-24 w-full" />
      <Skeleton className="h-96 w-full" />
    </div>
  )
}
