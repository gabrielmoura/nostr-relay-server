import { useTranslation } from "react-i18next"

import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"

type LabelsHelpDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function LabelsHelpDialog({ open, onOpenChange }: LabelsHelpDialogProps) {
  const { t } = useTranslation()

  const sections = [
    {
      title: t("labels.help.targetType.title", "Tipo de alvo"),
      description: t("labels.help.targetType.description", "Define se o label aponta para um evento, uma pubkey, um address NIP-33, uma referencia externa ou um topico."),
    },
    {
      title: t("labels.help.namespace.title", "Namespace"),
      description: t("labels.help.namespace.description", "Agrupa a ontologia do label. Use os namespaces predefinidos quando quiser interoperabilidade e um namespace customizado quando a taxonomia for interna."),
    },
    {
      title: t("labels.help.targetValue.title", "Valor do alvo"),
      description: t("labels.help.targetValue.description", "Aceita o valor canonico do alvo e, quando aplicavel, tambem NIP-19 como note/nevent, npub/nprofile e naddr. A UI normaliza antes do envio."),
    },
    {
      title: t("labels.help.labels.title", "Labels"),
      description: t("labels.help.labels.description", "Representam as classificacoes efetivas do evento kind 1985. Voce pode usar presets operacionais ou informar um label customizado."),
    },
    {
      title: t("labels.help.comment.title", "Comentário"),
      description: t("labels.help.comment.description", "Campo livre para justificativa, contexto de moderacao ou observacoes internas. O valor vai para o content do evento."),
    },
  ]

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("labels.help.title", "Ajuda do formulário de labels")}</DialogTitle>
          <DialogDescription>{t("labels.help.description", "Entenda o papel de cada campo antes de publicar um evento NIP-32.")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          {sections.map((section) => (
            <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-muted/20 p-4" key={section.title}>
              <p className="font-heading text-sm font-semibold text-foreground">{section.title}</p>
              <p className="mt-1 text-sm text-muted-foreground">{section.description}</p>
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  )
}
