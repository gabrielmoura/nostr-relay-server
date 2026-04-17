import { ExternalLink } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { EventImageGrid } from "./event-image-grid"
import { EventVideoPlayer } from "./event-video-player"
import { parseImetaResources, parseMediaURLsFromTags, pickVideoURL, VIDEO_KINDS } from "@/lib/event-parser"
import { unique } from "@/lib/event-parser"
import type { TagTuple } from "@/lib/event-parser"

interface EventMediaProps {
  content: string
  tags: TagTuple[]
  imageURLs: string[]
  kind: number
}

export function EventMedia({ content, tags, imageURLs, kind }: EventMediaProps) {
  const { t } = useTranslation()
  const imeta = parseImetaResources(tags)
  const mediaURLs = unique([...imeta.mediaURLs, ...parseMediaURLsFromTags(tags)])
  const videoURL = VIDEO_KINDS.includes(kind) ? pickVideoURL(mediaURLs) : null
  const videoPoster = imageURLs[0] ?? ""

  return (
    <>
      {imageURLs.length > 0 && <EventImageGrid urls={imageURLs} />}
      {videoURL && <EventVideoPlayer poster={videoPoster} url={videoURL} />}
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