import { RefreshCw, Workflow } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { formatDateTime } from "@/lib/utils"
import type { BlossomWorkerRecord } from "@/types/admin"

type BlossomWorkersDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  items: BlossomWorkerRecord[]
  onRefresh: () => void
}

export function BlossomWorkersDialog({ open, onOpenChange, items, onRefresh }: BlossomWorkersDialogProps) {
  const { t } = useTranslation()

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-5xl max-h-[85vh] overflow-hidden p-0">
        <DialogHeader className="border-b border-border/60 px-6 py-4">
          <div className="flex items-center justify-between gap-3 pr-8">
            <DialogTitle className="flex items-center gap-2 text-sm font-mono uppercase tracking-[0.18em] text-primary">
              <Workflow className="size-4" />
              {t("blossom.workers.modalTitle", "Workers Blossom")}
            </DialogTitle>
            <Button onClick={onRefresh} size="sm" type="button" variant="outline">
              <RefreshCw className="size-4" />
              {t("jobs.actions.refresh", "Atualizar")}
            </Button>
          </div>
        </DialogHeader>

        <div className="overflow-auto px-6 py-5">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Job</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Hash</TableHead>
                <TableHead>Detalhe</TableHead>
                <TableHead>Atualizado</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((item) => (
                <TableRow key={item.job_id}>
                  <TableCell>
                    <div>
                      <p className="font-medium">{item.job_type}</p>
                      <p className="font-mono text-xs text-muted-foreground">{item.job_id}</p>
                    </div>
                  </TableCell>
                  <TableCell><Badge variant={item.status === "failed" ? "danger" : item.status === "running" ? "warning" : "default"}>{item.status}</Badge></TableCell>
                  <TableCell className="font-mono text-xs">{item.target_hash ?? "-"}</TableCell>
                  <TableCell>{item.detail}</TableCell>
                  <TableCell>{formatDateTime(item.updated_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </DialogContent>
    </Dialog>
  )
}
