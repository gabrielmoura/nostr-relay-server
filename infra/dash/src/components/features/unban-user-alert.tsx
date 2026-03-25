import { toast } from "sonner"

import { useUnbanMutation } from "@/hooks/use-admin-data"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"

export function UnbanUserAlert({ pubkey, label = "Desbanir" }: { pubkey: string; label?: string }) {
  const mutation = useUnbanMutation()

  const handleConfirm = async () => {
    await mutation.mutateAsync(pubkey)
    toast.success("Usuario removido da lista de banidos.")
  }

  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button size="sm" variant="outline">
          {label}
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Remover banimento?</AlertDialogTitle>
          <AlertDialogDescription>Esta acao chama `DELETE /admin/users/:pubkey/ban` quando disponivel e sincroniza a interface local.</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancelar</AlertDialogCancel>
          <AlertDialogAction disabled={mutation.isPending} onClick={handleConfirm}>
            {mutation.isPending ? "Removendo..." : "Confirmar"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
