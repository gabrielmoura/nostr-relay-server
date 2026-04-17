import { useMemo, useState } from "react"
import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { VirtualizedList } from "@/components/shared/virtualized-list"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useDisconnectConnectionMutation, useInfiniteConnections } from "@/hooks/use-admin-data"
import { shortenId } from "@/lib/utils"

export function ActiveConnectionsPage() {
  const { t } = useTranslation()
  const query = useInfiniteConnections("active")
  const disconnectMutation = useDisconnectConnectionMutation()
  const [mode, setMode] = useState<"all" | "authed" | "anonymous">("all")

  const pages = query.data?.pages ?? []
  const allConnections = pages.flatMap((page) => page.items)
  const total = pages[0]?.total ?? 0

  const rows = useMemo(() => {
    const data = allConnections
    if (mode === "authed") {
      return data.filter((item) => Boolean(item.authed))
    }
    if (mode === "anonymous") {
      return data.filter((item) => !item.authed)
    }
    return data
  }, [allConnections, mode])

  if (query.isLoading && allConnections.length === 0) {
    return <LoadingPanel label={t("activeConnections.loading")} />
  }

  if (query.isError) {
    return <ErrorPanel description={t("activeConnections.errorDescription")} onRetry={() => void query.refetch()} title={t("activeConnections.errorTitle")} />
  }

  return (
    <div className="space-y-6">
      <PageHeader description={t("activeConnections.description")} title={t("activeConnections.title")} />

      <div className="space-y-4">
        <Tabs onValueChange={(value) => setMode(value as typeof mode)} value={mode}>
          <TabsList>
            <TabsTrigger value="all">{t("activeConnections.all")}</TabsTrigger>
            <TabsTrigger value="authed">{t("activeConnections.authenticated")}</TabsTrigger>
            <TabsTrigger value="anonymous">{t("activeConnections.anonymous")}</TabsTrigger>
          </TabsList>
        </Tabs>

        {rows.length === 0 ? (
          <EmptyPanel description={t("activeConnections.emptyDescription")} title={t("activeConnections.emptyTitle")} />
        ) : (
          <VirtualizedList
            estimateSize={88}
            fetchMore={() => void query.fetchNextPage()}
            hasMore={query.hasNextPage}
            isFetchingMore={query.isFetchingNextPage}
            items={rows}
            renderItem={(connection) => (
              <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-card px-4 py-3">
                <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                  <div className="space-y-1">
                    <p className="font-mono text-xs text-foreground">{connection.ws_id}</p>
                    <p className="text-sm text-foreground">{connection.ip}</p>
                    {connection.authed ? (
                      <p className="text-xs text-muted-foreground">
                        {t("activeConnections.user")}: {" "}
                        <Link className="font-medium text-foreground underline decoration-dotted underline-offset-2" params={{ pubkey: connection.authed }} to="/users/$pubkey">
                          {shortenId(connection.authed, 14, 4)}
                        </Link>
                      </p>
                    ) : null}
                    <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                      <Badge variant={connection.authed ? "success" : "muted"}>{connection.authed ? t("activeConnections.authenticatedShort") : t("activeConnections.anonymousShort")}</Badge>
                      <Badge variant="muted">{t("activeConnections.subscriptionsCount", { count: connection.subscription_count })}</Badge>
                      {connection.user_agent ? <Badge variant="muted">{connection.user_agent}</Badge> : null}
                    </div>
                  </div>
                  <Button
                    disabled={disconnectMutation.isPending}
                    onClick={async () => {
                      await disconnectMutation.mutateAsync(connection.ws_id)
                      toast.success(t("activeConnections.connectionClosed"))
                    }}
                    size="sm"
                    variant="warning"
                  >
                    {t("activeConnections.closeConnection")}
                  </Button>
                </div>
              </div>
            )}
            total={mode === "all" ? total : rows.length}
          />
        )}
      </div>
    </div>
  )
}
