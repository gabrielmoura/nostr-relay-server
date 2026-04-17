import { useState } from "react"
import { Edit3, Trash2 } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import type { NIP05Identity } from "@/types/admin"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { useDeleteNIP05Mutation, useUpsertNIP05Mutation } from "@/hooks/use-admin-data"

type NIP05EditDialogProps = {
  item: NIP05Identity
}

export function NIP05EditDialog({ item }: NIP05EditDialogProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState(item.name)

  const upsertMutation = useUpsertNIP05Mutation()
  const deleteMutation = useDeleteNIP05Mutation()

  const handleSave = async () => {
    const normalizedName = name.trim().toLowerCase()
    if (!normalizedName) {
      toast.error(t("nip05.edit.toastNameRequired"))
      return
    }

    try {
      await upsertMutation.mutateAsync({ name: normalizedName, pubkey: item.pubkey })
      toast.success(t("nip05.edit.toastUpdateSuccess"))
      setOpen(false)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("nip05.edit.toastUpdateFailed"))
    }
  }

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync(item.name)
      toast.success(t("nip05.edit.toastDeleteSuccess"))
      setOpen(false)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("nip05.edit.toastDeleteFailed"))
    }
  }

  return (
    <Dialog
      onOpenChange={(next) => {
        setOpen(next)
        if (next) {
          setName(item.name)
        }
      }}
      open={open}
    >
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <DialogTrigger asChild>
              <Button size="sm" variant="outline">
                <Edit3 className="size-4" />
                {t("common.edit")}
              </Button>
            </DialogTrigger>
          </TooltipTrigger>
          <TooltipContent>{t("nip05.edit.triggerTooltip")}</TooltipContent>
        </Tooltip>
      </TooltipProvider>

      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>{t("nip05.edit.title")}</DialogTitle>
          <DialogDescription>{t("nip05.edit.description")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">{t("nip05.edit.pubkeyLabel")}</p>
            <Input readOnly value={item.pubkey} className="font-mono text-xs" />
          </div>
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">{t("nip05.edit.nameLabel")}</p>
            <Input onChange={(event) => setName(event.target.value)} value={name} />
          </div>
        </div>

        <DialogFooter className="sm:justify-between">
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button disabled={deleteMutation.isPending} variant="destructive">
                <Trash2 className="size-4" />
                {t("common.delete")}
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{t("nip05.edit.deleteTitle")}</AlertDialogTitle>
                <AlertDialogDescription>{t("nip05.edit.deleteDescription", { name: item.name })}</AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
                <AlertDialogAction onClick={() => void handleDelete()}>
                  {t("nip05.edit.deleteConfirm")}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>

          <div className="flex gap-2">
            <Button onClick={() => setOpen(false)} type="button" variant="outline">
              {t("common.cancel")}
            </Button>
            <Button disabled={upsertMutation.isPending} onClick={() => void handleSave()} type="button">
              {upsertMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
