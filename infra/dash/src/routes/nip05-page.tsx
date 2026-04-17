import { useState } from "react"
import { useTranslation } from "react-i18next"

import { NIP05CreateDialog } from "@/components/features/nip05-create-dialog"
import { NIP05EditDialog } from "@/components/features/nip05-edit-dialog"
import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useInfiniteNIP05 } from "@/hooks/use-admin-data"

export function NIP05Page() {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")

  const listQuery = useInfiniteNIP05(query)

  const items = listQuery.data?.pages.flatMap((page) => page.items) ?? []
  const hasItems = items.length > 0

  return (
    <div className="space-y-6">
      <PageHeader
        actions={<NIP05CreateDialog />}
        description={t("nip05.description")}
        title={t("nip05.title")}
      />

      <Card>
        <CardContent className="space-y-4 p-4">
          <Input onChange={(event) => setQuery(event.target.value)} placeholder={t("nip05.filterPlaceholder")} value={query} />

          {listQuery.isLoading && !hasItems ? <LoadingPanel label={t("nip05.loading")} /> : null}
          {listQuery.isError ? <ErrorPanel description={t("nip05.loadErrorDescription")} onRetry={() => void listQuery.refetch()} title={t("nip05.loadErrorTitle")} /> : null}
          {!listQuery.isLoading && !listQuery.isError && !hasItems ? <EmptyPanel description={t("nip05.emptyDescription")} title={t("nip05.emptyTitle")} /> : null}

          {hasItems ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("common.name")}</TableHead>
                  <TableHead>{t("common.pubkey")}</TableHead>
                  <TableHead>{t("common.user")}</TableHead>
                  <TableHead>{t("common.relayHints")}</TableHead>
                  <TableHead className="text-right">{t("common.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={`${item.name}:${item.pubkey}`}>
                    <TableCell className="font-medium">{item.name}</TableCell>
                    <TableCell className="break-all font-mono text-xs">{item.pubkey}</TableCell>
                    <TableCell>{item.display_name ?? "-"}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{item.relay_hints?.length ? item.relay_hints.join(", ") : t("nip05.noRelayHints")}</TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <NIP05EditDialog item={item} />
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : null}
        </CardContent>
      </Card>
    </div>
  )
}
