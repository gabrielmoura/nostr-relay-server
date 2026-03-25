import { cn, initials } from "@/lib/utils"

type AvatarProps = {
  name: string
  src?: string
  className?: string
}

export function Avatar({ name, src, className }: AvatarProps) {
  if (src) {
    return <img alt={name} className={cn("size-10 rounded-full border border-border object-cover", className)} src={src} />
  }

  return (
    <div className={cn("flex size-10 items-center justify-center rounded-full border border-border bg-muted font-heading text-sm font-semibold text-foreground", className)}>
      {initials(name)}
    </div>
  )
}
