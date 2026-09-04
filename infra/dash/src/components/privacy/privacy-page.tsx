import { useTranslation } from "react-i18next"

import { PrivacyNetworksPanel } from "@/components/privacy/privacy-networks-panel"
import { FeatureDisabledPanel } from "@/components/shared/feature-disabled-panel"
import { PageHeader } from "@/components/shared/page-header"
import { ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { usePrivacyStatus } from "@/hooks/use-admin-data"

export function PrivacyPage() {
  const { t } = useTranslation()
  const query = usePrivacyStatus()

  return (
    <div className="space-y-6">
      <PageHeader title={t("privacy.title")} description={t("privacy.description")} />

      {query.isLoading ? (
        <LoadingPanel label={t("privacy.loading")} />
      ) : query.isError || !query.data ? (
        <ErrorPanel
          title={t("privacy.errorTitle")}
          description={t("privacy.errorDescription")}
          onRetry={() => void query.refetch().catch(() => undefined)}
        />
      ) : (
        <PrivacyStatusContent privacy={query.data} />
      )}
    </div>
  )
}

function PrivacyStatusContent({ privacy }: { privacy: NonNullable<ReturnType<typeof usePrivacyStatus>["data"]> }) {
  const { t } = useTranslation()
  const networks = privacy.networks ?? []
  const activeNetworks = networks.filter((network) => network.started).length
  const failedNetworks = networks.filter((network) => network.status === "error").length

  return (
    <>
      <section className="grid gap-4 sm:grid-cols-3 xl:grid-cols-5">
        <Card>
          <CardHeader>
            <CardDescription>{t("privacy.enabled")}</CardDescription>
            <CardTitle>{privacy.enabled ? t("common.yes") : t("common.no")}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>{t("privacy.networksConfigured")}</CardDescription>
            <CardTitle>{networks.length}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>{t("privacy.networksActive")}</CardDescription>
            <CardTitle>{activeNetworks}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>{t("privacy.networksError")}</CardDescription>
            <CardTitle>{failedNetworks}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>{t("privacy.persistence")}</CardDescription>
            <CardTitle>{privacy.persistence ? t("common.yes") : t("common.no")}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>{t("privacy.stateDir")}</CardDescription>
            <CardTitle className="font-mono text-sm">{privacy.state_dir ?? "—"}</CardTitle>
          </CardHeader>
        </Card>
      </section>

      {!privacy.enabled ? (
        <FeatureDisabledPanel
          title={t("privacy.disabledTitle")}
          description={t("privacy.disabledDescription")}
          configKey="privacy"
          howToEnable={t("privacy.howToEnable")}
        />
      ) : (
        <PrivacyNetworksPanel networks={networks} />
      )}
    </>
  )
}
