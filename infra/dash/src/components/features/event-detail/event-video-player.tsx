import { Eye } from "lucide-react"
import Hls from "hls.js"
import * as dashjs from "dashjs"
import { MediaOutlet, MediaPlayer, MediaPoster } from "@vidstack/react"
import { useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { useMediaPlayerStore } from "@/stores/media-player-store"

interface EventVideoPlayerProps {
  url: string
  poster?: string
  lazy?: boolean
  className?: string
  showChrome?: boolean
}

export function EventVideoPlayer({ url, poster, lazy = false, className, showChrome = true }: EventVideoPlayerProps) {
  const { t } = useTranslation()
  const [loaded, setLoaded] = useState(!lazy)
  const preferMuted = useMediaPlayerStore((state: { preferMuted: boolean }) => state.preferMuted)

  const sourceMeta = useMemo(() => {
    const lower = url.toLowerCase()
    const isHls = lower.includes(".m3u8")
    const isDash = lower.includes(".mpd")

    return {
      engine: isHls ? "hls.js" : isDash ? "dashjs" : "native",
      supported: isHls ? Hls.isSupported() || typeof document !== "undefined" : isDash ? dashjs.supportsMediaSource() : true,
    }
  }, [url])

  useEffect(() => {
    console.info("[vidstack] player source", {
      engine: sourceMeta.engine,
      supported: sourceMeta.supported,
      url,
    })
  }, [sourceMeta.engine, sourceMeta.supported, url])

  const content = loaded ? (
    <MediaPlayer
      className={className ?? "max-h-[420px] w-full overflow-hidden rounded-md border border-border bg-black object-contain"}
      controls
      crossOrigin
      muted={preferMuted}
      playsInline
      poster={poster || undefined}
      src={url}
      streamType="on-demand"
      title="Event media"
      viewType="video"
    >
      {poster ? <MediaPoster alt={t("eventDetail.eventImageAlt")} src={poster} /> : null}
      <MediaOutlet />
    </MediaPlayer>
  ) : (
    <button
      className={className ?? "flex h-56 w-full items-center justify-center rounded-md border border-border bg-black/90 text-white transition-colors hover:bg-black"}
      onClick={() => setLoaded(true)}
      type="button"
    >
      <span className="inline-flex items-center gap-2 rounded-full bg-background/85 px-4 py-2 text-sm text-foreground shadow-sm">
        <Eye className="size-4" />
        {t("eventDetail.loadVideo", "Carregar vídeo")}
      </span>
    </button>
  )

  if (!showChrome) {
    return content
  }

  return (
    <div className="min-w-0 space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/20 p-4">
      <div className="flex items-center justify-between gap-2">
        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("eventDetail.videoPlayer")}</p>
        <span className="text-[10px] uppercase tracking-[0.12em] text-muted-foreground">{sourceMeta.engine}{sourceMeta.supported ? "" : " unsupported"}</span>
      </div>
      {content}
    </div>
  )
}
