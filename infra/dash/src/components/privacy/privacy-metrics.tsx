import { useTranslation } from "react-i18next"

import type { PrivacyNetworkMetrics } from "@/types/admin"

export function bytesToHuman(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB"]
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit += 1
  }
  return `${value >= 100 || value === Math.floor(value) ? value.toFixed(0) : value.toFixed(1)} ${units[unit]}`
}

export function uptimeMsToHuman(uptimeMs: number): string {
  if (!Number.isFinite(uptimeMs) || uptimeMs < 0) return "0s"
  const totalSeconds = Math.floor(uptimeMs / 1000)
  const days = Math.floor(totalSeconds / 86400)
  const hours = Math.floor((totalSeconds % 86400) / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  if (minutes > 0) return `${minutes}m`
  return `${totalSeconds}s`
}

interface PrivacyMetricsProps {
  metrics?: PrivacyNetworkMetrics | null
}

export function PrivacyMetrics({ metrics }: PrivacyMetricsProps) {
  const { t } = useTranslation()
  if (!metrics) {
    return <p className="text-sm text-muted-foreground">{t("privacy.noMetrics")}</p>
  }

  const rows: { label: string; value: string }[] = [
    { label: t("privacy.tx"), value: bytesToHuman(metrics.tx_bytes) },
    { label: t("privacy.rx"), value: bytesToHuman(metrics.rx_bytes) },
  ]
  if (metrics.peers != null) {
    rows.push({ label: t("privacy.peers"), value: String(metrics.peers) })
  }
  if (metrics.connections != null) {
    rows.push({ label: t("privacy.connections"), value: String(metrics.connections) })
  }

  return (
    <dl className="grid grid-cols-2 gap-3 text-sm">
      {rows.map((row) => (
        <div key={row.label} className="rounded border border-border px-3 py-2">
          <dt className="text-xs uppercase tracking-wider text-muted-foreground">{row.label}</dt>
          <dd className="mt-0.5 font-medium">{row.value}</dd>
        </div>
      ))}
    </dl>
  )
}
