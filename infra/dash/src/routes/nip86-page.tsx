import { Suspense } from "react"
import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"
import { ArrowRight, Shield, ShieldAlert, WifiOff } from "lucide-react"
import { ErrorBoundary } from "react-error-boundary"

import { AllowedPubkeysPanel } from "@/components/features/nip86/allowed-pubkeys-panel"
import { BannedEventsPanel } from "@/components/features/nip86/banned-events-panel"
import { BlockedIPsPanel } from "@/components/features/nip86/blocked-ips-panel"
import { RelayMetadataForm } from "@/components/features/nip86/relay-metadata-form"
import { PageHeader } from "@/components/shared/page-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { FeatureDisabledPanel } from "@/components/shared/feature-disabled-panel"
import { isFeatureDisabledError } from "@/hooks/use-admin-data"

const commandCards = [
  { key: "allowed", icon: Shield, variant: "success" as const },
  { key: "blockedIps", icon: WifiOff, variant: "danger" as const },
  { key: "bannedEvents", icon: ShieldAlert, variant: "warning" as const },
]

export function NIP86Page() {
  return (
    <ErrorBoundary
      fallbackRender={({ error }: { error: any }) => {
        if (isFeatureDisabledError(error)) {
          return (
            <FeatureDisabledPanel
              configKey="nip86.enabled"
              description="NIP-86 (Relay Management API) permite gerenciar listas de permissões, bloqueios e metadados do relay via comandos administrativos."
              howToEnable="nip86:
  enabled: true"
              title="NIP-86 Desabilitado"
            />
          )
        }
        throw error
      }}
    >
      <Suspense fallback={<NIP86Skeleton />}>
        <NIP86Content />
      </Suspense>
    </ErrorBoundary>
  )
}

function NIP86Content() {
  const { t } = useTranslation()

  return (
    <div className="space-y-6">
      <PageHeader
        actions={
          <Button asChild variant="outline">
            <Link to="/events/reported">
              {t("nip86.header.reviewReports")}
              <ArrowRight className="size-4" />
            </Link>
          </Button>
        }
        className="rounded-[var(--radius)] border border-primary/15 bg-[linear-gradient(135deg,rgba(14,165,233,0.08),rgba(34,197,94,0.08))] p-5 panel-shadow"
        description={t("nip86.header.description")}
        title={t("nip86.header.title")}
      />

      <section className="grid gap-4 xl:grid-cols-3">
        {commandCards.map((card) => {
          const Icon = card.icon
          return (
            <Card className="border-border/70 bg-card/95" key={card.key}>
              <CardHeader className="space-y-3">
                <div className="flex items-center justify-between">
                  <Badge variant={card.variant}>{t(`nip86.cards.${card.key}.badge`)}</Badge>
                  <div className="rounded-full bg-secondary p-2 text-primary">
                    <Icon className="size-4" />
                  </div>
                </div>
                <div>
                  <CardTitle className="text-lg">{t(`nip86.cards.${card.key}.title`)}</CardTitle>
                  <CardDescription>{t(`nip86.cards.${card.key}.description`)}</CardDescription>
                </div>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">{t(`nip86.cards.${card.key}.body`)}</p>
              </CardContent>
            </Card>
          )
        })}
      </section>

      <section className="grid gap-4 xl:grid-cols-[1.1fr_1fr]">
        <div className="space-y-4">
          <AllowedPubkeysPanel />
          <BlockedIPsPanel />
        </div>
        <div className="space-y-4">
          <RelayMetadataForm />
          <BannedEventsPanel />
        </div>
      </section>
    </div>
  )
}

function NIP86Skeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-40 w-full" />
      <div className="grid gap-4 xl:grid-cols-3">
        <Skeleton className="h-48 w-full" />
        <Skeleton className="h-48 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
      <div className="grid gap-4 xl:grid-cols-[1.1fr_1fr]">
        <Skeleton className="h-[600px] w-full" />
        <Skeleton className="h-[600px] w-full" />
      </div>
    </div>
  )
}
