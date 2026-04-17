import { useTranslation } from "react-i18next"

interface EventImageGridProps {
  urls: string[]
}

export function EventImageGrid({ urls }: EventImageGridProps) {
  const { t } = useTranslation()

  if (urls.length === 0) return null

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {urls.map((url) => (
        <a href={url} key={url} rel="noreferrer" target="_blank">
          <img alt={t("eventDetail.eventImageAlt")} className="h-52 w-full rounded-md border border-border object-cover" src={url} />
        </a>
      ))}
    </div>
  )
}