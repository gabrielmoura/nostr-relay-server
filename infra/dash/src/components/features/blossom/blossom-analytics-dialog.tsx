import type { ReactNode } from "react"
import { BarChart3, HardDrive, ShieldAlert, Workflow } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Bar, BarChart, Cell, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { BlossomAnalytics } from "@/types/admin"

type BlossomAnalyticsDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  analytics?: BlossomAnalytics
}

const chartColors = ["#0ea5e9", "#22c55e", "#f97316", "#8b5cf6", "#ef4444", "#14b8a6"]

export function BlossomAnalyticsDialog({ open, onOpenChange, analytics }: BlossomAnalyticsDialogProps) {
  const { t } = useTranslation()

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-6xl max-h-[88vh] overflow-hidden border-primary/15 bg-background/95 p-0 backdrop-blur-xl">
        <DialogHeader className="border-b border-border/60 px-6 py-4">
          <DialogTitle className="flex items-center gap-2 text-sm font-mono uppercase tracking-[0.18em] text-primary">
            <BarChart3 className="size-4" />
            {t("blossom.analytics.title", "Análises do Blossom")}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-6 overflow-auto px-6 py-5">
          <div className="grid gap-3 md:grid-cols-3">
            <KpiCard helper={t("blossom.analytics.reportsHelper", "Total de sinais de moderação recebidos.")} label={t("blossom.analytics.reports", "Reports")} value={analytics?.reports.total ?? 0} />
            <KpiCard helper={t("blossom.analytics.openHelper", "Pendências ainda exigindo triagem.")} label={t("blossom.analytics.open", "Abertos")} value={analytics?.reports.open ?? 0} />
            <KpiCard helper={t("blossom.analytics.resolvedHelper", "Reports já encerrados por operação.")} label={t("blossom.analytics.resolved", "Resolvidos")} value={analytics?.reports.resolved ?? 0} />
          </div>

          <div className="grid gap-4 xl:grid-cols-3">
            <ChartCard icon={HardDrive} title={t("blossom.analytics.mime", "Distribuição por MIME")}>
              <ResponsiveContainer height="100%" width="100%">
                <BarChart data={analytics?.objects.by_mime ?? []} layout="vertical" margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                  <XAxis hide type="number" />
                  <YAxis dataKey="name" type="category" width={100} tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }} tickLine={false} axisLine={false} />
                  <Tooltip />
                  <Bar dataKey="count" fill="var(--color-primary)" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard icon={ShieldAlert} title={t("blossom.analytics.review", "Estados de revisão")}>
              <ResponsiveContainer height="100%" width="100%">
                <PieChart>
                  <Pie data={analytics?.objects.by_review_state ?? []} cx="50%" cy="50%" dataKey="count" innerRadius={42} outerRadius={78} nameKey="name" paddingAngle={2}>
                    {(analytics?.objects.by_review_state ?? []).map((item, index) => <Cell fill={chartColors[index % chartColors.length]} key={item.name} />)}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard icon={Workflow} title={t("blossom.analytics.workers", "Workers por status")}>
              <ResponsiveContainer height="100%" width="100%">
                <BarChart data={analytics?.workers.by_status ?? []} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                  <XAxis dataKey="name" tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }} tickLine={false} axisLine={false} />
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
        <p className="mt-2 text-xl font-mono font-bold leading-none tracking-tight text-primary">{typeof value === "number" ? value.toLocaleString() : value}</p>
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
