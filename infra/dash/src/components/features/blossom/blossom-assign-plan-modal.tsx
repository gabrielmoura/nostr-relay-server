import { useEffect, useMemo, useState } from "react"
import { Search, Trash2, Unlink2 } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Avatar } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { useAssignBlossomPlanMutation, useBlossomPlanAssignments, useInfiniteUserSearch, useUnassignBlossomPlanMutation } from "@/hooks/use-admin-data"
import { formatDateTime, shortenId } from "@/lib/utils"

type BlossomAssignPlanModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  planId: string
  planName: string
}

export function BlossomAssignPlanModal({ open, onOpenChange, planId, planName }: BlossomAssignPlanModalProps) {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const [debouncedQuery, setDebouncedQuery] = useState("")
  const [selectedPubkey, setSelectedPubkey] = useState("")
  const assignMutation = useAssignBlossomPlanMutation()
  const unassignMutation = useUnassignBlossomPlanMutation()
  const assignmentsQuery = useBlossomPlanAssignments(planId, open)

  useEffect(() => {
    const timer = window.setTimeout(() => setDebouncedQuery(query.trim()), 250)
    return () => window.clearTimeout(timer)
  }, [query])

  useEffect(() => {
    if (!open) {
      setQuery("")
      setDebouncedQuery("")
      setSelectedPubkey("")
    }
  }, [open])

  const usersQuery = useInfiniteUserSearch(debouncedQuery, { enabled: open })
  const items = useMemo(() => usersQuery.data?.pages.flatMap((page) => page.items) ?? [], [usersQuery.data])
  const assignments = assignmentsQuery.data?.items ?? []

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="w-[calc(100vw-2rem)] max-w-5xl">
        <DialogHeader>
          <DialogTitle>{t("blossom.plans.assignTitle", "Associar plano")}</DialogTitle>
          <DialogDescription>
            {t("blossom.plans.assignDescription", "Associe o plano")} <strong>{planName}</strong> {t("blossom.plans.assignDescriptionTail", "a um usuario buscando por nome, npub ou pubkey.")}
          </DialogDescription>
        </DialogHeader>

        <div className="grid min-w-0 gap-4 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)]">
          <div className="min-w-0 space-y-3">
            <div className="relative">
              <Search className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input className="pl-9" onChange={(event) => setQuery(event.target.value)} placeholder={t("blossom.plans.assignSearch", "Buscar por nome, npub ou hex") } value={query} />
            </div>

            <div className="max-h-80 space-y-2 overflow-auto pr-1">
              {items.map((item) => {
                const label = item.displayName || item.handle || shortenId(item.pubkey, 14, 8)
                const active = selectedPubkey === item.pubkey
                return (
                  <button
                    className={`flex w-full cursor-pointer items-center gap-3 rounded-[var(--radius)] border p-3 text-left transition-colors ${active ? "border-primary bg-primary/5" : "border-border hover:border-primary/30 hover:bg-muted/40"}`}
                    key={item.pubkey}
                    onClick={() => setSelectedPubkey(item.pubkey)}
                    type="button"
                  >
                    <Avatar className="size-10 shrink-0" name={label} src={item.picture} />
                    <div className="min-w-0 flex-1 overflow-hidden">
                      <p className="truncate font-medium text-foreground">{label}</p>
                      <p className="truncate text-xs text-muted-foreground">{item.npub ? shortenId(item.npub, 18, 8) : shortenId(item.pubkey, 18, 8)}</p>
                    </div>
                  </button>
                )
              })}
              {items.length === 0 && !usersQuery.isLoading ? (
                <div className="rounded-[var(--radius)] border border-dashed p-6 text-center text-sm text-muted-foreground">
                  {t("blossom.plans.assignEmpty", "Nenhum usuario encontrado para esta busca.")}
                </div>
              ) : null}
            </div>
          </div>

          <div className="min-w-0 space-y-3 rounded-[var(--radius)] border border-border bg-muted/20 p-3">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="font-medium text-foreground">{t("blossom.plans.assignedUsers", "Usuarios associados")}</p>
                <p className="text-xs text-muted-foreground">{t("blossom.plans.assignedUsersHint", "Um usuario pode ter apenas um plano ativo por vez.")}</p>
              </div>
              <span className="rounded-full bg-background px-2 py-1 text-xs text-muted-foreground">{assignments.length}</span>
            </div>

            <div className="max-h-80 min-w-0 space-y-2 overflow-auto pr-1">
              {assignments.map((assignment) => {
                const label = assignment.display_name ?? shortenId(assignment.pubkey, 14, 8)
                return (
                  <div className="flex items-center gap-3 rounded-[var(--radius)] border border-border bg-background p-3" key={assignment.pubkey}>
                    <Avatar className="size-9 shrink-0" name={label} src={assignment.picture} />
                    <div className="min-w-0 flex-1 overflow-hidden">
                      <p className="truncate text-sm font-medium text-foreground">{label}</p>
                      <p className="truncate text-xs text-muted-foreground">{assignment.npub ? shortenId(assignment.npub, 16, 6) : shortenId(assignment.pubkey, 16, 6)}</p>
                      <p className="truncate text-[11px] text-muted-foreground">{formatDateTime(assignment.assigned_at)}</p>
                    </div>
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            disabled={unassignMutation.isPending}
                            onClick={async () => {
                              try {
                                await unassignMutation.mutateAsync({ plan_id: planId, pubkey: assignment.pubkey })
                                toast.success(t("blossom.plans.unassignSuccess", "Associacao removida com sucesso."))
                              } catch (error) {
                                toast.error(error instanceof Error ? error.message : t("common.error"))
                              }
                            }}
                            size="icon"
                            type="button"
                            variant="outline"
                          >
                            <Trash2 className="size-4" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>{t("blossom.plans.unassignAction", "Remover associacao")}</TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </div>
                )
              })}
              {assignments.length === 0 ? (
                <div className="rounded-[var(--radius)] border border-dashed p-4 text-center text-sm text-muted-foreground">
                  <Unlink2 className="mx-auto mb-2 size-4" />
                  {t("blossom.plans.assignedUsersEmpty", "Nenhum usuario associado a este plano.")}
                </div>
              ) : null}
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button onClick={() => onOpenChange(false)} type="button" variant="outline">{t("common.cancel", "Cancelar")}</Button>
          <Button
            disabled={!selectedPubkey || assignMutation.isPending}
            onClick={async () => {
              try {
                await assignMutation.mutateAsync({ plan_id: planId, pubkey: selectedPubkey })
                toast.success(t("blossom.plans.assignSuccess", "Plano associado com sucesso."))
                onOpenChange(false)
              } catch (error) {
                toast.error(error instanceof Error ? error.message : t("common.error"))
              }
            }}
            type="button"
          >
            {assignMutation.isPending ? t("common.saving", "Salvando...") : t("blossom.plans.assignAction", "Associar")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
