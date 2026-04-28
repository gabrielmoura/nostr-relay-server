import { Suspense } from "react"
import { useTranslation } from "react-i18next"
import { Users } from "lucide-react"
import { ErrorBoundary } from "react-error-boundary"

import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel } from "@/components/shared/state-panels"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useInfiniteGroups, isFeatureDisabledError } from "@/hooks/use-admin-data"
import { FeatureDisabledPanel } from "@/components/shared/feature-disabled-panel"
import { Skeleton } from "@/components/ui/skeleton"

export function GroupsPage() {
  return (
    <ErrorBoundary
      fallbackRender={({ error, resetErrorBoundary }: { error: any; resetErrorBoundary: any }) => {
        if (isFeatureDisabledError(error)) {
          return (
            <FeatureDisabledPanel
              configKey="nip29.enabled"
              description="NIP-29 (Relay-based Groups) permite criar e gerenciar grupos moderados e canais de chat diretamente no relay."
              howToEnable="nip29:
  enabled: true"
              title="Módulo de Grupos Desabilitado"
            />
          )
        }
        return (
          <div className="p-6 text-center">
            <h2 className="text-lg font-bold text-destructive">Algo deu errado</h2>
            <p className="text-muted-foreground">{error.message}</p>
            <button onClick={resetErrorBoundary} className="mt-4 text-primary underline">Tentar novamente</button>
          </div>
        )
      }}
    >
      <Suspense fallback={<GroupsSkeleton />}>
        <GroupsContent />
      </Suspense>
    </ErrorBoundary>
  )
}

function GroupsContent() {
  const { t } = useTranslation()
  const listQuery = useInfiniteGroups()

  const items = listQuery.data?.pages.flatMap((page) => page.items) ?? []
  const hasItems = items.length > 0

  if (!hasItems && !listQuery.isLoading) {
    return (
      <div className="space-y-6">
        <PageHeader
          description={t("groups.description")}
          title={t("groups.title")}
        />
        <EmptyPanel
          description={t("groups.empty")}
          title={t("groups.empty")}
        />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        description={t("groups.description")}
        title={t("groups.title")}
      />

      <Card>
        <CardContent className="p-4">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("groups.table.id")}</TableHead>
                <TableHead>{t("groups.table.name")}</TableHead>
                <TableHead>{t("groups.table.members")}</TableHead>
                <TableHead>{t("groups.table.privacy")}</TableHead>
                <TableHead>{t("groups.table.status")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((item) => (
                <TableRow key={item.group_id}>
                  <TableCell className="font-mono text-xs">{item.group_id}</TableCell>
                  <TableCell className="font-medium">{item.name}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Users className="size-3" />
                      {item.member_count}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={item.private ? "muted" : "default"}>
                      {item.private ? t("groups.table.private") : t("groups.table.public")}
                    </Badge>
                    {item.hidden && (
                      <Badge className="ml-1" variant="danger">
                        {t("groups.table.hidden")}
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant={item.closed ? "warning" : "success"}>
                      {item.closed ? t("groups.table.closed") : t("groups.table.open")}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}

function GroupsSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-24 w-full" />
      <Skeleton className="h-96 w-full" />
    </div>
  )
}
