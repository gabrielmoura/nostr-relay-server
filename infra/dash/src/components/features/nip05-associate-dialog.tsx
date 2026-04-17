import { useMemo, useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"
import { useTranslation } from "react-i18next"
import { z } from "zod"
import { toast } from "sonner"

import { useUpsertNIP05Mutation } from "@/hooks/use-admin-data"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { Input } from "@/components/ui/input"

const schema = z.object({
  name: z.string().min(1, "Informe um nome NIP-05.").regex(/^[a-z0-9._-]+$/, "Use apenas a-z, 0-9, '.', '_' e '-'."),
})

type NIP05AssociateDialogProps = {
  pubkey: string
  defaultName: string
  triggerLabel?: string
}

export function NIP05AssociateDialog({ pubkey, defaultName, triggerLabel }: NIP05AssociateDialogProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const mutation = useUpsertNIP05Mutation()

  const normalizedDefaultName = useMemo(() => defaultName.trim().toLowerCase(), [defaultName])

  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: normalizedDefaultName,
    },
  })

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    if (next) {
      form.reset({ name: normalizedDefaultName })
    }
  }

  const handleSubmit = form.handleSubmit(async (values) => {
    const name = values.name.trim().toLowerCase()
    await mutation.mutateAsync({
      name,
      pubkey,
    })

    toast.success(t("nip05.associateUser.toastSuccess"))
    setOpen(false)
  })

  return (
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogTrigger asChild>
        <Button variant="secondary">{triggerLabel ?? t("nip05.associateUser.trigger")}</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("nip05.associateUser.title")}</DialogTitle>
          <DialogDescription>{t("nip05.associateUser.description")}</DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form className="space-y-4" onSubmit={handleSubmit}>
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input autoFocus placeholder="alice" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter>
              <Button onClick={() => setOpen(false)} type="button" variant="outline">
                {t("common.cancel")}
              </Button>
              <Button disabled={mutation.isPending} type="submit">
                {mutation.isPending ? t("common.saving") : t("common.save")}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
