import { useNavigate, useSearch } from "@tanstack/react-router"

import { BlossomWorkspace } from "@/components/features/blossom/blossom-workspace"
import { normalizeBlossomSearch, type BlossomRouteSearch } from "@/lib/blossom"

export function BlossomPage() {
  const navigate = useNavigate({ from: "/blossom" })
  const search = useSearch({ from: "/blossom" })

  return (
    <BlossomWorkspace
      onSearchChange={(patch) => {
        void navigate({
          search: (previous: BlossomRouteSearch) => ({
            ...previous,
            ...patch,
          }),
        })
      }}
      search={normalizeBlossomSearch(search)}
    />
  )
}
