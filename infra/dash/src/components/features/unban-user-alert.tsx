import { toast } from "sonner"
import { useTranslation } from "react-i18next"

import { useUnbanMutation } from "@/hooks/use-admin-data"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"

export function UnbanUserAlert({ pubkey, label }: { pubkey: string; label?: string }) {
  const { t } = useTranslation()
  const mutation = useUnbanMutation()

  const handleConfirm = async () => {
    await mutation.mutateAsync(pubkey)
    toast.success(t("moderation.unban.success"))
  }

  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button size="sm" variant="outline">
          {label ?? t("moderation.unban.trigger")}
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("moderation.unban.title")}</AlertDialogTitle>
          <AlertDialogDescription>{t("moderation.unban.description")}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
          <AlertDialogAction disabled={mutation.isPending} onClick={handleConfirm}>
            {mutation.isPending ? t("moderation.unban.pending") : t("moderation.unban.confirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
