import type { ReactNode } from "react"
import { BarChart3, Hash, Waypoints } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Bar, BarChart, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { AdminLabelsSummary } from "@/types/admin"

interface LabelsAnalyticsModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  summary?: AdminLabelsSummary
}

export function LabelsAnalyticsModal({ open, onOpenChange, summary }: LabelsAnalyticsModalProps) {
  const { t } = useTranslation()

  const namespaceData = summary?.namespaces ?? []
  const labelsData = summary?.labels ?? []
  const targetTypes = summary?.target_types ?? []

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-6xl max-h-[88vh] overflow-hidden border-primary/15 bg-background/95 backdrop-blur-xl p-0">
        <DialogHeader className="border-b border-border/60 px-6 py-4">
          <DialogTitle className="flex items-center gap-2 text-sm font-mono uppercase tracking-[0.18em] text-primary">
            <BarChart3 className="size-4" />
            {t("labels.analytics.title", "Análises dos labels")}
          </DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-auto px-6 py-5 space-y-6">
          <div className="grid gap-3 md:grid-cols-3">
            <KpiCard helper={t("labels.stats.eventsHelper")} label={t("labels.stats.events")} value={summary?.total_events ?? 0} />
            <KpiCard helper={t("labels.stats.targetsHelper")} label={t("labels.stats.targets")} value={summary?.total_targets ?? 0} />
            <KpiCard helper={t("labels.stats.topLabelHelper")} label={t("labels.stats.topLabel")} value={labelsData[0]?.label ?? "-"} />
          </div>

          <div className="grid gap-4 xl:grid-cols-3">
            <ChartCard icon={Hash} title={t("labels.filters.namespace", "Namespace")}>
              <ResponsiveContainer height="100%" width="100%">
                <BarChart data={namespaceData} layout="vertical" margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                  <XAxis hide type="number" />
                  <YAxis dataKey="namespace" type="category" width={96} tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }} tickLine={false} axisLine={false} />
                  <Tooltip />
                  <Bar dataKey="count" fill="var(--color-primary)" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard icon={BarChart3} title={t("labels.filters.label", "Label")}>
              <ResponsiveContainer height="100%" width="100%">
                <PieChart>
                  <Pie data={labelsData.slice(0, 6)} cx="50%" cy="50%" dataKey="count" innerRadius={45} outerRadius={78} nameKey="label" paddingAngle={2} />
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard icon={Waypoints} title={t("labels.filters.targetType", "Tipo de alvo")}>
              <ResponsiveContainer height="100%" width="100%">
                <BarChart data={targetTypes} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                  <XAxis dataKey="target_type" tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }} tickLine={false} axisLine={false} />
                  <YAxis hide />
                  <Tooltip />
                  <Bar dataKey="count" fill="#22c55e" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </ChartCard>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function KpiCard({ label, value, helper }: { label: string; value: number | string; helper?: string }) {
  return (
    <Card className="overflow-hidden border-primary/5 bg-card/60 backdrop-blur-sm">
      <CardContent className="p-4">
        <p className="text-xs font-bold uppercase tracking-wider text-muted-foreground">{label}</p>
        <p className="mt-2 text-xl font-mono font-bold tracking-tight text-primary leading-none">{typeof value === "number" ? value.toLocaleString() : value}</p>
        {helper ? <p className="mt-1 text-xs text-muted-foreground">{helper}</p> : null}
      </CardContent>
    </Card>
  )
}

function ChartCard({ title, icon: Icon, children }: { title: string; icon: typeof BarChart3; children: ReactNode }) {
  return (
    <Card className="border-primary/5 bg-card/60 backdrop-blur-sm">
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-sm font-semibold">
          <Icon className="size-4 text-primary" />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-[240px] w-full">{children}</div>
      </CardContent>
    </Card>
  )
}
