import { useTranslation } from "react-i18next"

interface EventVideoPlayerProps {
  url: string
  poster?: string
}

export function EventVideoPlayer({ url, poster }: EventVideoPlayerProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/20 p-4">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("eventDetail.videoPlayer")}</p>
      <video
        className="max-h-[420px] w-full rounded-md border border-border bg-black"
        controls
        poster={poster || undefined}
        preload="metadata"
        src={url}
      />
    </div>
  )
}