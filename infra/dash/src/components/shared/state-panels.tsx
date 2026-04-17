import { AlertTriangle, Inbox, LoaderCircle } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"

export function LoadingPanel({ label = "Carregando dados do relay..." }: { label?: string }) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardContent className="flex min-h-48 flex-col items-center justify-center gap-3 text-center">
        <LoaderCircle className="size-7 animate-spin text-muted-foreground" />
        <p className="font-heading text-base font-semibold">{t("state.loading")}</p>
        <p className="text-sm text-muted-foreground">{label}</p>
      </CardContent>
    </Card>
  )
}

export function EmptyPanel({ title, description, action }: { title: string; description: string; action?: React.ReactNode }) {
  return (
    <Card>
      <CardContent className="flex min-h-48 flex-col items-center justify-center gap-3 text-center">
        <Inbox className="size-7 text-muted-foreground" />
        <p className="font-heading text-base font-semibold">{title}</p>
        <p className="max-w-md text-sm text-muted-foreground">{description}</p>
        {action}
      </CardContent>
    </Card>
  )
}

export function ErrorPanel({ title, description, onRetry }: { title: string; description: string; onRetry?: () => void }) {
  const { t } = useTranslation()
  return (
    <Card className="border-red-200">
      <CardContent className="flex min-h-48 flex-col items-center justify-center gap-3 text-center">
        <AlertTriangle className="size-7 text-red-600" />
        <p className="font-heading text-base font-semibold">{title}</p>
        <p className="max-w-md text-sm text-muted-foreground">{description}</p>
        {onRetry ? (
          <Button onClick={onRetry} type="button" variant="outline">
            {t("state.retry")}
          </Button>
        ) : null}
      </CardContent>
    </Card>
  )
}
