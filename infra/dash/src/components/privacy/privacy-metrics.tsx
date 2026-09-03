import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import type { PrivacyNetworkMetrics } from "@/types/admin"

// How the traffic counter behaves per connection mode:
//  - "external": the relay's forwarder is reachable by the daemon and counts bytes.
//  - "native":   the privacy network runs in-process and the byte counters cannot
//                be wired to the forwarder, so TX/RX are unavailable (not zero).
const NATIVE_MODE = "native"

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
  /** Connection mode of the network: "native" | "external". */
  mode?: string
  metrics?: PrivacyNetworkMetrics | null
}

// TrafficValue renders a measured tx/rx value, or an explanatory "N/A" badge
// with a tooltip when byte counting is unavailable (native mode).
function TrafficValue({ available, value, reason }: { available: boolean; value: string; reason: string }) {
  const { t } = useTranslation()
  if (available) {
    return <span className="font-medium">{value}</span>
  }
  return (
    <TooltipProvider delayDuration={150}>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            tabIndex={-1}
            className="inline-flex cursor-help items-center rounded px-1.5 py-0.5 font-medium"
            aria-label={reason}
          >
            <Badge variant="muted">{t("privacy.na")}</Badge>
          </button>
        </TooltipTrigger>
        <TooltipContent>
          <p className="max-w-56 text-xs">{reason}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

export function PrivacyMetrics({ mode, metrics }: PrivacyMetricsProps) {
  const { t } = useTranslation()
  if (!metrics) {
    return <p className="text-sm text-muted-foreground">{t("privacy.noMetrics")}</p>
  }

  // In native mode the forwarder's byte counters cannot be wired to the
  // in-process daemon, so TX/RX are genuinely unavailable — never render "0 B"
  // as if it were real traffic.
  const trafficAvailable = mode !== NATIVE_MODE
  const unavailableReason = t("privacy.metricsUnavailableDescription")

  const rows: { label: string; value: ReactNode }[] = [
    { label: t("privacy.tx"), value: <TrafficValue available={trafficAvailable} value={bytesToHuman(metrics.tx_bytes)} reason={unavailableReason} /> },
    { label: t("privacy.rx"), value: <TrafficValue available={trafficAvailable} value={bytesToHuman(metrics.rx_bytes)} reason={unavailableReason} /> },
  ]
  if (metrics.peers != null) {
    rows.push({ label: t("privacy.peers"), value: <span className="font-medium">{String(metrics.peers)}</span> })
  }
  if (metrics.connections != null) {
    rows.push({ label: t("privacy.connections"), value: <span className="font-medium">{String(metrics.connections)}</span> })
  }

  return (
    <dl className="grid grid-cols-2 gap-3 text-sm">
      {rows.map((row) => (
        <div key={row.label} className="rounded border border-border px-3 py-2">
          <dt className="text-xs uppercase tracking-wider text-muted-foreground">{row.label}</dt>
          <dd className="mt-0.5">{row.value}</dd>
        </div>
      ))}
      {!trafficAvailable ? (
        <div className="col-span-2">
          <p className="text-xs italic text-muted-foreground">{t("privacy.metricsUnavailableTitle")}</p>
        </div>
      ) : null}
    </dl>
  )
}
