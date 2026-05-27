import { Copy, Download, Sparkles, Trash2 } from "lucide-react"
import { toast } from "sonner"
import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Sheet, SheetContent } from "@/components/ui/sheet"
import { useBlossomObjectDetail } from "@/hooks/use-admin-data"
import { blossomExifVariant, blossomReviewVariant } from "@/lib/blossom"
import { formatBytes, formatCount, formatDateTime, shortenId } from "@/lib/utils"

type BlossomObjectSheetProps = {
  hash: string
  open: boolean
  onOpenChange: (open: boolean) => void
  onApprove: (hash: string) => void
  onDelete: (hash: string) => void
  onRequeue: (hash: string) => void
}

export function BlossomObjectSheet({ hash, open, onOpenChange, onApprove, onDelete, onRequeue }: BlossomObjectSheetProps) {
  const { t } = useTranslation()
  const detailQuery = useBlossomObjectDetail(hash, open)
  const detail = detailQuery.data

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent className="w-[min(32rem,96vw)] overflow-y-auto" side="right" title={t("blossom.sheet.title", "Detalhes do arquivo Blossom")}>
        {!detail ? null : (
          <div className="space-y-5 pr-6">
            <div className="space-y-3">
              <div className="overflow-hidden rounded-[var(--radius)] border border-border bg-muted/30">
                {detail.thumbnail_url ? <img alt={detail.hash} className="aspect-video w-full object-cover" src={detail.thumbnail_url} /> : null}
              </div>
              <div className="flex flex-wrap gap-2">
                <Badge variant={blossomReviewVariant(detail.review_state)}>{detail.review_state}</Badge>
                <Badge variant={blossomExifVariant(detail.exif_status)}>{detail.exif_status}</Badge>
                <Badge variant="muted">{detail.mime_type}</Badge>
              </div>
              <div>
                <p className="font-heading text-lg font-semibold break-all">{shortenId(detail.hash, 16, 8)}</p>
                <p className="text-sm text-muted-foreground">{t("blossom.sheet.uploader", "Uploader")}: <span className="font-mono">{shortenId(detail.uploader_pubkey, 12, 6)}</span></p>
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <Metric label={t("blossom.table.size", "Tamanho")} value={formatBytes(detail.size)} />
              <Metric label={t("blossom.sheet.downloads", "Downloads")} value={formatCount(detail.download_count)} />
              <Metric label={t("blossom.sheet.dimensions", "Dimensões")} value={detail.width && detail.height ? `${detail.width}x${detail.height}` : "-"} />
              <Metric label={t("blossom.sheet.duration", "Duração")} value={detail.duration_ms ? `${Math.round(detail.duration_ms / 1000)}s` : "-"} />
              <Metric label={t("blossom.sheet.bitrate", "Bitrate")} value={detail.bitrate_kbps ? `${detail.bitrate_kbps} kbps` : "-"} />
              <Metric label={t("blossom.sheet.lastDownload", "Último download")} value={detail.last_downloaded_at ? formatDateTime(detail.last_downloaded_at) : "-"} />
              <Metric label={t("blossom.sheet.reports", "Reports abertos")} value={String(detail.report_count ?? 0)} />
            </div>

            <div className="space-y-2">
              <p className="text-sm font-semibold text-foreground">{t("blossom.sheet.quickActions", "Ações rápidas")}</p>
              <div className="flex flex-wrap gap-2">
                <Button
                  onClick={() => {
                    void navigator.clipboard.writeText(detail.direct_url)
                    toast.success(t("blossom.sheet.copied", "URL direta copiada"))
                  }}
                  type="button"
                  variant="outline"
                >
                  <Copy className="size-4" />
                  {t("blossom.sheet.copyUrl", "Copiar URL")}
                </Button>
                <Button
                  onClick={() => {
                    if (!detail.blossom_id) {
                      return
                    }
                    void navigator.clipboard.writeText(detail.blossom_id)
                    toast.success(t("blossom.sheet.copiedId", "Blossom ID copiado"))
                  }}
                  type="button"
                  variant="outline"
                >
                  <Copy className="size-4" />
                  {t("blossom.sheet.copyBlossomId", "Copiar Blossom ID")}
                </Button>
                <Button asChild type="button" variant="outline">
                  <a href={detail.direct_url} rel="noreferrer" target="_blank">
                    <Download className="size-4" />
                    {t("blossom.sheet.download", "Baixar")}
                  </a>
                </Button>
                <Button onClick={() => onRequeue(detail.hash)} type="button" variant="warning">
                  <Sparkles className="size-4" />
                  {t("blossom.sheet.reprocess", "Forçar otimização")}
                </Button>
                <Button onClick={() => onDelete(detail.hash)} type="button" variant="destructive">
                  <Trash2 className="size-4" />
                  {t("blossom.sheet.delete", "Hard-delete")}
                </Button>
              </div>
            </div>

            {detail.flag_reason ? (
              <div className="rounded-[var(--radius)] border border-orange-200 bg-orange-50 p-4 text-sm text-orange-900">
                <p className="font-semibold">{t("blossom.sheet.flagReason", "Motivo de revisão")}</p>
                <p className="mt-1">{detail.flag_reason}</p>
              </div>
            ) : null}

            <div className="space-y-2">
              <p className="text-sm font-semibold text-foreground">NIP-94</p>
              <div className="rounded-[var(--radius)] border border-border bg-muted/20 p-3 text-sm">
                {Object.entries(detail.nip94_tags ?? {}).map(([key, value]) => (
                  <div className="grid grid-cols-[5rem_1fr] gap-2 py-1" key={key}>
                    <span className="font-mono text-muted-foreground">{key}</span>
                    <span className="break-all text-foreground">{value}</span>
                  </div>
                ))}
              </div>
            </div>

            {(detail.mirrors ?? []).length > 0 ? (
              <div className="space-y-2">
                <p className="text-sm font-semibold text-foreground">{t("blossom.sheet.mirrors", "Mirrors")}</p>
                <div className="space-y-2 rounded-[var(--radius)] border border-border bg-muted/20 p-3 text-sm">
                  {(detail.mirrors ?? []).map((mirror) => (
                    <p className="break-all" key={mirror}>{mirror}</p>
                  ))}
                </div>
              </div>
            ) : null}

            <div className="flex gap-2">
              <Button onClick={() => onApprove(detail.hash)} type="button">{t("blossom.bulk.approve", "Aprovar")}</Button>
              <Button onClick={() => onOpenChange(false)} type="button" variant="outline">{t("common.close", "Fechar")}</Button>
            </div>
          </div>
        )}
      </SheetContent>
    </Sheet>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/20 p-3">
      <p className="text-xs uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className="mt-1 font-medium text-foreground">{value}</p>
    </div>
  )
}
