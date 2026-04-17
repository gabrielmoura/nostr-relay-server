import { useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"
import { useTranslation } from "react-i18next"
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
  triggerLabel,
  triggerVariant = "default",
  contextEventId,
}: BanUserDialogProps) {
  const { t } = useTranslation()
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

    toast.success(t("moderation.ban.success", { user: shortenId(values.pubkey, 10, 4) }))
    setOpen(false)
    form.reset({ pubkey: "", reason: "", mode: "permanent", periodValue: 24, periodUnit: "hours", relatedIds: "" })
  })

  return (
    <Dialog onOpenChange={setOpen} open={open}>
      <DialogTrigger asChild>
        <Button variant={triggerVariant}>{triggerLabel ?? t("moderation.ban.trigger")}</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("moderation.ban.title")}</DialogTitle>
          <DialogDescription>{t("moderation.ban.description")}</DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form className="space-y-4" onSubmit={handleSubmit}>
            <FormField
              control={form.control}
              name="pubkey"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("moderation.ban.pubkeyLabel")}</FormLabel>
                  <FormControl>
                    <Input placeholder={t("moderation.ban.pubkeyPlaceholder")} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {contextEventId ? (
                <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-muted/40 p-3 text-xs text-muted-foreground">
                {t("moderation.ban.contextEvent")} <span className="font-mono text-foreground">{contextEventId}</span>
              </div>
            ) : null}
            <FormField
              control={form.control}
              name="reason"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("moderation.ban.reasonLabel")}</FormLabel>
                  <FormControl>
                    <Textarea placeholder={t("moderation.ban.reasonPlaceholder")} {...field} />
                  </FormControl>
                  <FormDescription>{t("moderation.ban.reasonHelp")}</FormDescription>
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
                    <FormLabel>{t("moderation.ban.durationLabel")}</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder={t("moderation.ban.durationPlaceholder")} />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="permanent">{t("moderation.ban.durationPermanent")}</SelectItem>
                        <SelectItem value="temporary">{t("moderation.ban.durationTemporary")}</SelectItem>
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
                    <FormLabel>{t("moderation.ban.relatedLabel")}</FormLabel>
                    <FormControl>
                      <Input placeholder={t("moderation.ban.relatedPlaceholder")} {...field} />
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
                      <FormLabel>{t("moderation.ban.periodLabel")}</FormLabel>
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
                      <FormLabel>{t("moderation.ban.unitLabel")}</FormLabel>
                      <Select onValueChange={field.onChange} value={field.value}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder={t("moderation.ban.unitPlaceholder")} />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectItem value="hours">{t("moderation.ban.unitHours")}</SelectItem>
                          <SelectItem value="days">{t("moderation.ban.unitDays")}</SelectItem>
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
                {t("common.cancel")}
              </Button>
              <Button disabled={mutation.isPending} type="submit" variant="destructive">
                {mutation.isPending ? t("moderation.ban.pending") : t("moderation.ban.confirm")}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
