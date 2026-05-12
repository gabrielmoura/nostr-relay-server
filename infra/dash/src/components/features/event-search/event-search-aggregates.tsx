import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Bar, BarChart, Cell, Pie, PieChart, ResponsiveContainer, Tooltip as RechartsTooltip, XAxis, YAxis } from "recharts"
import { shortenId } from "@/lib/utils"

interface AggregatesData {
  kinds: Array<{ kind: number; count: number }>
  top_authors: Array<{ pubkey: string; count: number }>
  top_tags: Array<{ tag: string; count: number }>
}

interface EventSearchAggregatesProps {
  data?: AggregatesData
  isLoading: boolean
  isError: boolean
  onRetry: () => void
  onKindSelect?: (kind: number) => void
  onTagSelect?: (tag: string) => void
}

export function EventSearchAggregates({ data, isLoading, isError, onRetry, onKindSelect, onTagSelect }: EventSearchAggregatesProps) {
  const { t } = useTranslation()

  if (isLoading) return null
  if (isError) return null
  if (!data) return null

  return (
    <div className="grid gap-4 lg:grid-cols-3">
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-semibold">{t("eventSearch.frequentKinds")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-[200px] w-full">
            <ResponsiveContainer height="100%" width="100%">
              <BarChart data={data.kinds} layout="vertical" margin={{ top: 0, right: 0, left: 0, bottom: 0 }}>
                <XAxis type="number" hide />
                <YAxis dataKey="kind" type="category" width={40} tickLine={false} axisLine={false} tick={{ fontSize: 12, fill: "var(--color-muted-foreground)" }} />
                <RechartsTooltip
                  cursor={{ fill: "var(--color-muted)" }}
                  content={({ active, payload }: any) => {
                    if (active && payload && payload.length) {
                      return (
                        <div className="rounded border border-border bg-background p-2 text-xs shadow-sm">
                          <p className="font-medium text-foreground">Kind {payload[0].payload.kind}</p>
                          <p className="text-muted-foreground">{payload[0].value} eventos</p>
                        </div>
                      )
                    }
                    return null
                  }}
                />
                <Bar dataKey="count" fill="var(--color-primary)" onClick={(payload: any) => payload?.kind && onKindSelect?.(payload.kind)} radius={[0, 4, 4, 0]} style={{ cursor: onKindSelect ? "pointer" : "default" }} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>
      
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-semibold">{t("eventSearch.activeAuthors")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {data.top_authors.map((item) => (
            <div className="flex items-center justify-between text-sm" key={`author-${item.pubkey}`}>
              <Link className="underline decoration-dotted underline-offset-2 hover:text-primary transition-colors" params={{ pubkey: item.pubkey }} to="/users/$pubkey">
                {shortenId(item.pubkey, 10, 4)}
              </Link>
              <Badge variant="muted">{item.count}</Badge>
            </div>
          ))}
        </CardContent>
      </Card>
      
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-semibold">{t("eventSearch.commonTags")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-[200px] w-full relative">
            <ResponsiveContainer height="100%" width="100%">
              <PieChart margin={{ top: 0, right: 0, left: 0, bottom: 0 }}>
                <Pie
                  data={data.top_tags}
                  cx="50%"
                  cy="50%"
                  innerRadius={50}
                  outerRadius={80}
                  paddingAngle={2}
                  dataKey="count"
                  nameKey="tag"
                  onClick={(payload: any) => payload?.tag && onTagSelect?.(payload.tag)}
                >
                  {data.top_tags.map((entry, index) => {
                    // Generate colors based on primary color with varying opacities
                    const opacity = 1 - (index * 0.15)
                    return <Cell key={`cell-${index}`} fill={`oklch(var(--color-primary) / ${Math.max(0.2, opacity)})`} style={{ cursor: onTagSelect ? "pointer" : "default" }} />
                  })}
                </Pie>
                <RechartsTooltip
                  content={({ active, payload }: any) => {
                    if (active && payload && payload.length) {
                      return (
                        <div className="rounded border border-border bg-background p-2 text-xs shadow-sm">
                          <p className="font-medium text-foreground">#{payload[0].name}</p>
                          <p className="text-muted-foreground">{payload[0].value} eventos</p>
                        </div>
                      )
                    }
                    return null
                  }}
                />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
