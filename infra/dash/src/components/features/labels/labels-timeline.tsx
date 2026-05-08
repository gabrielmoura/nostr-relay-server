import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { BanUserDialog } from "@/components/features/ban-user-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { labelBadgeVariant } from "@/lib/labels"
import type { AdminLabelEvent } from "@/types/admin"
import { formatDateTime, shortenId } from "@/lib/utils"

export function LabelsTimeline({ items, hasMore, isFetchingMore, onLoadMore }: { items: AdminLabelEvent[]; hasMore?: boolean; isFetchingMore?: boolean; onLoadMore: () => void }) {
  const { t } = useTranslation()

  return (
    <div className="space-y-3">
      {items.map((item) => {
        const isPubkey = item.target.type === "pubkey"
        const targetDisplay = item.target.type === "pubkey"
          ? shortenId(item.target.value, 12, 6)
          : item.target.type === "event"
            ? shortenId(item.target.value, 12, 6)
            : item.target.value

        return (
          <Card key={item.id}>
            <CardContent className="p-4">
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-3">
                  <div className="flex flex-wrap gap-2">
                    {item.labels.map((label) => (
                      <Badge key={`${item.id}-${label}`} variant={labelBadgeVariant(label)}>{label}</Badge>
                    ))}
                    <Badge variant="muted">{item.namespace}</Badge>
                    <Badge variant="default">{item.target.type}</Badge>
                  </div>

                  <div className="space-y-1 text-sm">
                    <p className="font-medium text-foreground">{targetDisplay}</p>
                    <p className="text-xs text-muted-foreground">
                      {t("labels.timeline.author", "Autor")}: {shortenId(item.author_npub || item.pubkey, 16, 6)}
                    </p>
                    <p className="text-xs text-muted-foreground">{formatDateTime(item.created_at)}</p>
                    {item.target.relay_hint ? <p className="break-all text-xs text-muted-foreground">{item.target.relay_hint}</p> : null}
                    {item.content ? <p className="text-sm text-foreground">{item.content}</p> : null}
                  </div>
                </div>

                <div className="flex flex-wrap gap-2">
                  {item.target.type === "event" ? (
                    <Button asChild size="sm" variant="outline">
                      <Link params={{ eventId: item.target.value }} to="/events/$eventId">{t("labels.actions.viewEvent", "Ver evento")}</Link>
                    </Button>
                  ) : null}
                  {isPubkey ? (
                    <BanUserDialog
                      defaultPubkey={item.target.value}
                      defaultReason={`Labeled: ${item.labels.join(", ")}`}
                      triggerLabel={t("labels.actions.banPubkey", "Banir pubkey")}
                      triggerVariant="warning"
                    />
                  ) : null}
                </div>
              </div>
            </CardContent>
          </Card>
        )
      })}

      {hasMore ? (
        <div className="flex justify-center pt-2">
          <Button onClick={onLoadMore} type="button" variant="outline">
            {isFetchingMore ? t("labels.timeline.loadingMore", "Carregando...") : t("labels.timeline.loadMore", "Carregar mais")}
          </Button>
        </div>
      ) : null}
    </div>
  )
}
