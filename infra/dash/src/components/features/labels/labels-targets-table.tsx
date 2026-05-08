import { useTranslation } from "react-i18next"

import { BanUserDialog } from "@/components/features/ban-user-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { labelBadgeVariant, type GroupedLabelTarget } from "@/lib/labels"
import { formatDateTime, shortenId } from "@/lib/utils"

export function LabelsTargetsTable({ items, hasMore, isFetchingMore, onLoadMore }: { items: GroupedLabelTarget[]; hasMore?: boolean; isFetchingMore?: boolean; onLoadMore?: () => void }) {
  const { t } = useTranslation()

  return (
    <div className="space-y-3">
      {items.map((item) => (
        <Card key={item.key}>
          <CardContent className="p-4">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
              <div className="space-y-3">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant="default">{item.type}</Badge>
                  <Badge variant="muted">{t("labels.targets.eventsCount", { count: item.eventsCount, defaultValue: `${item.eventsCount} evento(s)` })}</Badge>
                </div>

                <div>
                  <p className="break-all font-medium text-foreground">
                    {item.type === "pubkey" || item.type === "event" ? shortenId(item.value, 14, 6) : item.value}
                  </p>
                  {item.relayHint ? <p className="break-all text-xs text-muted-foreground">{item.relayHint}</p> : null}
                  <p className="text-xs text-muted-foreground">{formatDateTime(item.lastCreatedAt)}</p>
                </div>

                <div className="flex flex-wrap gap-2">
                  {item.labels.map((label) => (
                    <Badge key={`${item.key}-${label}`} variant={labelBadgeVariant(label)}>{label}</Badge>
                  ))}
                </div>
              </div>

              <div className="flex min-w-52 flex-col gap-3">
                <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-muted/30 p-3 text-xs text-muted-foreground">
                  <p className="font-medium text-foreground">{t("labels.targets.namespaces", "Namespaces")}</p>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {Object.entries(item.namespaceCounts).map(([namespace, count]) => (
                      <Badge key={`${item.key}-${namespace}`} variant="muted">{namespace}: {count}</Badge>
                    ))}
                  </div>
                </div>

                {item.type === "pubkey" ? (
                  <BanUserDialog
                    defaultPubkey={item.value}
                    defaultReason={`Labeled: ${item.labels.join(", ")}`}
                    triggerLabel={t("labels.actions.banPubkey", "Banir pubkey")}
                    triggerVariant="warning"
                  />
                ) : null}
              </div>
            </div>
          </CardContent>
        </Card>
      ))}

      {hasMore && onLoadMore ? (
        <div className="flex justify-center pt-2">
          <Button onClick={onLoadMore} type="button" variant="outline">
            {isFetchingMore ? t("labels.timeline.loadingMore") : t("labels.timeline.loadMore")}
          </Button>
        </div>
      ) : null}
    </div>
  )
}
