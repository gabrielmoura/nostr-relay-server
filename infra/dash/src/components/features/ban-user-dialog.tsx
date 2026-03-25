import { useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { toast } from "sonner"

import { useBanMutation } from "@/hooks/use-admin-data"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { shortenId } from "@/lib/utils"

const schema = z.object({
  pubkey: z.string().min(6, "Informe um pubkey ou npub valido."),
  reason: z.string().min(3, "Descreva o motivo do banimento."),
  mode: z.enum(["permanent", "temporary"]),
  periodValue: z.number().min(1).optional(),
  periodUnit: z.enum(["hours", "days"]).optional(),
  relatedIds: z.string().optional(),
}).superRefine((data, ctx) => {
  if (data.mode === "temporary") {
    if (!data.periodValue || data.periodValue < 1) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: "Informe o periodo do banimento.", path: ["periodValue"] })
    }
    if (!data.periodUnit) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: "Selecione a unidade do periodo.", path: ["periodUnit"] })
    }
  }
})

type BanUserDialogProps = {
  defaultPubkey?: string
  defaultReason?: string
  triggerLabel?: string
  triggerVariant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "warning"
  contextEventId?: string
}

export function BanUserDialog({
  defaultPubkey = "",
  defaultReason = "",
  triggerLabel = "Banir usuario",
  triggerVariant = "default",
  contextEventId,
}: BanUserDialogProps) {
  const [open, setOpen] = useState(false)
  const mutation = useBanMutation()
  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      pubkey: defaultPubkey,
      reason: defaultReason,
      mode: "permanent",
      periodValue: 24,
      periodUnit: "hours",
      relatedIds: "",
    },
  })

  const mode = form.watch("mode")

  const handleSubmit = form.handleSubmit(async (values) => {
    const relatedIds = values.relatedIds
      ?.split(",")
      .map((entry: string) => entry.trim())
      .filter(Boolean)

    await mutation.mutateAsync({
      pubkey: values.pubkey,
      reason: values.reason,
      related_ids: relatedIds,
      mode: values.mode,
      period_value: values.mode === "temporary" ? values.periodValue : undefined,
      period_unit: values.mode === "temporary" ? values.periodUnit : undefined,
    })

    toast.success(`Usuario ${shortenId(values.pubkey, 10, 4)} banido com sucesso.`)
    setOpen(false)
    form.reset({ pubkey: "", reason: "", mode: "permanent", periodValue: 24, periodUnit: "hours", relatedIds: "" })
  })

  return (
    <Dialog onOpenChange={setOpen} open={open}>
      <DialogTrigger asChild>
        <Button variant={triggerVariant}>{triggerLabel}</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Banir usuario</DialogTitle>
          <DialogDescription>Confirme a moderacao com contexto suficiente para auditoria e acompanhamento do time.</DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form className="space-y-4" onSubmit={handleSubmit}>
            <FormField
              control={form.control}
              name="pubkey"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Pubkey ou npub</FormLabel>
                  <FormControl>
                    <Input placeholder="npub1..., hex pubkey ou identificador interno" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {contextEventId ? (
              <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-muted/40 p-3 text-xs text-muted-foreground">
                Acao contextualizada ao evento: <span className="font-mono text-foreground">{contextEventId}</span>
              </div>
            ) : null}
            <FormField
              control={form.control}
              name="reason"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Motivo do banimento</FormLabel>
                  <FormControl>
                    <Textarea placeholder="Explique o motivo: spam, flood, phishing, abuso de API..." {...field} />
                  </FormControl>
                  <FormDescription>Os dados sao enviados para `/admin/users/:pubkey/ban` e preservados no estado local da UI.</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField
                control={form.control}
                name="mode"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Duracao</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Selecione uma duracao" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="permanent">Permanente</SelectItem>
                        <SelectItem value="temporary">Temporario</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="relatedIds"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Eventos relacionados</FormLabel>
                    <FormControl>
                      <Input placeholder="id-1, id-2" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            {mode === "temporary" ? (
              <div className="grid gap-4 sm:grid-cols-2">
                <FormField
                  control={form.control}
                  name="periodValue"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Periodo</FormLabel>
                      <FormControl>
                        <Input
                          min={1}
                          name={field.name}
                          onBlur={field.onBlur}
                          onChange={(event) => {
                            const next = event.target.value
                            field.onChange(next === "" ? undefined : Number(next))
                          }}
                          ref={field.ref}
                          type="number"
                          value={field.value ?? ""}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="periodUnit"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Unidade</FormLabel>
                      <Select onValueChange={field.onChange} value={field.value}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="Selecione unidade" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectItem value="hours">Horas</SelectItem>
                          <SelectItem value="days">Dias</SelectItem>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            ) : null}
            <DialogFooter>
              <Button onClick={() => setOpen(false)} type="button" variant="outline">
                Cancelar
              </Button>
              <Button disabled={mutation.isPending} type="submit" variant="destructive">
                {mutation.isPending ? "Confirmando..." : "Confirmar banimento"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
