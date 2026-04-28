import { useTranslation } from "react-i18next"
import { AlertCircle, Terminal } from "lucide-react"

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
    <div className="flex h-[80vh] items-center justify-center p-6">
      <Card className="max-w-2xl border-dashed">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex size-16 items-center justify-center rounded-full bg-warning/10 text-warning">
            <AlertCircle className="size-10" />
          </div>
          <CardTitle className="text-2xl">{title}</CardTitle>
          <CardDescription className="text-base">{description}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="rounded-md bg-muted p-4">
            <h4 className="mb-2 flex items-center gap-2 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
              <Terminal className="size-4" />
              {t("common.howToEnable")}
            </h4>
            <pre className="overflow-x-auto text-xs font-mono text-primary p-2 bg-background rounded border">
              {howToEnable}
            </pre>
          </div>
          <p className="text-sm text-muted-foreground text-center">
            {t("common.restartRequired", "A restart of the relay server is required after changing the configuration.")}
          </p>
        </CardContent>
        <CardFooter className="flex justify-center">
          <Button onClick={() => window.location.reload()} variant="outline">
            {t("common.retry")}
          </Button>
        </CardFooter>
      </Card>
    </div>
  )
}
