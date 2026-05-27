import { create } from "zustand"
import { createJSONStorage, persist } from "zustand/middleware"
import { immer } from "zustand/middleware/immer"

type RelayPresetsStoreState = {
  scopes: Record<string, string[]>
  getScopeRelays: (scope: string, fallback: string[]) => string[]
  setScopeRelays: (scope: string, relays: string[]) => void
}

export const useRelayPresetsStore = create<RelayPresetsStoreState>()(
  persist(
    immer((set, get) => ({
      scopes: {},
      getScopeRelays: (scope, fallback) => get().scopes[scope] ?? fallback,
      setScopeRelays: (scope, relays) => set((state) => {
        state.scopes[scope] = relays
      }),
    })),
    {
      name: "relay-presets-store",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({ scopes: state.scopes }),
    },
  ),
)
