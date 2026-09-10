import { useTranslation } from "react-i18next"
import { AlertCircle, RefreshCw, Terminal } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"

interface FeatureDisabledPanelProps {
  title: string
  description: string
  configKey: string
  howToEnable: string
}

export function FeatureDisabledPanel({ title, description, configKey, howToEnable }: FeatureDisabledPanelProps) {
  const { t } = useTranslation()

  return (
    <div className="flex min-h-[min(32rem,calc(100vh-14rem))] items-center justify-center py-6">
      <Card className="w-full max-w-2xl overflow-hidden border-warning/40">
        <CardHeader className="border-b border-warning/20 bg-warning/5 text-center sm:text-left">
          <div className="flex flex-col items-center gap-4 sm:flex-row sm:items-start">
            <div className="flex size-12 shrink-0 items-center justify-center rounded-full bg-warning/10 text-warning">
              <AlertCircle aria-hidden="true" className="size-6" />
            </div>
            <div className="space-y-1.5">
              <CardTitle className="text-xl">{title}</CardTitle>
              <CardDescription className="text-base">{description}</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-5 pt-5">
          <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/50 p-4">
            <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
              <Terminal aria-hidden="true" className="size-4 text-primary" />
              {t("common.howToEnable")}
            </h4>
            <p className="mt-2 text-sm text-muted-foreground">{t("common.configurationKey", "Configuration key")}</p>
            <code className="mt-2 block overflow-x-auto rounded-md border bg-background px-3 py-2 font-mono text-sm text-foreground">
              {configKey}
            </code>
            <pre className="mt-3 overflow-x-auto rounded-md border bg-background p-3 font-mono text-xs leading-5 text-foreground">
              {howToEnable}
            </pre>
          </div>
          <p className="text-sm text-muted-foreground">
            {t("common.restartRequired")}
          </p>
        </CardContent>
        <CardFooter className="justify-end border-t bg-muted/30 pt-5">
          <Button onClick={() => window.location.reload()} variant="outline">
            <RefreshCw aria-hidden="true" className="size-4" />
            {t("common.retry")}
          </Button>
        </CardFooter>
      </Card>
    </div>
  )
}
