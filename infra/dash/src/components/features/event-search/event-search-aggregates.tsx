import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { shortenId } from "@/lib/utils"

interface AggregatesData {
  kinds: Array<{ kind: number; count: number }>
  top_authors: Array<{ pubkey: string; count: number }>
  top_tags: Array<{ tag: string; count: number }>
}

interface EventSearchAggregatesProps {
  data?: AggregatesData
  isLoading: boolean
  isError: boolean
  onRetry: () => void
}

export function EventSearchAggregates({ data, isLoading, isError, onRetry }: EventSearchAggregatesProps) {
  const { t } = useTranslation()

  if (isLoading) return null
  if (isError) return null
  if (!data) return null

  return (
    <div className="grid gap-4 lg:grid-cols-3">
      <Card>
        <CardContent className="space-y-2 p-4">
          <p className="font-heading text-sm font-semibold">{t("eventSearch.frequentKinds")}</p>
          {data.kinds.map((item) => (
            <div className="flex items-center justify-between text-sm" key={`kind-${item.kind}`}>
              <span>kind {item.kind}</span>
              <Badge variant="muted">{item.count}</Badge>
            </div>
          ))}
        </CardContent>
      </Card>
      <Card>
        <CardContent className="space-y-2 p-4">
          <p className="font-heading text-sm font-semibold">{t("eventSearch.activeAuthors")}</p>
          {data.top_authors.map((item) => (
            <div className="flex items-center justify-between text-sm" key={`author-${item.pubkey}`}>
              <Link className="underline decoration-dotted underline-offset-2" params={{ pubkey: item.pubkey }} to="/users/$pubkey">
                {shortenId(item.pubkey, 10, 4)}
              </Link>
              <Badge variant="muted">{item.count}</Badge>
            </div>
          ))}
        </CardContent>
      </Card>
      <Card>
        <CardContent className="space-y-2 p-4">
          <p className="font-heading text-sm font-semibold">{t("eventSearch.commonTags")}</p>
          {data.top_tags.map((item) => (
            <div className="flex items-center justify-between text-sm" key={`tag-${item.tag}`}>
              <Link search={{ tags: item.tag }} to="/events/search">
                #{item.tag}
              </Link>
              <Badge variant="muted">{item.count}</Badge>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}