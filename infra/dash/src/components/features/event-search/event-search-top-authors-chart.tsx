import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"
import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { EventAggregateAuthor } from "@/types/admin"
import { shortenId } from "@/lib/utils"

interface EventSearchTopAuthorsChartProps {
  items: EventAggregateAuthor[]
  onAuthorSelect?: (pubkey: string) => void
}

export function EventSearchTopAuthorsChart({ items, onAuthorSelect }: EventSearchTopAuthorsChartProps) {
  const { t } = useTranslation()

  return (
    <Card className="border-primary/5 bg-card/60 backdrop-blur-sm">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-semibold">{t("eventSearch.activeAuthors")}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-[240px] w-full">
          <ResponsiveContainer height="100%" width="100%">
            <BarChart data={items} layout="vertical" margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
              <XAxis hide type="number" />
              <YAxis dataKey="pubkey" type="category" width={96} tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }} tickFormatter={(value: string) => shortenId(value, 10, 4)} tickLine={false} axisLine={false} />
              <Tooltip content={({ active, payload }: any) => active && payload?.length ? (
                <div className="rounded border border-border bg-background p-2 text-xs shadow-sm">
                  <p className="font-medium text-foreground">{shortenId(payload[0].payload.pubkey, 14, 4)}</p>
                  <p className="text-muted-foreground">{payload[0].value} eventos</p>
                </div>
              ) : null} />
              <Bar dataKey="count" fill="var(--color-primary)" onClick={(payload: any) => payload?.payload?.pubkey && onAuthorSelect?.(payload.payload.pubkey)} radius={[0, 4, 4, 0]} style={{ cursor: onAuthorSelect ? "pointer" : "default" }} />
            </BarChart>
          </ResponsiveContainer>
        </div>
        <div className="mt-4 space-y-2">
          {items.slice(0, 5).map((item) => (
            <div className="flex items-center justify-between text-sm" key={item.pubkey}>
              <div className="flex min-w-0 items-center gap-2">
                <button className="min-w-0 text-left text-primary underline decoration-dotted underline-offset-2 hover:text-primary/80 transition-colors" onClick={() => onAuthorSelect?.(item.pubkey)} type="button">
                  <span className="block truncate">{item.display_name || shortenId(item.pubkey, 10, 4)}</span>
                </button>
                <Link className="text-xs text-muted-foreground underline underline-offset-2 hover:text-primary" params={{ pubkey: item.pubkey }} to="/users/$pubkey">
                  {t("eventSearch.openAuthor", "abrir")}
                </Link>
              </div>
              <span className="text-muted-foreground">{item.count}</span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
