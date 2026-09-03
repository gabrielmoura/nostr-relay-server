import { useTranslation } from "react-i18next"

import { PrivacyMetrics, uptimeMsToHuman } from "@/components/privacy/privacy-metrics"
import { EmptyPanel } from "@/components/shared/state-panels"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import type { PrivacyNetwork } from "@/types/admin"

const statusVariant = {
  operational: "success",
  starting: "warning",
  error: "danger",
  disabled: "muted",
} as const

interface PrivacyNetworksPanelProps {
  networks: PrivacyNetwork[]
}

export function PrivacyNetworksPanel({ networks }: PrivacyNetworksPanelProps) {
  const { t } = useTranslation()

  if (networks.length === 0) {
    return <EmptyPanel title={t("privacy.emptyTitle")} description={t("privacy.emptyDescription")} />
  }

  const firstId = networks[0]?.id ?? "tor"

  return (
    <Tabs defaultValue={firstId} className="w-full">
      <TabsList className="mb-4 flex w-full flex-wrap justify-start">
        {networks.map((network) => (
          <TabsTrigger key={network.id} className="flex items-center gap-2" value={network.id}>
            <span>{network.name}</span>
            <Badge variant={statusVariant[network.status]}>{t(`privacy.${network.status}`)}</Badge>
          </TabsTrigger>
        ))}
      </TabsList>

      {networks.map((network) => (
        <TabsContent key={network.id} value={network.id}>
          <NetworkCard network={network} />
        </TabsContent>
      ))}
    </Tabs>
  )
}

function NetworkCard({ network }: { network: PrivacyNetwork }) {
  const { t } = useTranslation()

  return (
    <div className="grid gap-4 xl:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between gap-2">
            <span>{network.name}</span>
            <Badge variant={statusVariant[network.status]}>{t(`privacy.${network.status}`)}</Badge>
          </CardTitle>
          <CardDescription>{network.id}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={network.enabled ? "success" : "muted"}>{network.enabled ? t("privacy.enabled") : t("privacy.disabled")}</Badge>
            <Badge variant={network.started ? "success" : "muted"}>{network.started ? t("privacy.started") : t("privacy.stopped")}</Badge>
            <Badge variant="default">{t("privacy.mode")}: {network.mode}</Badge>
            <Badge variant="default">{t("privacy.uptime")}: {uptimeMsToHuman(network.uptime_ms)}</Badge>
          </div>

          <div>
            <p className="mb-1 text-xs uppercase tracking-wider text-muted-foreground">{t("privacy.addresses")}</p>
            {network.addresses.length > 0 ? (
              <ul className="space-y-1">
                {network.addresses.map((address) => (
                  <li key={address} className="rounded border border-border px-3 py-1.5 font-mono text-xs">
                    {address}
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-muted-foreground">{t("privacy.noAddresses")}</p>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("privacy.metricsTitle")}</CardTitle>
          <CardDescription>{t("privacy.metricsDescription")}</CardDescription>
        </CardHeader>
        <CardContent>
          <PrivacyMetrics metrics={network.metrics} />
          {network.error ? (
            <p className="mt-3 rounded border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">{network.error}</p>
          ) : null}
        </CardContent>
      </Card>
    </div>
  )
}
