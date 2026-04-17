import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Search } from "lucide-react"
import type { EventSearchFilters } from "@/types/admin"
import { defaultFilters } from "@/lib/event-search"

interface EventSearchFormProps {
  draft: EventSearchFilters
  onDraftChange: (draft: EventSearchFilters) => void
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void
  onClear: () => void
}

export function EventSearchForm({ draft, onDraftChange, onSubmit, onClear }: EventSearchFormProps) {
  const { t } = useTranslation()

  return (
    <form className="space-y-3" onSubmit={onSubmit}>
      <div className="grid gap-3 lg:grid-cols-[2fr_1fr_1fr_auto]">
        <Input
          placeholder={t("eventSearch.queryPlaceholder")}
          value={draft.query}
          onChange={(event) => onDraftChange({ ...draft, query: event.target.value })}
        />
        <Input
          placeholder={t("eventSearch.kindsPlaceholder")}
          value={draft.kinds.join(",")}
          onChange={(event) =>
            onDraftChange({
              ...draft,
              kinds: event.target.value
                .split(",")
                .map((value) => Number(value.trim()))
                .filter((value) => !Number.isNaN(value)),
            })
          }
        />
        <Input
          placeholder={t("eventSearch.tagsPlaceholder")}
          value={draft.tags.join(",")}
          onChange={(event) =>
            onDraftChange({
              ...draft,
              tags: event.target.value.split(",").map((value) => value.trim()).filter(Boolean),
            })
          }
        />
        <Button type="submit">
          <Search className="size-4" />
          {t("common.search")}
        </Button>
      </div>
      <div className="grid gap-3 lg:grid-cols-[2fr_1fr_auto]">
        <Input
          placeholder={t("eventSearch.authorsPlaceholder")}
          value={draft.authors.join(",")}
          onChange={(event) =>
            onDraftChange({
              ...draft,
              authors: event.target.value.split(",").map((value) => value.trim()).filter(Boolean),
            })
          }
        />
        <Input
          placeholder={t("eventSearch.limitPlaceholder")}
          type="number"
          value={draft.limit}
          onChange={(event) => onDraftChange({ ...draft, limit: Number(event.target.value) || 100 })}
        />
        <Button
          type="button"
          variant="outline"
          onClick={onClear}
        >
          {t("eventSearch.clear")}
        </Button>
      </div>
    </form>
  )
}