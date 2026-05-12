import { Mail, Lock, Server } from "lucide-react"
import { useTranslation } from "react-i18next"

interface DMRelayListCardProps {
  relays: string[]
}

export function DMRelayListCard({ relays }: DMRelayListCardProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-4 rounded-md border border-purple-500/20 bg-purple-500/5 p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Mail className="size-5 text-purple-500" />
          <h3 className="font-semibold text-foreground/90">DM Relays (NIP-17)</h3>
        </div>
        <Lock className="size-4 text-muted-foreground opacity-60" />
      </div>

      {relays.length === 0 ? (
        <p className="text-sm text-muted-foreground italic">Nenhum relay especificado.</p>
      ) : (
        <ul className="grid gap-2 sm:grid-cols-2">
          {relays.map((relay) => (
            <li 
              key={relay} 
              className="flex items-center gap-2 rounded-md border border-border/60 bg-background/50 px-3 py-2 text-sm text-foreground/80"
            >
              <Server className="size-3.5 text-purple-500/70" />
              <span className="truncate font-mono">{relay}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
