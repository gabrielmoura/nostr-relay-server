import { useNavigate, useSearch } from "@tanstack/react-router"

import { LabelsWorkspace } from "@/components/features/labels/labels-workspace"
import { parseLabelsRouteLabels, serializeLabelsRouteLabels, type LabelsRouteSearch, type LabelsRouteView } from "@/lib/labels"

export function LabelsPage() {
  const navigate = useNavigate({ from: "/labels" })
  const search = useSearch({ from: "/labels" })

  return (
    <LabelsWorkspace
      filters={{
        namespace: search.namespace,
        labels: parseLabelsRouteLabels(search.labels),
        q: search.q,
        target: search.target,
        target_type: search.targetType,
      }}
      onFiltersChange={(patch) => {
        void navigate({
          search: (previous: LabelsRouteSearch) => ({
            ...previous,
            namespace: patch.namespace || undefined,
            labels: serializeLabelsRouteLabels(patch.labels ?? parseLabelsRouteLabels(previous.labels)),
            q: patch.q || undefined,
            target: patch.target || undefined,
            targetType: patch.target_type || undefined,
          }),
        })
      }}
      onResetFilters={() => {
        void navigate({ search: (previous: LabelsRouteSearch) => ({ view: previous.view ?? "timeline" }) })
      }}
      onViewChange={(nextView: LabelsRouteView) => {
        void navigate({ search: (previous: LabelsRouteSearch) => ({ ...previous, view: nextView }) })
      }}
      view={search.view ?? "timeline"}
    />
  )
}
