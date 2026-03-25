import { Link } from "@tanstack/react-router"

import { Avatar } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import type { UserProfile } from "@/types/admin"
import { cn, shortenId } from "@/lib/utils"

export function UserAvatarChip({ user, subtitle, compact = false }: { user: UserProfile; subtitle?: string; compact?: boolean }) {
  const displayName = user.displayName || user.handle || user.pubkey
  const handle = user.handle || "@desconhecido"
  const npub = user.npub || user.pubkey

  return (
    <div className={cn("flex items-center gap-3", compact && "gap-2")}>
      <Avatar className={compact ? "size-9" : undefined} name={displayName} src={user.picture} />
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <Link className="truncate font-medium text-foreground hover:text-primary" params={{ pubkey: user.pubkey }} to="/users/$pubkey">
            {displayName}
          </Link>
          {user.status === "banned" ? <Badge variant="danger">banido</Badge> : null}
          {user.status === "suspect" ? <Badge variant="warning">suspeito</Badge> : null}
          {user.status === "online" ? <Badge variant="success">online</Badge> : null}
        </div>
        <p className="truncate text-xs text-muted-foreground">
          {handle} · {shortenId(npub, 10, 4)}
          {subtitle ? ` · ${subtitle}` : ""}
        </p>
      </div>
    </div>
  )
}
