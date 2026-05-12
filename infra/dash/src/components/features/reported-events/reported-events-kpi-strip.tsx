import { RadialBar, RadialBarChart, ResponsiveContainer } from "recharts"

import { Card, CardContent } from "@/components/ui/card"
import type { ReportedKpiMetric } from "@/lib/reported-events-analytics"

interface ReportedEventsKpiStripProps {
  metrics: ReportedKpiMetric[]
}

export function ReportedEventsKpiStrip({ metrics }: ReportedEventsKpiStripProps) {
  return (
    <div className="grid gap-3 md:grid-cols-3">
      {metrics.map((metric) => (
        <Card className="overflow-hidden border-primary/5 bg-card/60 backdrop-blur-sm" key={metric.key}>
          <CardContent className="flex items-center justify-between gap-4 p-4">
            <div>
              <p className="text-xs font-bold uppercase tracking-wider text-muted-foreground">{metric.label}</p>
              <p className="mt-2 font-mono text-2xl font-bold tracking-tight text-primary">{metric.value.toLocaleString()}</p>
              {metric.helper ? <p className="mt-1 text-xs text-muted-foreground">{metric.helper}</p> : null}
            </div>
            <div className="h-16 w-16 shrink-0">
              <ResponsiveContainer height="100%" width="100%">
                <RadialBarChart cx="50%" cy="50%" data={[{ value: metric.progress * 100 }]} endAngle={-270} innerRadius="70%" outerRadius="100%" startAngle={90}>
                  <RadialBar background cornerRadius={10} dataKey="value" fill="var(--color-primary)" />
                </RadialBarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
