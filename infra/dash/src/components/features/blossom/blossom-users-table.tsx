import { useCallback, useMemo, useRef, useState } from "react"
import { flexRender, getCoreRowModel, useReactTable, type SortingState } from "@tanstack/react-table"
import { useVirtualizer } from "@tanstack/react-virtual"
import { BarChart3, Copy, Trash2, UserCheck, UserX } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Avatar } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { useBlossomUserDetail } from "@/hooks/use-admin-data"
import { formatBytes, formatDateTime, shortenId } from "@/lib/utils"
import type { BlossomPolicyMode, BlossomUserRecord } from "@/types/admin"

type BlossomUsersTableProps = {
  items: BlossomUserRecord[]
  onSave: (pubkey: string, enabled: boolean) => void
  onPurge: (pubkey: string) => void
  onSortChange: (sortBy?: string, sortDir?: "asc" | "desc") => void
  sortBy?: string
  sortDir?: "asc" | "desc"
  hasNextPage?: boolean
  isFetchingNextPage?: boolean
  fetchNextPage: () => void
  policyMode: BlossomPolicyMode
}

export function BlossomUsersTable({
  items,
  onSave,
  onPurge,
  onSortChange,
  sortBy,
  sortDir,
  hasNextPage,
  isFetchingNextPage,
  fetchNextPage,
  policyMode,
}: BlossomUsersTableProps) {
  const { t } = useTranslation()
  const tableContainerRef = useRef<HTMLDivElement>(null)
  const [selectedUserPubkey, setSelectedUserPubkey] = useState("")

  const sorting = useMemo<SortingState>(() => (sortBy && sortDir ? [{ id: sortBy, desc: sortDir === "desc" }] : []), [sortBy, sortDir])

  const setSorting = (updater: React.SetStateAction<SortingState>) => {
    const next = typeof updater === "function" ? updater(sorting) : updater
    if (next.length === 0) {
      onSortChange(undefined, undefined)
      return
    }
    const first = next[0]
    onSortChange(first?.id, first?.desc ? "desc" : "asc")
  }

  const columns = useMemo<any[]>(() => {
    const base = [
      {
        id: "identity",
        header: () => (
          <HeaderTooltip
            description={t("blossom.users.identityTip", "Identidade do uploader com avatar, nome preferencial e identificador nostr.")}
            label={t("blossom.users.identity", "Uploader")}
          />
        ),
        cell: ({ row }: { row: { original: BlossomUserRecord } }) => {
          const item = row.original
          const label = item.display_name ?? shortenId(item.pubkey, 14, 8)
          return (
            <div className="flex min-w-0 items-center gap-3">
              <Avatar className="size-9" name={label} src={item.picture} />
              <div className="min-w-0">
                <p className="truncate font-medium text-foreground">{label}</p>
                <p className="truncate font-mono text-xs text-muted-foreground">{shortenId(item.pubkey, 18, 8)}</p>
              </div>
            </div>
          )
        },
        enableSorting: false,
      },
      {
        id: "npub",
        header: () => (
          <HeaderTooltip
            description={t("blossom.users.npubTip", "Identificador Nostr derivado da pubkey. Clique para copiar quando disponivel.")}
            label="npub"
          />
        ),
        cell: ({ row }: { row: { original: BlossomUserRecord } }) => {
          const item = row.original
          const value = item.npub ?? item.pubkey
          return (
            <button
              className="flex max-w-[16rem] cursor-pointer items-center gap-2 font-mono text-xs text-primary"
              onClick={async () => {
                await navigator.clipboard.writeText(value)
                toast.success(t("blossom.users.copied", "Copiado."))
              }}
              type="button"
            >
              <span className="truncate">{shortenId(value, 18, 8)}</span>
              <Copy className="size-3.5 shrink-0" />
            </button>
          )
        },
        enableSorting: false,
      },
      {
        accessorKey: "storage_used_bytes",
        id: "storage_used_bytes",
        header: () => (
          <HeaderTooltip
            description={t("blossom.users.storageTip", "Espaco total utilizado no servidor por este uploader.")}
            label={t("blossom.users.storage", "Uso de disco")}
          />
        ),
        cell: ({ row }: { row: { original: BlossomUserRecord } }) => {
          const item = row.original
          return <QuotaUsage used={item.storage_used_bytes} quota={item.storage_quota_bytes} />
        },
      },
      {
        accessorKey: "monthly_egress_bytes",
        id: "monthly_egress_bytes",
        header: () => (
          <HeaderTooltip
            description={t("blossom.users.egressTip", "Volume de dados transferidos por download no mes corrente.")}
            label={t("blossom.users.egress", "Egress mes")}
          />
        ),
        cell: ({ row }: { row: { original: BlossomUserRecord } }) => {
          const item = row.original
          return <QuotaUsage used={item.monthly_egress_bytes} quota={item.egress_quota_bytes} />
        },
      },
      {
        accessorKey: "object_count",
        id: "object_count",
        header: () => (
          <HeaderTooltip
            description={t("blossom.users.objectsTip", "Quantidade total de arquivos atualmente vinculados a este uploader.")}
            label={t("blossom.users.objects", "Objetos")}
          />
        ),
        cell: ({ row }: { row: { original: BlossomUserRecord } }) => <span>{row.original.object_count}</span>,
      },
      {
        accessorKey: "last_upload_at",
        id: "last_upload_at",
        header: () => (
          <HeaderTooltip
            description={t("blossom.users.lastUploadTip", "Data do ultimo upload realizado por este usuario.")}
            label={t("blossom.users.lastUpload", "Ultimo upload")}
          />
        ),
        cell: ({ row }: { row: { original: BlossomUserRecord } }) => <span>{row.original.last_upload_at ? formatDateTime(row.original.last_upload_at) : "-"}</span>,
      },
    ]

    if (policyMode === "enabled_users") {
      base.push({
        id: "enabled",
        header: () => (
          <HeaderTooltip
            description={t("blossom.users.enabledTip", "Controle de whitelist. So aparece quando a politica efetiva exige usuarios habilitados.")}
            label={t("blossom.users.enabled", "Habilitar")}
          />
        ),
        cell: ({ row }: { row: { original: BlossomUserRecord } }) => {
          const item = row.original
          return (
            <Button onClick={() => onSave(item.pubkey, !item.enabled)} size="sm" type="button" variant={item.enabled ? "default" : "outline"}>
              {item.enabled ? <UserCheck className="size-4" /> : <UserX className="size-4" />}
              {item.enabled ? t("blossom.users.enabledState", "Habilitado") : t("blossom.users.disabledState", "Desabilitado")}
            </Button>
          )
        },
        enableSorting: false,
      })
    }

    base.push({
      id: "actions",
      header: () => <span>{t("common.actions", "Acoes")}</span>,
      cell: ({ row }: { row: { original: BlossomUserRecord } }) => (
        <div className="flex items-center justify-end gap-1">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button onClick={() => setSelectedUserPubkey(row.original.pubkey)} size="icon" type="button" variant="outline">
                  <BarChart3 className="size-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t("blossom.users.detailsAction", "Abrir consumo e detalhes")}</TooltipContent>
            </Tooltip>
          </TooltipProvider>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button onClick={() => onPurge(row.original.pubkey)} size="icon" type="button" variant="destructive">
                  <Trash2 className="size-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t("blossom.users.purge", "Purge")}</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      ),
      enableSorting: false,
    })

    return base
  }, [onPurge, onSave, policyMode, t])

  const table = useReactTable({
    data: items,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    manualSorting: true,
  })

  const rows = table.getRowModel().rows

  const rowVirtualizer = useVirtualizer({
    count: hasNextPage ? rows.length + 1 : rows.length,
    getScrollElement: () => tableContainerRef.current,
    estimateSize: () => 88,
    overscan: 10,
  })

  const handleScroll = useCallback(
    (event: React.UIEvent<HTMLDivElement>) => {
      const target = event.target as HTMLDivElement
      if (target.scrollHeight - target.scrollTop - target.clientHeight < 120 && hasNextPage && !isFetchingNextPage) {
        fetchNextPage()
      }
    },
    [fetchNextPage, hasNextPage, isFetchingNextPage],
  )

  return (
    <>
    <Card className="flex h-[640px] min-w-0 flex-col overflow-hidden">
      <div className="flex-1 overflow-auto" onScroll={handleScroll} ref={tableContainerRef}>
        <div className="relative min-w-0" style={{ height: `${rowVirtualizer.getTotalSize()}px` }}>
          <Table>
            <TableHeader className="sticky top-0 z-10 bg-card">
              {table.getHeaderGroups().map((headerGroup: any) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header: any) => (
                    <TableHead
                      className={header.column.getCanSort() ? "cursor-pointer select-none" : undefined}
                      key={header.id}
                      onClick={header.column.getToggleSortingHandler()}
                    >
                      <div className="flex items-center gap-2">
                        {flexRender(header.column.columnDef.header, header.getContext())}
                        {header.column.getIsSorted() === "asc" ? <span>↑</span> : null}
                        {header.column.getIsSorted() === "desc" ? <span>↓</span> : null}
                      </div>
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                const isLoaderRow = virtualRow.index >= rows.length
                const row = isLoaderRow ? null : rows[virtualRow.index]
                return (
                  <TableRow
                    className="absolute left-0 flex w-full items-stretch border-b bg-card"
                    key={virtualRow.key}
                    style={{ transform: `translateY(${virtualRow.start}px)`, height: `${virtualRow.size}px` }}
                  >
                    {isLoaderRow ? (
                      <TableCell className="w-full py-6 text-center" colSpan={columns.length}>
                        {isFetchingNextPage ? t("common.loading", "Carregando...") : ""}
                      </TableCell>
                    ) : (
                      row?.getVisibleCells().map((cell: any) => (
                        <TableCell className="flex min-w-0 flex-1 items-center overflow-hidden" key={cell.id}>
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </TableCell>
                      ))
                    )}
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      </div>
    </Card>
    <BlossomUserUsageDialog onOpenChange={(open) => !open && setSelectedUserPubkey("")} pubkey={selectedUserPubkey} />
    </>
  )
}

function HeaderTooltip({ label, description }: { label: string; description: string }) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger className="cursor-help font-medium underline decoration-dotted underline-offset-4">{label}</TooltipTrigger>
        <TooltipContent className="max-w-xs text-xs leading-relaxed">{description}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

function QuotaUsage({ used, quota }: { used: number; quota?: number }) {
  const percent = quota && quota > 0 ? Math.min(100, Math.round((used / quota) * 100)) : undefined
  return (
    <div className="min-w-0 max-w-[14rem] space-y-2">
      <div className="flex items-center justify-between gap-3 text-xs">
        <span className="truncate font-medium text-foreground">{formatBytes(used)}</span>
        <span className="truncate text-muted-foreground">{quota ? formatBytes(quota) : "Ilimitado"}</span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-muted">
        <div className="h-full rounded-full bg-primary transition-all" style={{ width: `${percent ?? 0}%` }} />
      </div>
    </div>
  )
}

function BlossomUserUsageDialog({ pubkey, onOpenChange }: { pubkey: string; onOpenChange: (open: boolean) => void }) {
  const { t } = useTranslation()
  const detailQuery = useBlossomUserDetail(pubkey, Boolean(pubkey))
  const user = detailQuery.data

  return (
    <Dialog onOpenChange={onOpenChange} open={Boolean(pubkey)}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("blossom.users.detailsTitle", "Consumo do usuario")}</DialogTitle>
          <DialogDescription>{t("blossom.users.detailsDescription", "Resumo de uso, cotas e atividade recente do uploader selecionado.")}</DialogDescription>
        </DialogHeader>

        {!user ? (
          <div className="rounded-[var(--radius)] border border-dashed p-6 text-center text-sm text-muted-foreground">
            {t("common.loading", "Carregando...")}
          </div>
        ) : (
          <div className="space-y-4">
            <div className="flex min-w-0 items-center gap-3 rounded-[var(--radius)] border border-border p-4">
              <Avatar className="size-12 shrink-0" name={user.display_name ?? shortenId(user.pubkey, 14, 8)} src={user.picture} />
              <div className="min-w-0 overflow-hidden">
                <p className="truncate font-medium text-foreground">{user.display_name ?? shortenId(user.pubkey, 14, 8)}</p>
                <p className="truncate text-xs text-muted-foreground">{user.npub ? shortenId(user.npub, 18, 8) : shortenId(user.pubkey, 18, 8)}</p>
              </div>
            </div>

            <div className="grid gap-4 md:grid-cols-3">
              <MetricCard label={t("blossom.users.storage", "Uso de disco")} quota={user.storage_quota_bytes} used={user.storage_used_bytes} />
              <MetricCard label={t("blossom.users.egress", "Egress mes")} quota={user.egress_quota_bytes} used={user.monthly_egress_bytes} />
              <div className="rounded-[var(--radius)] border border-border p-4">
                <p className="text-sm text-muted-foreground">{t("blossom.users.objects", "Objetos")}</p>
                <p className="mt-2 text-2xl font-semibold text-foreground">{user.object_count}</p>
                <p className="mt-2 text-xs text-muted-foreground">{user.last_upload_at ? formatDateTime(user.last_upload_at) : "-"}</p>
              </div>
            </div>

            <div className="rounded-[var(--radius)] border border-border p-4">
              <div className="mb-3 flex items-center justify-between gap-3">
                <p className="font-medium text-foreground">{t("blossom.users.recentFiles", "Arquivos recentes")}</p>
                <span className="text-xs text-muted-foreground">{user.files.length}</span>
              </div>
              <div className="space-y-2">
                {user.files.slice(0, 5).map((file) => (
                  <div className="flex min-w-0 items-center justify-between gap-3 rounded-[var(--radius)] bg-muted/30 px-3 py-2" key={file.hash}>
                    <div className="min-w-0 overflow-hidden">
                      <p className="truncate text-sm font-medium text-foreground">{file.mime_type || file.extension || shortenId(file.hash, 12, 6)}</p>
                      <p className="truncate text-xs text-muted-foreground">{shortenId(file.hash, 16, 6)}</p>
                    </div>
                    <span className="shrink-0 text-xs text-muted-foreground">{formatBytes(file.size)}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

function MetricCard({ label, used, quota }: { label: string; used: number; quota?: number }) {
  const percent = quota && quota > 0 ? Math.min(100, Math.round((used / quota) * 100)) : 0
  return (
    <div className="rounded-[var(--radius)] border border-border p-4">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="mt-2 text-lg font-semibold text-foreground">{formatBytes(used)}</p>
      <p className="text-xs text-muted-foreground">{quota ? `de ${formatBytes(quota)}` : "Ilimitado"}</p>
      <div className="mt-3 h-2 overflow-hidden rounded-full bg-muted">
        <div className="h-full rounded-full bg-primary" style={{ width: `${percent}%` }} />
      </div>
    </div>
  )
}
