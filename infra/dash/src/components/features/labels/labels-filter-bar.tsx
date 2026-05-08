import { useState } from "react"
import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { labelBadgeVariant, labelPresets, labelTargetOptions, normalizeLabelValue } from "@/lib/labels"
import type { AdminLabelsFilters, AdminLabelsSummary } from "@/types/admin"

const allValue = "__all__"

type LabelsFilterBarProps = {
  filters: AdminLabelsFilters
  summary?: AdminLabelsSummary
  onChange: (patch: Partial<AdminLabelsFilters>) => void
  onReset: () => void
}

export function LabelsFilterBar({ filters, summary, onChange, onReset }: LabelsFilterBarProps) {
  const { t } = useTranslation()
  const [customLabel, setCustomLabel] = useState("")

  const handleAddCustomLabel = () => {
    const normalized = normalizeLabelValue(customLabel)
    if (!normalized) {
      return
    }

    const current = new Set(filters.labels ?? [])
    current.add(normalized)
    onChange({ labels: [...current] })
    setCustomLabel("")
  }

  const selectedLabels = filters.labels ?? []

  const toggleLabel = (label: string) => {
    const normalized = normalizeLabelValue(label)
    const current = new Set(selectedLabels)
    if (current.has(normalized)) {
      current.delete(normalized)
    } else {
      current.add(normalized)
    }
    onChange({ labels: [...current] })
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle>{t("labels.filters.title", "Operational filters")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 lg:grid-cols-[2fr_1fr_1fr]">
          <Input
            onChange={(event) => onChange({ q: event.target.value || undefined })}
            placeholder={t("labels.filters.searchPlaceholder", "Buscar por target, comentario ou autor")}
            value={filters.q ?? ""}
          />

          <Select onValueChange={(value) => onChange({ namespace: value === allValue ? undefined : value })} value={filters.namespace ?? allValue}>
            <SelectTrigger>
              <SelectValue placeholder={t("labels.filters.namespace", "Namespace")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={allValue}>{t("labels.filters.allNamespaces", "Todos os namespaces")}</SelectItem>
              {(summary?.namespaces ?? []).map((item) => (
                <SelectItem key={item.namespace} value={item.namespace ?? ""}>{item.namespace}</SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select onValueChange={(value) => onChange({ target_type: value === allValue ? undefined : (value as AdminLabelsFilters["target_type"]) })} value={filters.target_type ?? allValue}>
            <SelectTrigger>
              <SelectValue placeholder={t("labels.filters.targetType", "Tipo de alvo")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={allValue}>{t("labels.filters.allTargetTypes", "Todos os tipos")}</SelectItem>
              {labelTargetOptions.map((option) => (
                <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-3 rounded-[calc(var(--radius)-0.25rem)] border border-border bg-muted/20 p-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-medium text-foreground">{t("labels.filters.label", "Label")}</span>
            {selectedLabels.length > 0 ? (
              selectedLabels.map((label) => (
                <button className="cursor-pointer" key={label} onClick={() => toggleLabel(label)} type="button">
                  <Badge variant={labelBadgeVariant(label)}>{label}</Badge>
                </button>
              ))
            ) : (
              <Badge variant="muted">{t("labels.filters.noLabelSelected", "Nenhum label selecionado")}</Badge>
            )}
          </div>

          <div className="flex flex-wrap gap-2">
            {labelPresets.map((preset) => {
              const active = selectedLabels.includes(preset.value)
              return (
                <Button
                  className="h-auto justify-start py-1.5"
                  key={preset.value}
                  onClick={() => toggleLabel(preset.value)}
                  type="button"
                  variant={active ? "default" : "outline"}
                >
                  {preset.value}
                </Button>
              )
            })}

            {(summary?.labels ?? [])
              .map((item) => item.label)
              .filter((value): value is string => Boolean(value) && !labelPresets.some((preset) => preset.value === value))
              .slice(0, 8)
              .map((label) => {
                const active = selectedLabels.includes(label)
                return (
                  <Button key={label} onClick={() => toggleLabel(label)} size="sm" type="button" variant={active ? "default" : "outline"}>
                    {label}
                  </Button>
                )
              })}
          </div>

          <div className="flex gap-2">
            <Input
              onChange={(event) => setCustomLabel(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") {
                  event.preventDefault()
                  handleAddCustomLabel()
                }
              }}
              placeholder={t("labels.filters.customLabelPlaceholder", "Filtrar por label customizado")}
              value={customLabel}
            />
            <Button onClick={handleAddCustomLabel} type="button" variant="outline">
              {t("labels.filters.addCustom", "Aplicar")}
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3">
          <Input
            className="max-w-md"
            onChange={(event) => onChange({ target: event.target.value || undefined })}
            placeholder={t("labels.filters.targetPlaceholder", "Filtrar por target exato (event id, pubkey, a/r/t)")}
            value={filters.target ?? ""}
          />

          <Button onClick={onReset} type="button" variant="outline">
            {t("labels.filters.reset", "Limpar filtros")}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
