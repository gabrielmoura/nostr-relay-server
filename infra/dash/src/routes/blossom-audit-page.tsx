import { useState } from "react"
import { Link } from "@tanstack/react-router"
import { ArrowLeft } from "lucide-react"
import { useTranslation } from "react-i18next"

import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel } from "@/components/shared/state-panels"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useInfiniteBlossomAudit } from "@/hooks/use-admin-data"
import { formatDateTime, shortenId } from "@/lib/utils"

export function BlossomAuditPage() {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const auditQuery = useInfiniteBlossomAudit({ q: query })
  const items = auditQuery.data?.pages.flatMap((page) => page.items) ?? []

  return (
    <div className="space-y-6">
      <PageHeader
        actions={<Button asChild type="button" variant="outline"><Link to="/blossom"><ArrowLeft className="size-4" />{t("blossom.plans.back", "Voltar ao Blossom")}</Link></Button>}
        description={t("blossom.audit.routeDescription", "Analise a trilha imutável de mutações críticas sem competir com a navegação operacional principal.")}
        title={t("blossom.audit.routeTitle", "Auditoria Blossom")}
      />

      <Input onChange={(event) => setQuery(event.target.value)} placeholder={t("blossom.audit.search", "Buscar ação, alvo, actor ou request-id")} value={query} />

      {items.length === 0 ? <EmptyPanel description={t("blossom.audit.emptyDescription", "Nenhum evento de auditoria corresponde aos filtros atuais.")} title={t("blossom.audit.emptyTitle", "Sem eventos de auditoria")} /> : (
        <Card>
          <Table>
            <TableHeader><TableRow><TableHead>Ação</TableHead><TableHead>Alvo</TableHead><TableHead>Actor</TableHead><TableHead>Quando</TableHead></TableRow></TableHeader>
            <TableBody>
              {items.map((item) => (
                <TableRow key={item.id}>
                  <TableCell><div><p className="font-medium">{item.action}</p><p className="text-xs text-muted-foreground">{item.request_id ?? "-"}</p></div></TableCell>
                  <TableCell><p className="font-mono text-xs">{shortenId(item.target_id, 12, 6)}</p><p className="text-xs text-muted-foreground">{item.target_type}</p></TableCell>
                  <TableCell className="font-mono text-xs">{shortenId(item.actor_pubkey, 10, 4)}</TableCell>
                  <TableCell>{formatDateTime(item.created_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}
    </div>
  )
}
