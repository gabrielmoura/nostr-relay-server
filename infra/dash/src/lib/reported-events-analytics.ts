import type { ReportedEventsSummary } from "@/types/admin"

export type ReportedKpiMetric = {
  key: string
  label: string
  value: number
  helper?: string
  progress: number
}

function ratio(value: number, max: number) {
  if (!max) return 0
  return Math.max(0.08, Math.min(1, value / max))
}

export function buildReportedKpiMetrics(summary: ReportedEventsSummary): ReportedKpiMetric[] {
  const max = Math.max(summary.total_events, summary.total_reports, summary.unique_target_authors, 1)
  const topType = summary.report_types[0]?.name

  return [
    {
      key: "events",
      label: "Eventos reportados",
      value: summary.total_events,
      progress: ratio(summary.total_events, max),
    },
    {
      key: "reports",
      label: "Reports acumulados",
      value: summary.total_reports,
      progress: ratio(summary.total_reports, max),
    },
    {
      key: "authors",
      label: "Autores distintos",
      value: summary.unique_target_authors,
      helper: topType ? `Tipo dominante: ${topType}` : undefined,
      progress: ratio(summary.unique_target_authors, max),
    },
  ]
}

export function reportTypeColor(type: string) {
  switch (type) {
    case "spam":
      return "#f59e0b"
    case "illegal":
      return "#dc2626"
    case "impersonation":
      return "#7c3aed"
    case "malware":
      return "#ea580c"
    case "nudity":
      return "#ec4899"
    case "profanity":
      return "#2563eb"
    default:
      return "#64748b"
  }
}
