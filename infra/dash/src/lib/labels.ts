import type { AdminLabelCount, AdminLabelEvent, AdminLabelTargetType } from "@/types/admin"

export type LabelsRouteView = "timeline" | "targets"

export type LabelsRouteSearch = {
  namespace?: string
  label?: string
  targetType?: AdminLabelTargetType
  target?: string
  q?: string
  view?: LabelsRouteView
}

export type LabelPreset = {
  value: string
  namespace: string
  severity: "p0" | "p1" | "p2" | "p3"
}

export type GroupedLabelTarget = {
  key: string
  type: AdminLabelTargetType
  value: string
  relayHint?: string
  labels: string[]
  namespaceCounts: Record<string, number>
  eventsCount: number
  lastCreatedAt: number
  lastComment?: string
}

export const labelNamespaces = ["ugc", "content-warning", "dtsp", "legal", "moderation/resolution"]

export const labelTargetOptions: Array<{ value: AdminLabelTargetType; label: string }> = [
  { value: "event", label: "Event" },
  { value: "pubkey", label: "Pubkey" },
  { value: "address", label: "Address" },
  { value: "reference", label: "Reference" },
  { value: "topic", label: "Topic" },
]

export const labelPresets: LabelPreset[] = [
  { value: "csam", namespace: "dtsp", severity: "p0" },
  { value: "terrorism", namespace: "dtsp", severity: "p0" },
  { value: "credible_threats", namespace: "dtsp", severity: "p1" },
  { value: "doxxing", namespace: "dtsp", severity: "p1" },
  { value: "malware", namespace: "dtsp", severity: "p1" },
  { value: "nonconsensual", namespace: "dtsp", severity: "p1" },
  { value: "hate", namespace: "dtsp", severity: "p2" },
  { value: "illegal_goods", namespace: "dtsp", severity: "p2" },
  { value: "violence", namespace: "dtsp", severity: "p2" },
  { value: "self_harm", namespace: "dtsp", severity: "p2" },
  { value: "nsfw", namespace: "content-warning", severity: "p3" },
  { value: "spam", namespace: "ugc", severity: "p3" },
  { value: "impersonation", namespace: "ugc", severity: "p2" },
  { value: "copyright", namespace: "legal", severity: "p3" },
]

export function normalizeLabelValue(value: string) {
  return value.trim().toLowerCase()
}

export function normalizeLabelsSearch(search: Record<string, unknown>): LabelsRouteSearch {
  const view = search.view === "targets" ? "targets" : "timeline"
  const targetType = typeof search.targetType === "string" ? search.targetType : undefined
  return {
    namespace: typeof search.namespace === "string" && search.namespace ? search.namespace : undefined,
    label: typeof search.label === "string" && search.label ? search.label : undefined,
    targetType: isLabelTargetType(targetType) ? targetType : undefined,
    target: typeof search.target === "string" && search.target ? search.target : undefined,
    q: typeof search.q === "string" && search.q ? search.q : undefined,
    view,
  }
}

export function groupLabelsByTarget(items: AdminLabelEvent[]): GroupedLabelTarget[] {
  const grouped = new Map<string, GroupedLabelTarget>()

  for (const item of items) {
    const key = `${item.target.type}:${item.target.value}`
    const current = grouped.get(key)
    if (current) {
      current.eventsCount += 1
      current.lastCreatedAt = Math.max(current.lastCreatedAt, item.created_at)
      current.lastComment = current.lastComment || item.content || undefined
      current.namespaceCounts[item.namespace] = (current.namespaceCounts[item.namespace] ?? 0) + 1
      for (const label of item.labels) {
        if (!current.labels.includes(label)) {
          current.labels.push(label)
        }
      }
      continue
    }

    grouped.set(key, {
      key,
      type: item.target.type,
      value: item.target.value,
      relayHint: item.target.relay_hint,
      labels: [...item.labels],
      namespaceCounts: { [item.namespace]: 1 },
      eventsCount: 1,
      lastCreatedAt: item.created_at,
      lastComment: item.content || undefined,
    })
  }

  return [...grouped.values()].sort((a, b) => b.lastCreatedAt - a.lastCreatedAt)
}

export function labelBadgeVariant(label: string): "default" | "danger" | "warning" | "success" | "muted" {
  const normalized = normalizeLabelValue(label)
  if (["csam", "terrorism", "credible_threats"].includes(normalized)) {
    return "danger"
  }
  if (["malware", "violence", "illegal_goods", "impersonation", "hate", "doxxing"].includes(normalized)) {
    return "warning"
  }
  if (["reviewed", "dismissed", "false-positive", "no-action"].includes(normalized)) {
    return "success"
  }
  if (["nsfw", "spam", "copyright", "self_harm", "nonconsensual"].includes(normalized)) {
    return "muted"
  }
  return "default"
}

export function topSummaryLabel(items: AdminLabelCount[]) {
  return items[0]?.label ?? items[0]?.namespace ?? items[0]?.target_type ?? "-"
}

function isLabelTargetType(value: string | undefined): value is AdminLabelTargetType {
  return ["event", "pubkey", "address", "reference", "topic"].includes(value ?? "")
}
