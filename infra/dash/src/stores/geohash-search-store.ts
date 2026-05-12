import { create } from "zustand"
import { createJSONStorage, persist } from "zustand/middleware"
import { immer } from "zustand/middleware/immer"

interface GeohashSearchStoreState {
  recent: string[]
  addRecent: (value: string) => void
}

export const useGeohashSearchStore = create<GeohashSearchStoreState>()(
  persist(
    immer((set) => ({
      recent: [],
      addRecent: (value) => set((state) => {
        const normalized = value.trim().toLowerCase()
        if (!normalized) {
          return
        }
        state.recent = [normalized, ...state.recent.filter((item) => item !== normalized)].slice(0, 8)
      }),
    })),
    {
      name: "geohash-search-store",
      storage: createJSONStorage(() => localStorage),
    },
  ),
)
