import { create } from "zustand"
import { createJSONStorage, persist } from "zustand/middleware"
import { immer } from "zustand/middleware/immer"

interface MediaPlayerStoreState {
  preferMuted: boolean
  setPreferMuted: (value: boolean) => void
}

export const useMediaPlayerStore = create<MediaPlayerStoreState>()(
  persist(
    immer((set) => ({
      preferMuted: false,
      setPreferMuted: (value) => set((state) => {
        state.preferMuted = value
      }),
    })),
    {
      name: "media-player-store",
      storage: createJSONStorage(() => localStorage),
    },
  ),
)
