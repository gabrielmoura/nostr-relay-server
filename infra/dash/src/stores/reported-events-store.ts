import { create } from "zustand"
import { createJSONStorage, persist } from "zustand/middleware"
import { immer } from "zustand/middleware/immer"

import type { ReportedEventsFilters } from "@/types/admin"

interface ReportedEventsStoreState {
  filters: ReportedEventsFilters
  selectedChart: "timeline" | "type" | "author" | "target" | null
  setQuery: (query: string) => void
  setType: (type: string) => void
  setTimelineBucket: (bucket: string) => void
  setAuthor: (pubkey: string) => void
  setTarget: (eventID: string) => void
  clearChartSelection: () => void
  resetFilters: () => void
}

const initialFilters: ReportedEventsFilters = {
  query: "",
  type: "all",
}

function parseBucketRange(bucket: string) {
  const start = new Date(`${bucket}T00:00:00Z`)
  const end = new Date(`${bucket}T23:59:59Z`)
  return {
    since: Math.floor(start.getTime() / 1000),
    until: Math.floor(end.getTime() / 1000),
  }
}

export const useReportedEventsStore = create<ReportedEventsStoreState>()(
  persist(
    immer((set) => ({
      filters: initialFilters,
      selectedChart: null,
      setQuery: (query) => set((state) => {
        state.filters.query = query
      }),
      setType: (type) => set((state) => {
        state.filters.type = type
        state.selectedChart = type === "all" ? null : "type"
      }),
      setTimelineBucket: (bucket) => set((state) => {
        const { since, until } = parseBucketRange(bucket)
        state.filters.since = since
        state.filters.until = until
        state.selectedChart = "timeline"
      }),
      setAuthor: (pubkey) => set((state) => {
        state.filters.target_pubkey = pubkey
        state.selectedChart = "author"
      }),
      setTarget: (eventID) => set((state) => {
        state.filters.target_event_id = eventID
        state.selectedChart = "target"
      }),
      clearChartSelection: () => set((state) => {
        delete state.filters.target_pubkey
        delete state.filters.target_event_id
        delete state.filters.since
        delete state.filters.until
        if (state.selectedChart === "type") {
          state.filters.type = "all"
        }
        state.selectedChart = null
      }),
      resetFilters: () => set((state) => {
        state.filters = initialFilters
        state.selectedChart = null
      }),
    })),
    {
      name: "reported-events-store",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        filters: state.filters,
        selectedChart: state.selectedChart,
      }),
    },
  ),
)
