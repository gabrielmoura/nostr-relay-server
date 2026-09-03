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

  if (query.isLoading) {
    return <LoadingPanel label={t("privacy.loading")} />
  }

  if (query.isError || !query.data) {
    return <ErrorPanel title={t("privacy.errorTitle")} description={t("privacy.errorDescription")} onRetry={() => void query.refetch()} />
  }

  if (!query.data.enabled) {
    return (
      <FeatureDisabledPanel
        title={t("privacy.disabledTitle")}
        description={t("privacy.disabledDescription")}
        configKey="privacy"
        howToEnable={t("privacy.howToEnable")}
      />
    )
  }

  const privacy = query.data

  return (
    <div className="space-y-6">
      <PageHeader title={t("privacy.title")} description={t("privacy.description")} />

      <section className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardHeader>
            <CardDescription>{t("privacy.enabled")}</CardDescription>
            <CardTitle>{privacy.enabled ? t("common.yes") : t("common.no")}</CardTitle>
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

      <PrivacyNetworksPanel networks={privacy.networks} />
    </div>
  )
}
