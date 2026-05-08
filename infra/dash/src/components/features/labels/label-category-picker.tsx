import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { labelBadgeVariant, labelPresets } from "@/lib/labels"

type LabelCategoryPickerProps = {
  selected: string[]
  customValue: string
  onCustomValueChange: (value: string) => void
  onAddCustom: () => void
  onTogglePreset: (value: string) => void
  onRemove: (value: string) => void
}

export function LabelCategoryPicker({ selected, customValue, onCustomValueChange, onAddCustom, onTogglePreset, onRemove }: LabelCategoryPickerProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        {selected.map((value) => (
          <button className="cursor-pointer" key={value} onClick={() => onRemove(value)} type="button">
            <Badge variant={labelBadgeVariant(value)}>{value}</Badge>
          </button>
        ))}
        {selected.length === 0 ? <Badge variant="muted">{t("labels.create.noLabels", "Nenhum label selecionado")}</Badge> : null}
      </div>

      <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
        {labelPresets.map((preset) => {
          const active = selected.includes(preset.value)
          return (
            <Button
              className="justify-start"
              key={preset.value}
              onClick={() => onTogglePreset(preset.value)}
              type="button"
              variant={active ? "default" : "outline"}
            >
              {preset.value}
            </Button>
          )
        })}
      </div>

      <div className="flex gap-2">
        <Input onChange={(event) => onCustomValueChange(event.target.value)} placeholder={t("labels.create.customPlaceholder", "Adicionar label customizado")}
          value={customValue}
        />
        <Button onClick={onAddCustom} type="button" variant="outline">
          {t("labels.create.addCustom", "Adicionar")}
        </Button>
      </div>
    </div>
  )
}
