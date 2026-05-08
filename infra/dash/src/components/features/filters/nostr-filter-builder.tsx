import { useState, useMemo } from "react"
import { useTranslation } from "react-i18next"
import { 
  Filter as FilterIcon, Plus, X, Search, Tag, Database, Hash, 
  GitBranch, Users, Calendar, Info, MapPin, Code, MessageSquare 
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { Separator } from "@/components/ui/separator"
import { normalizeFilterIdentifier } from "@/lib/nostr"

export interface NostrFilter {
  ids?: string[]
  authors?: string[]
  kinds?: number[]
  since?: number
  until?: number
  limit?: number
  search?: string
  [key: string]: any 
}

interface NostrFilterBuilderProps {
  initialFilter?: NostrFilter
  onChange: (filter: NostrFilter) => void
  title?: string
  description?: string
}

export function NostrFilterBuilder({ initialFilter = {}, onChange, title, description }: NostrFilterBuilderProps) {
  const { t } = useTranslation()
  const [filter, setFilter] = useState<NostrFilter>(initialFilter)

  // Local states for inputs
  const [inputKind, setInputKind] = useState("")
  const [inputAuthor, setInputAuthor] = useState("")
  const [inputID, setInputID] = useState("")
  const [inputTagKey, setInputTagKey] = useState("t")
  const [inputTagValue, setInputTagValue] = useState("")

  const updateFilter = (newFields: Partial<NostrFilter>) => {
    const updated = { ...filter, ...newFields }
    setFilter(updated)
    onChange(updated)
  }

  const addListItem = (field: "kinds" | "authors" | "ids", value: any) => {
    if (value === undefined || value === null || (typeof value === "string" && !value.trim())) return
    const normalizedValue = typeof value === "string" ? normalizeFilterListValue(field, value) : value
    const current = (filter[field] as any[]) || []
    if (!current.includes(normalizedValue)) {
      updateFilter({ [field]: [...current, normalizedValue] })
    }
  }

  const removeListItem = (field: "kinds" | "authors" | "ids", value: any) => {
    const current = (filter[field] as any[]) || []
    updateFilter({ [field]: current.filter((v: any) => v !== value) })
  }

  const addTag = (key?: string, value?: string) => {
    const k = key || inputTagKey
    const v = value || inputTagValue
    if (!k.trim() || !v.trim()) return
    
    const actualKey = k.startsWith("#") ? k : `#${k}`
    const current = (filter[actualKey] as string[]) || []
    const normalizedValue = normalizeTagIdentifier(actualKey, v.trim())
    if (!current.includes(normalizedValue)) {
      updateFilter({ [actualKey]: [...current, normalizedValue] })
    }
    if (!key) setInputTagValue("")
  }

  const removeTag = (key: string, value: string) => {
    const current = (filter[key] as string[]) || []
    const updated = current.filter((v: string) => v !== value)
    const newFilter = { ...filter }
    if (updated.length === 0) {
      delete newFilter[key]
    } else {
      newFilter[key] = updated
    }
    setFilter(newFilter)
    onChange(newFilter)
  }

  const tagFilters = useMemo(() => {
    return Object.keys(filter)
      .filter(k => k.startsWith("#"))
      .map(k => ({ key: k, values: filter[k] as string[] }))
  }, [filter])

  const handleDateChange = (field: "since" | "until", value: string) => {
    if (!value) {
      const newFilter = { ...filter }
      delete newFilter[field]
      setFilter(newFilter)
      onChange(newFilter)
      return
    }
    const unix = Math.floor(new Date(value).getTime() / 1000)
    updateFilter({ [field]: unix })
  }

  const formatUnixForInput = (unix?: number) => {
    if (!unix) return ""
    return new Date(unix * 1000).toISOString().slice(0, 16)
  }

  const quickAddKind = (k: number) => addListItem("kinds", k)

  const handleNIPInput = (key: string, value: string) => {
    if (value) {
      addTag(key, value)
    }
  }

  const normalizeFilterListValue = (field: "kinds" | "authors" | "ids", value: string) => {
    if (field === "authors") {
      return normalizeFilterIdentifier("pubkey", value)
    }
    if (field === "ids") {
      return normalizeFilterIdentifier("event", value)
    }
    return value
  }

  const normalizeTagIdentifier = (tagKey: string, value: string) => {
    switch (tagKey.replace(/^#/, "")) {
      case "e":
      case "q":
        return normalizeFilterIdentifier("event", value)
      case "p":
        return normalizeFilterIdentifier("pubkey", value)
      case "a":
        return normalizeFilterIdentifier("address", value)
      default:
        return value.trim()
    }
  }

  return (
    <Card className="panel-shadow border-primary/10 bg-card/40 backdrop-blur-sm">
      <CardHeader className="pb-3 px-4">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2 text-md font-heading">
            <FilterIcon className="size-4 text-primary" />
            {title || t("events.search.filtersTitle", "Filtros Nostr")}
          </CardTitle>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="rounded-full bg-secondary/30 p-1 cursor-help hover:bg-secondary/50 transition-colors">
                  <Info className="size-3.5 text-muted-foreground" />
                </div>
              </TooltipTrigger>
              <TooltipContent className="max-w-xs p-3">
                <div className="space-y-2">
                  <p className="font-semibold text-xs border-bottom pb-1 mb-1">Padrões Suportados:</p>
                  <ul className="text-[10px] space-y-1 list-disc pl-3">
                    <li><b>NIP-01:</b> Filtros base e Tags genéricas.</li>
                    <li><b>NIP-50:</b> Busca por texto/regex (`search`).</li>
                    <li><b>NIP-24/52:</b> Buscas por `title`, `location`.</li>
                    <li><b>NIP-34/29:</b> Git (`name`, `c`) e Grupos (`h`).</li>
                    <li><b>Indexadores:</b> Tags de identidade externa (`i`).</li>
                  </ul>
                </div>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
        {description && <p className="text-[11px] text-muted-foreground mt-1 leading-relaxed">{description}</p>}
      </CardHeader>
      
      <CardContent className="px-4 pb-4">
        <Tabs defaultValue="basic" className="space-y-4">
          <TabsList className="grid w-full grid-cols-3 bg-secondary/20 p-1 rounded-md h-8">
            <TabsTrigger value="basic" className="text-[10px] gap-1.5 h-6 uppercase font-bold">
              <Database className="size-3" />
              {t("common.basic", "Básico")}
            </TabsTrigger>
            <TabsTrigger value="tags" className="text-[10px] gap-1.5 h-6 uppercase font-bold">
              <Hash className="size-3" />
              {t("common.tags", "Tags")}
            </TabsTrigger>
            <TabsTrigger value="nips" className="text-[10px] gap-1.5 h-6 uppercase font-bold">
              <Tag className="size-3" />
              NIPs
            </TabsTrigger>
          </TabsList>

          {/* BASIC FILTERS */}
          <TabsContent value="basic" className="space-y-4 pt-1 animate-in slide-in-from-left-2 duration-300">
            {/* Full Text Search (NIP-50) */}
            <div className="space-y-1.5">
              <Label className="text-[10px] text-muted-foreground uppercase flex items-center gap-1.5">
                <Search className="size-3" />
                {t("common.search", "Busca (NIP-50)")}
              </Label>
              <div className="relative">
                <Input
                  className="h-9 pr-8 bg-background/50 focus-visible:ring-primary/30"
                  onChange={(e) => updateFilter({ search: e.target.value || undefined })}
                  placeholder={t("eventSearch.queryPlaceholder", "Texto, hashtags, regex...")}
                  value={filter.search || ""}
                />
                {filter.search && (
                  <button 
                    onClick={() => updateFilter({ search: undefined })}
                    className="absolute right-2 top-1.5 p-1 text-muted-foreground hover:text-foreground transition-colors"
                  >
                    <X className="size-3" />
                  </button>
                )}
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label className="text-[10px] text-muted-foreground uppercase">{t("events.search.kind", "Kinas (Kinds)")}</Label>
                <div className="flex gap-1.5">
                  <Input
                    className="h-8 text-xs bg-background/50"
                    onChange={(e) => setInputKind(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && (addListItem("kinds", parseInt(inputKind)), setInputKind(""))}
                    placeholder="1, 0, 7..."
                    type="number"
                    value={inputKind}
                  />
                  <Button onClick={() => (addListItem("kinds", parseInt(inputKind)), setInputKind(""))} size="sm" variant="secondary" type="button" className="h-8 px-2">
                    <Plus className="size-3" />
                  </Button>
                </div>
                <div className="flex flex-wrap gap-1 mt-1.5 min-h-[1.5rem]">
                  {filter.kinds?.map((k) => (
                    <Badge key={k} variant="muted" className="gap-1 px-1.5 py-0.5 text-[9px] font-mono border-primary/10">
                      {k}
                      <X className="size-2.5 cursor-pointer hover:text-destructive" onClick={() => removeListItem("kinds", k)} />
                    </Badge>
                  ))}
                </div>
                <div className="flex gap-1 overflow-x-auto pb-1 mt-1 no-scrollbar opacity-70 hover:opacity-100 transition-opacity">
                  {[1, 0, 3, 7, 30023, 2003].map(k => (
                    <button key={k} onClick={() => quickAddKind(k)} className="text-[9px] bg-secondary/50 px-1.5 py-0.5 rounded-sm hover:bg-primary/20 hover:text-primary border border-primary/5 whitespace-nowrap">
                      {k}
                    </button>
                  ))}
                </div>
              </div>

              <div className="space-y-1.5">
                <Label className="text-[10px] text-muted-foreground uppercase">{t("users.search.label", "Autores")}</Label>
                <div className="flex gap-1.5">
                  <Input
                    className="h-8 text-xs bg-background/50 font-mono"
                    onChange={(e) => setInputAuthor(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && (addListItem("authors", inputAuthor.trim()), setInputAuthor(""))}
                    placeholder="hex pubkey..."
                    value={inputAuthor}
                  />
                  <Button onClick={() => (addListItem("authors", inputAuthor.trim()), setInputAuthor(""))} size="sm" variant="secondary" type="button" className="h-8 px-2">
                    <Plus className="size-3" />
                  </Button>
                </div>
                <div className="flex flex-wrap gap-1 mt-1.5">
                  {filter.authors?.map((a) => (
                    <Badge key={a} variant="muted" className="gap-1 px-1.5 py-0.5 text-[9px] font-mono border-primary/10">
                      {a.slice(0, 8)}...
                      <X className="size-2.5 cursor-pointer hover:text-destructive" onClick={() => removeListItem("authors", a)} />
                    </Badge>
                  ))}
                </div>
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label className="text-[10px] text-muted-foreground uppercase">{t("events.search.since", "Início")}</Label>
                <Input
                  className="h-8 text-[10px] bg-background/50"
                  onChange={(e) => handleDateChange("since", e.target.value)}
                  type="datetime-local"
                  value={formatUnixForInput(filter.since)}
                />
              </div>

              <div className="space-y-1.5">
                <Label className="text-[10px] text-muted-foreground uppercase">{t("events.search.until", "Fim")}</Label>
                <Input
                  className="h-8 text-[10px] bg-background/50"
                  onChange={(e) => handleDateChange("until", e.target.value)}
                  type="datetime-local"
                  value={formatUnixForInput(filter.until)}
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <Label className="text-[10px] text-muted-foreground uppercase">{t("common.limit", "Limite")}</Label>
              <Input
                className="h-8 text-xs bg-background/50"
                min={1}
                onChange={(e) => updateFilter({ limit: parseInt(e.target.value) || undefined })}
                type="number"
                value={filter.limit || ""}
              />
            </div>
          </TabsContent>

          {/* TAG FILTERS */}
          <TabsContent value="tags" className="space-y-4 pt-1 animate-in slide-in-from-left-2 duration-300">
            <div className="bg-secondary/10 rounded-lg p-3 space-y-3 border border-border/50">
              <div className="grid grid-cols-[80px_1fr_auto] gap-2 items-end">
                <div className="space-y-1">
                  <Label className="text-[9px] text-muted-foreground uppercase font-bold">Key</Label>
                  <div className="flex items-center h-8 px-2 bg-background/50 border rounded-md">
                    <span className="text-xs text-muted-foreground mr-1">#</span>
                    <Input
                      className="border-0 p-0 h-6 focus-visible:ring-0 text-xs font-mono bg-transparent"
                      maxLength={10}
                      onChange={(e) => setInputTagKey(e.target.value)}
                      placeholder="t, e, p..."
                      value={inputTagKey}
                    />
                  </div>
                </div>
                <div className="space-y-1">
                  <Label className="text-[9px] text-muted-foreground uppercase font-bold">Value</Label>
                  <Input
                    className="h-8 text-xs bg-background/50"
                    onChange={(e) => setInputTagValue(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && addTag()}
                    placeholder="valor da tag..."
                    value={inputTagValue}
                  />
                </div>
                <Button onClick={() => addTag()} size="sm" variant="default" type="button" className="h-8 w-8 p-0">
                  <Plus className="size-4" />
                </Button>
              </div>

              <div className="flex flex-wrap gap-1.5 mt-1 border-t border-border/30 pt-2">
                <TooltipProvider>
                  {["t", "e", "p", "d", "h", "q"].map(k => (
                    <Tooltip key={k}>
                      <TooltipTrigger asChild>
                        <button 
                          onClick={() => setInputTagKey(k)} 
                          className={`text-[9px] px-2 py-0.5 rounded-full border transition-all ${inputTagKey === k ? "bg-primary text-primary-foreground border-primary scale-105" : "bg-muted hover:bg-secondary border-border"}`}
                        >
                          #{k}
                        </button>
                      </TooltipTrigger>
                      <TooltipContent className="text-[10px] animate-in zoom-in-95 duration-100">
                        <p>{t(`nostrFilterBuilder.tagTooltips.${k}`)}</p>
                      </TooltipContent>
                    </Tooltip>
                  ))}
                </TooltipProvider>
              </div>
            </div>

            <div className="space-y-2">
              <Label className="text-[10px] text-muted-foreground uppercase font-bold">{t("common.activeTags", "Tags Selecionadas")}</Label>
              <div className="min-h-[80px] rounded-md border border-dashed border-primary/20 bg-primary/5 p-3 flex flex-wrap gap-2 items-start content-start">
                {tagFilters.length === 0 && (
                  <div className="w-full flex flex-col items-center justify-center py-2 opacity-30">
                    <Hash className="size-6 mb-1" />
                    <span className="text-[10px] uppercase font-bold">Vazio</span>
                  </div>
                )}
                {tagFilters.map(({ key, values }) => (
                  <div key={key} className="flex flex-wrap gap-1 p-1 rounded-md bg-background/50 border border-primary/10 max-w-full">
                    <div className="px-1.5 py-0.5 text-[9px] font-bold text-primary mr-0.5 bg-primary/10 rounded-sm">
                      {key}
                    </div>
                    {values.map(v => (
                      <Badge key={v} variant="success" className="gap-1 px-1.5 py-0.5 text-[10px] font-mono leading-none border-emerald-500/20">
                        <span className="max-w-[150px] truncate">{v}</span>
                        <X className="size-2.5 cursor-pointer hover:text-destructive" onClick={() => removeTag(key, v)} />
                      </Badge>
                    ))}
                  </div>
                ))}
              </div>
            </div>
          </TabsContent>

          {/* NIPs & EXTENSIONS */}
          <TabsContent value="nips" className="space-y-4 pt-1 animate-in slide-in-from-left-2 duration-300">
             <div className="grid gap-3">
               {/* General Metadata (NIP-24/52) */}
               <div className="p-3 rounded-lg border border-border/50 bg-secondary/5 space-y-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-1.5">
                      <MessageSquare className="size-3.5 text-pink-500" />
                      <span className="text-[11px] font-bold">Metadata Geral</span>
                    </div>
                  </div>
                  <div className="grid gap-2">
                    <div className="space-y-1">
                      <Label className="text-[9px] uppercase text-muted-foreground font-bold">Título (#title)</Label>
                      <Input 
                        className="h-8 text-xs bg-background/50" 
                        placeholder="Pesquisar por título..."
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            handleNIPInput("title", (e.currentTarget as HTMLInputElement).value)
                            e.currentTarget.value = ""
                          }
                        }}
                      />
                    </div>
                    <div className="space-y-1">
                      <Label className="text-[9px] uppercase text-muted-foreground font-bold">Localização (#location)</Label>
                      <Input 
                        className="h-8 text-xs bg-background/50" 
                        placeholder="Pesquisar por local..."
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            handleNIPInput("location", (e.currentTarget as HTMLInputElement).value)
                            e.currentTarget.value = ""
                          }
                        }}
                      />
                    </div>
                  </div>
               </div>

               {/* Git & Groups (NIP-34 / NIP-29) */}
               <div className="grid grid-cols-2 gap-3">
                  <div className="p-3 rounded-lg border border-border/50 bg-blue-500/5 space-y-2">
                    <div className="flex items-center gap-1.5 pb-1 border-b border-blue-500/10">
                      <GitBranch className="size-3.5 text-blue-500" />
                      <span className="text-[10px] font-bold uppercase">Git</span>
                    </div>
                    <Input 
                      className="h-7 text-[10px] bg-background/30" 
                      placeholder="#name..."
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          handleNIPInput("name", (e.currentTarget as HTMLInputElement).value)
                          e.currentTarget.value = ""
                        }
                      }}
                    />
                    <Input 
                      className="h-7 text-[10px] bg-background/30 font-mono" 
                      placeholder="#commit..."
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          handleNIPInput("c", (e.currentTarget as HTMLInputElement).value)
                          e.currentTarget.value = ""
                        }
                      }}
                    />
                  </div>

                  <div className="p-3 rounded-lg border border-border/50 bg-orange-500/5 space-y-2">
                    <div className="flex items-center gap-1.5 pb-1 border-b border-orange-500/10">
                      <Users className="size-3.5 text-orange-500" />
                      <span className="text-[10px] font-bold uppercase">Groups</span>
                    </div>
                    <Input 
                      className="h-7 text-[10px] bg-background/30 font-mono" 
                      placeholder="Group ID (#h)..."
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          handleNIPInput("h", (e.currentTarget as HTMLInputElement).value)
                          e.currentTarget.value = ""
                        }
                      }}
                    />
                  </div>
               </div>

               {/* Indexers & Torrent (NIP-35/73) */}
               <div className="p-3 rounded-lg border border-border/50 bg-emerald-500/5 space-y-3">
                  <div className="flex items-center gap-1.5 pb-1 border-b border-emerald-500/10">
                    <Code className="size-3.5 text-emerald-500" />
                    <span className="text-[11px] font-bold">Indexadores & Torrents</span>
                  </div>
                  <div className="grid gap-2 sm:grid-cols-2">
                    <div className="space-y-1">
                      <Label className="text-[9px] uppercase text-muted-foreground font-bold">Ext. Identity (#i)</Label>
                      <Input 
                        className="h-8 text-xs bg-background/30" 
                        placeholder="platform:id..."
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            handleNIPInput("i", (e.currentTarget as HTMLInputElement).value)
                            e.currentTarget.value = ""
                          }
                        }}
                      />
                    </div>
                    <div className="space-y-1">
                      <Label className="text-[9px] uppercase text-muted-foreground font-bold">Info Hash (#x)</Label>
                      <Input 
                        className="h-8 text-xs bg-background/30 font-mono" 
                        placeholder="btih:hash..."
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            handleNIPInput("x", (e.currentTarget as HTMLInputElement).value)
                            e.currentTarget.value = ""
                          }
                        }}
                      />
                    </div>
                  </div>
               </div>
             </div>
          </TabsContent>
        </Tabs>

        {/* Footer actions */}
        <div className="mt-6 pt-4 border-t border-border/30 flex items-center justify-between gap-4">
           <div className="text-[9px] text-muted-foreground/60 font-mono truncate bg-secondary/10 px-2 py-1 rounded max-w-[60%]">
             {JSON.stringify(filter)}
           </div>
           <Button 
             variant="ghost" 
             size="sm" 
             className="h-7 text-[10px] px-2 text-muted-foreground hover:text-destructive transition-colors"
             onClick={() => {
               const reset = {}
               setFilter(reset)
               onChange(reset)
             }}
           >
             {t("common.reset", "Limpar")}
           </Button>
        </div>
      </CardContent>
    </Card>
  )
}
