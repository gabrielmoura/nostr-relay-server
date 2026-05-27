import { useMemo, useState } from "react"
import { Link } from "@tanstack/react-router"
import { flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { ArrowLeft, HardDrive, Link2, PencilLine, Plus, ShieldCheck, Trash2 } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { BlossomAssignPlanModal } from "@/components/features/blossom/blossom-assign-plan-modal"
import { BlossomDeletePlanModal } from "@/components/features/blossom/blossom-delete-plan-modal"
import { BlossomPlanModal } from "@/components/features/blossom/blossom-plan-modal"
import { PageHeader } from "@/components/shared/page-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { useBlossomPlans, useDeleteBlossomPlanMutation, useUpsertBlossomPlanMutation } from "@/hooks/use-admin-data"
import { formatBytes } from "@/lib/utils"
import type { BlossomPlan } from "@/types/admin"

export function BlossomPlansPage() {
  const { t } = useTranslation()
  const plansQuery = useBlossomPlans()
  const upsertPlanMutation = useUpsertBlossomPlanMutation()
  const deletePlanMutation = useDeleteBlossomPlanMutation()

  const plans = plansQuery.data?.items ?? []
  
  const [editingPlan, setEditingPlan] = useState<BlossomPlan | undefined>()
  const [deletingPlan, setDeletingPlan] = useState<BlossomPlan | undefined>()
  const [assigningPlan, setAssigningPlan] = useState<BlossomPlan | undefined>()
  const [planModalOpen, setPlanModalOpen] = useState(false)

  const summary = useMemo(() => {
    const freePlans = plans.filter((item) => item.scope === "free")
    const enabledPlans = plans.filter((item) => item.scope === "enabled_users")
    return {
      total: plans.length,
      free: freePlans.length,
      enabled: enabledPlans.length,
      defaults: plans.filter((item) => item.is_default).length,
    }
  }, [plans])

  const handleEdit = (plan: BlossomPlan) => {
    setEditingPlan(plan)
    setPlanModalOpen(true)
  }

  const handleCreate = () => {
    setEditingPlan(undefined)
    setPlanModalOpen(true)
  }

  const handleSave = async (draft: BlossomPlan) => {
    try {
      if (!draft.name || !draft.id) {
        toast.error(t("blossom.plans.validationError", "ID e Nome são obrigatórios."))
        return
      }
      await upsertPlanMutation.mutateAsync(draft)
      toast.success(t("blossom.plans.saved", "Plano salvo com sucesso."))
      setPlanModalOpen(false)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("common.error"))
    }
  }

  const handleDelete = async () => {
    if (!deletingPlan?.id) {
      return
    }
    try {
      await deletePlanMutation.mutateAsync(deletingPlan.id)
      toast.success(t("blossom.plans.deleted", "Plano removido."))
      setDeletingPlan(undefined)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("common.error"))
    }
  }

  const table = useReactTable({
    data: plans,
    columns: [
      {
        accessorKey: "name",
        header: t("common.name", "Nome"),
        cell: ({ row }: { row: { original: BlossomPlan } }) => {
          const item = row.original
          return (
            <div>
              <p className="font-medium text-foreground">{item.name}</p>
              {item.description ? <p className="mt-1 max-w-md text-sm text-muted-foreground">{item.description}</p> : null}
            </div>
          )
        },
      },
      {
        id: "scope",
        header: t("blossom.plans.scope", "Escopo"),
        cell: ({ row }: { row: { original: BlossomPlan } }) => (
          <Badge variant={row.original.scope === "enabled_users" ? "warning" : "default"}>
            {row.original.scope === "enabled_users" ? t("blossom.policy.modeEnabledLabel", "Somente habilitados") : t("blossom.policy.modeFreeLabel", "Livre")}
          </Badge>
        ),
      },
      {
        id: "storage",
        header: t("blossom.plans.storageMb", "Armazenamento"),
        cell: ({ row }: { row: { original: BlossomPlan } }) => row.original.storage_quota_bytes ? formatBytes(row.original.storage_quota_bytes) : t("blossom.plans.unlimited", "Ilimitado"),
      },
      {
        id: "egress",
        header: t("blossom.plans.egressMb", "Egress"),
        cell: ({ row }: { row: { original: BlossomPlan } }) => row.original.egress_quota_bytes ? formatBytes(row.original.egress_quota_bytes) : t("blossom.plans.unlimited", "Ilimitado"),
      },
      {
        id: "default",
        header: t("blossom.plans.default", "Padrao"),
        cell: ({ row }: { row: { original: BlossomPlan } }) => row.original.is_default ? <Badge variant="success">default</Badge> : "-",
      },
      {
        id: "actions",
        header: t("common.actions", "Acoes"),
        cell: ({ row }: { row: { original: BlossomPlan } }) => (
          <div className="flex items-center gap-2">
            <ActionIconButton label={t("blossom.plans.assignAction", "Associar")} onClick={() => setAssigningPlan(row.original)} variant="outline">
              <Link2 className="size-4" />
            </ActionIconButton>
            <ActionIconButton label={t("common.edit", "Editar")} onClick={() => handleEdit(row.original)} variant="outline">
              <PencilLine className="size-4" />
            </ActionIconButton>
            <ActionIconButton label={t("common.delete", "Remover")} onClick={() => setDeletingPlan(row.original)} variant="destructive">
              <Trash2 className="size-4" />
            </ActionIconButton>
          </div>
        ),
      },
    ] as any[],
    getCoreRowModel: getCoreRowModel(),
  })

  return (
    <div className="space-y-6">
      <PageHeader
        actions={<Button asChild type="button" variant="outline"><Link to="/blossom"><ArrowLeft className="size-4" />{t("blossom.plans.back", "Voltar ao Blossom")}</Link></Button>}
        className="rounded-[var(--radius)] border border-primary/15 bg-[linear-gradient(135deg,rgba(30,64,175,0.08),rgba(245,158,11,0.07))] p-5 panel-shadow"
        description={t("blossom.plans.description", "Defina planos nomeados, selecione padrões por escopo e configure cotas para os usuários.")}
        title={t("blossom.plans.title", "Planos e Cotas Blossom")}
      />

      <div className="grid gap-4 md:grid-cols-4">
        <SummaryCard icon={HardDrive} label={t("blossom.plans.summary.total", "Planos ativos")} value={summary.total} />
        <SummaryCard icon={ShieldCheck} label={t("blossom.plans.summary.free", "Planos livres")} value={summary.free} />
        <SummaryCard icon={ShieldCheck} label={t("blossom.plans.summary.enabled", "Planos habilitados")} value={summary.enabled} />
        <SummaryCard icon={PencilLine} label={t("blossom.plans.summary.defaults", "Defaults definidos")} value={summary.defaults} />
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between gap-3">
          <CardTitle>{t("blossom.plans.catalog", "Catálogo de planos")}</CardTitle>
          <Button onClick={handleCreate} size="sm" type="button"><Plus className="size-4" />{t("blossom.plans.create", "Criar plano")}</Button>
        </CardHeader>
        <CardContent>
          {plans.length === 0 ? (
            <div className="rounded-lg border border-dashed p-8 text-center text-muted-foreground">
              {t("blossom.plans.empty", "Nenhum plano cadastrado. Crie um novo plano para começar.")}
            </div>
          ) : (
            <Table>
              <TableHeader>
                {table.getHeaderGroups().map((headerGroup: any) => (
                  <TableRow key={headerGroup.id}>
                    {headerGroup.headers.map((header: any) => (
                      <TableHead key={header.id}>{flexRender(header.column.columnDef.header, header.getContext())}</TableHead>
                    ))}
                  </TableRow>
                ))}
              </TableHeader>
              <TableBody>
                {table.getRowModel().rows.map((row: any) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell: any) => (
                      <TableCell key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Card className="border-dashed border-primary/25 bg-primary/5">
        <CardContent className="p-4 text-sm text-muted-foreground">
          {t("blossom.plans.futureHint", "Estrutura pronta para evoluir com BUD-07, cobranca por consumo e politicas comerciais sem reescrever o catalogo de planos.")}
        </CardContent>
      </Card>

      <BlossomPlanModal onOpenChange={setPlanModalOpen} onSave={handleSave} open={planModalOpen} plan={editingPlan} saving={upsertPlanMutation.isPending} />
      <BlossomDeletePlanModal deleting={deletePlanMutation.isPending} onConfirm={() => void handleDelete()} onOpenChange={(open) => !open && setDeletingPlan(undefined)} open={Boolean(deletingPlan)} planName={deletingPlan?.name ?? ""} />
      <BlossomAssignPlanModal onOpenChange={(open) => !open && setAssigningPlan(undefined)} open={Boolean(assigningPlan)} planId={assigningPlan?.id ?? ""} planName={assigningPlan?.name ?? ""} />
    </div>
  )
}

function SummaryCard({ icon: Icon, label, value }: { icon: typeof HardDrive; label: string; value: number }) {
  return <Card><CardContent className="flex items-center gap-3 p-4"><div className="rounded-full bg-primary/10 p-2 text-primary"><Icon className="size-4" /></div><div><p className="text-sm text-muted-foreground">{label}</p><p className="font-heading text-3xl font-semibold text-foreground">{value}</p></div></CardContent></Card>
}

function ActionIconButton({ children, label, onClick, variant }: { children: React.ReactNode; label: string; onClick: () => void; variant: "outline" | "destructive" }) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button onClick={onClick} size="icon" type="button" variant={variant}>
            {children}
          </Button>
        </TooltipTrigger>
        <TooltipContent>{label}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
