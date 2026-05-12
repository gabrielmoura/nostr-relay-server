import { ExternalLink } from "lucide-react"
import { useTranslation } from "react-i18next"

import { EventVideoPlayer } from "./event-video-player"
import { MediaCarousel } from "./media-carousel"
import { collectAltTexts, collectMediaAssets, parseImetaResources } from "@/lib/event-parser"
import type { TagTuple } from "@/lib/event-parser"

interface EventMediaProps {
  content: string
  tags: TagTuple[]
  imageURLs: string[]
  kind: number
}

import type { MediaItem } from "./media-carousel"

export function EventMedia({ content, tags, imageURLs, kind }: EventMediaProps) {
  const { t } = useTranslation()
  const imeta = parseImetaResources(tags)
  const altTexts = collectAltTexts(tags)
  const assets = collectMediaAssets(tags, content, imageURLs)
  const mediaURLs = assets.map((asset) => asset.url)
  const videoPoster = assets.find((asset) => asset.type === "image")?.url ?? imageURLs[0] ?? ""
  const unifiedMedia: MediaItem[] = assets.map((asset) => ({ type: asset.type, url: asset.url, alt: asset.alt || altTexts[0] }))

  return (
    <>
      {unifiedMedia.length > 1 ? (
        <MediaCarousel poster={videoPoster} media={unifiedMedia} />
      ) : unifiedMedia.length === 1 ? (
        unifiedMedia[0]!.type === "video" ? (
          <EventVideoPlayer poster={videoPoster} url={unifiedMedia[0]!.url} />
        ) : (
          <img alt={unifiedMedia[0]!.alt || t("eventDetail.eventImageAlt")} className="max-h-[60vh] w-full rounded-md border border-border object-contain" src={unifiedMedia[0]!.url} />
        )
      ) : null}
      {mediaURLs.length > 0 && (
        <div className="space-y-2 rounded-[calc(var(--radius)-0.2rem)] border border-border bg-muted/20 p-4">
          <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">{t("eventDetail.mediaUrlsImeta")}</p>
          <div className="space-y-2">
            {mediaURLs.map((url) => (
              <a
                className="flex items-center gap-1 break-all text-sm text-primary underline decoration-dotted underline-offset-2"
                href={url}
                key={url}
                rel="noreferrer"
                target="_blank"
              >
                <ExternalLink className="size-3.5" />
                {url}
              </a>
            ))}
          </div>
          {altTexts.length > 0 ? <p className="break-words text-xs text-muted-foreground">{t("eventDetail.altPrefix")} {altTexts.join(" | ")}</p> : null}
        </div>
      )}
    </>
  )
}

interface EventImetaInfoProps {
  mimeTypes: string[]
  altTexts: string[]
}

export function EventImetaInfo({ mimeTypes, altTexts }: EventImetaInfoProps) {
  const { t } = useTranslation()

  if (mimeTypes.length === 0 && altTexts.length === 0) return null

  return (
    <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border p-3">
      <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{t("eventDetail.imeta")}</p>
      {mimeTypes.length > 0 ? <p className="mt-1 break-all text-xs text-foreground">{t("eventDetail.mimePrefix")} {mimeTypes.join(", ")}</p> : null}
      {altTexts.length > 0 ? <p className="mt-1 break-all text-xs text-foreground">{t("eventDetail.altPrefix")} {altTexts.join(" | ")}</p> : null}
    </div>
  )
}
