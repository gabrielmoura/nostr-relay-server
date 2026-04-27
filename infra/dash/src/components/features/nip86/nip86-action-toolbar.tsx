import type { ReactNode } from "react"

import { Search } from "lucide-react"

import { Input } from "@/components/ui/input"

type NIP86ActionToolbarProps = {
  placeholder: string
  value: string
  onChange: (value: string) => void
  action?: ReactNode
}

export function NIP86ActionToolbar({ placeholder, value, onChange, action }: NIP86ActionToolbarProps) {
  return (
    <div className="flex flex-col gap-3 border-b border-border/80 pb-4 md:flex-row md:items-center md:justify-between">
      <div className="relative max-w-xl flex-1">
        <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input className="pl-9" onChange={(event) => onChange(event.target.value)} placeholder={placeholder} value={value} />
      </div>
      {action ? <div className="flex shrink-0 items-center gap-2">{action}</div> : null}
    </div>
  )
}
