import { create } from "zustand"
import { createJSONStorage, persist } from "zustand/middleware"
import { immer } from "zustand/middleware/immer"

import { clearMirrorHistoryEntries, loadMirrorHistory, saveMirrorHistoryEntry, type BlossomMirrorHistoryEntry } from "@/lib/indexeddb"

type BlossomOperatorStoreState = {
  recentMimeFilters: string[]
  mirrorDraft: {
    sourceURL: string
    expectedSHA256: string
  }
  mirrorHistory: BlossomMirrorHistoryEntry[]
  mirrorHistoryLoaded: boolean
  addRecentMimeFilter: (value: string) => void
  setMirrorDraft: (draft: { sourceURL?: string; expectedSHA256?: string }) => void
  resetMirrorDraft: () => void
  hydrateMirrorHistory: () => Promise<void>
  addMirrorHistory: (entry: BlossomMirrorHistoryEntry) => Promise<void>
  clearMirrorHistory: () => Promise<void>
}

export const useBlossomOperatorStore = create<BlossomOperatorStoreState>()(
  persist(
    immer((set, get) => ({
      recentMimeFilters: [],
      mirrorDraft: {
        sourceURL: "",
        expectedSHA256: "",
      },
      mirrorHistory: [],
      mirrorHistoryLoaded: false,
      addRecentMimeFilter: (value) => set((state) => {
        const normalized = value.trim().toLowerCase()
        if (!normalized || !normalized.includes("/")) {
          return
        }
        state.recentMimeFilters = [normalized, ...state.recentMimeFilters.filter((item) => item !== normalized)].slice(0, 8)
      }),
      setMirrorDraft: (draft) => set((state) => {
        if (draft.sourceURL !== undefined) {
          state.mirrorDraft.sourceURL = draft.sourceURL
        }
        if (draft.expectedSHA256 !== undefined) {
          state.mirrorDraft.expectedSHA256 = draft.expectedSHA256
        }
      }),
      resetMirrorDraft: () => set((state) => {
        state.mirrorDraft.sourceURL = ""
        state.mirrorDraft.expectedSHA256 = ""
      }),
      hydrateMirrorHistory: async () => {
        const items = await loadMirrorHistory()
        set((state) => {
          state.mirrorHistory = items
          state.mirrorHistoryLoaded = true
        })
      },
      addMirrorHistory: async (entry) => {
        await saveMirrorHistoryEntry(entry)
        set((state) => {
          state.mirrorHistory = [entry, ...state.mirrorHistory.filter((item) => item.id !== entry.id)].slice(0, 20)
          state.mirrorHistoryLoaded = true
        })
      },
      clearMirrorHistory: async () => {
        await clearMirrorHistoryEntries()
        set((state) => {
          state.mirrorHistory = []
          state.mirrorHistoryLoaded = true
        })
      },
    })),
    {
      name: "blossom-operator-store",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({ recentMimeFilters: state.recentMimeFilters, mirrorDraft: state.mirrorDraft }),
    },
  ),
)
