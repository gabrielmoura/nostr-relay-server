import { useTranslation } from "react-i18next"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

type BlossomDeletePlanModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  planName: string
  onConfirm: () => void
  deleting: boolean
}

export function BlossomDeletePlanModal({ open, onOpenChange, planName, onConfirm, deleting }: BlossomDeletePlanModalProps) {
  const { t } = useTranslation()

  return (
    <AlertDialog onOpenChange={onOpenChange} open={open}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("blossom.plans.deleteTitle", "Excluir plano")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("blossom.plans.deleteDescription", "Tem certeza que deseja excluir o plano")} <strong>{planName}</strong>? {t("blossom.plans.deleteIrreversible", "Esta acao nao pode ser desfeita.")}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t("common.cancel", "Cancelar")}</AlertDialogCancel>
          <AlertDialogAction disabled={deleting} onClick={onConfirm}>{deleting ? t("common.saving", "Salvando...") : t("common.delete", "Remover")}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
