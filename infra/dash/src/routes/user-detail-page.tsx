import { Link } from "@tanstack/react-router"
import { useParams } from "@tanstack/react-router"
import { Copy } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { BanUserDialog } from "@/components/features/ban-user-dialog"
import { NIP05AssociateDialog } from "@/components/features/nip05-associate-dialog"
import { UnbanUserAlert } from "@/components/features/unban-user-alert"
import { PageHeader } from "@/components/shared/page-header"
import { ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { UserAvatarChip } from "@/components/shared/user-avatar-chip"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useBanStatus, useUser, useUserNIP05 } from "@/hooks/use-admin-data"
import { shortenId } from "@/lib/utils"

export function UserDetailPage() {
  const { t } = useTranslation()
  const { pubkey } = useParams({ from: "/users/$pubkey" })
  const userQuery = useUser(pubkey)
  const banStatus = useBanStatus(pubkey)
  const userNIP05Query = useUserNIP05(pubkey)

  if (userQuery.isLoading) {
    return <LoadingPanel label={t("userDetail.loading")} />
  }

  if (userQuery.isError || !userQuery.data) {
    return <ErrorPanel description={t("userDetail.errorDescription")} onRetry={() => void userQuery.refetch()} title={t("userDetail.errorTitle")} />
  }

  const user = userQuery.data
  const currentAssociation = userNIP05Query.data

  const defaultNIP05Name = buildDefaultNIP05Name(user)

  return (
    <div className="space-y-6">
      <PageHeader
        actions={
          <>
            <Button asChild variant="outline">
              <Link search={{ authors: pubkey }} to="/events/search">{t("userDetail.monitor")}</Link>
            </Button>
            <Button
              onClick={async () => {
                await navigator.clipboard.writeText(user.npub)
                toast.success(t("userDetail.npubCopied"))
              }}
              variant="outline"
            >
              <Copy className="size-4" />
              {t("userDetail.copyNpub")}
            </Button>
            <NIP05AssociateDialog defaultName={defaultNIP05Name} pubkey={pubkey} />
            {banStatus.data?.banned ? <UnbanUserAlert pubkey={pubkey} /> : <BanUserDialog defaultPubkey={pubkey} triggerLabel={t("moderation.ban.trigger")} triggerVariant="warning" />}
          </>
        }
        description={t("userDetail.description")}
        title={t("userDetail.title")}
      />

      <div className="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
        <Card>
          <CardContent className="space-y-4 p-5">
            <UserAvatarChip subtitle={user.nip05 ?? user.metadata} user={user} />
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Pubkey</p>
                <p className="mt-2 break-all font-mono text-sm text-foreground">{pubkey}</p>
              </div>
              <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">NPUB</p>
                <p className="mt-2 break-all font-mono text-sm text-foreground">{user.npub || shortenId(pubkey, 18, 4)}</p>
              </div>
              <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{t("userDetail.trust")}</p>
                <p className="mt-2 text-sm text-foreground">{user.trustScore != null ? user.trustScore.toFixed(2) : t("userDetail.notAvailable")}</p>
              </div>
              <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Metadata</p>
                <p className="mt-2 text-sm text-foreground">{user.metadata ?? t("userDetail.noMetadata")}</p>
              </div>
              <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border p-3 sm:col-span-2">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{t("userDetail.associatedNip05")}</p>
                <p className="mt-2 text-sm text-foreground">
                  {currentAssociation?.exists ? currentAssociation.name : t("userDetail.noManualAssociation")}
                </p>
                {currentAssociation?.relay_hints?.length ? (
                  <p className="mt-1 text-xs text-muted-foreground">{t("common.relayHints")}: {currentAssociation.relay_hints.join(", ")}</p>
                ) : null}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("userDetail.moderationStatus")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-wrap gap-2">
              <Badge variant={banStatus.data?.banned ? "danger" : "success"}>{banStatus.data?.banned ? t("userDetail.banned") : t("userDetail.allowed")}</Badge>
              {user.status ? <Badge variant="muted">{user.status}</Badge> : null}
            </div>
            <div className="rounded-[calc(var(--radius)-0.2rem)] border border-border p-3 text-sm text-muted-foreground">
              <p className="font-medium text-foreground">{t("userDetail.registeredReason")}</p>
                <p className="mt-1">{banStatus.data?.reason ?? user.reason ?? t("userDetail.noReason")}</p>
              </div>
            </CardContent>
          </Card>
      </div>
    </div>
  )
}

function buildDefaultNIP05Name(user: { handle?: string; displayName?: string; nip05?: string }) {
  const fromNIP05 = user.nip05?.split("@")[0]?.trim().toLowerCase() ?? ""
  if (isValidNIP05Name(fromNIP05)) {
    return fromNIP05
  }

  const fromHandle = user.handle?.replace(/^@/, "").trim().toLowerCase() ?? ""
  if (isValidNIP05Name(fromHandle)) {
    return fromHandle
  }

  const fromDisplayName = (user.displayName ?? "")
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "_")
    .replace(/[^a-z0-9._-]/g, "")
  if (isValidNIP05Name(fromDisplayName)) {
    return fromDisplayName
  }

  return ""
}

function isValidNIP05Name(value: string) {
  return /^[a-z0-9._-]+$/.test(value)
}
