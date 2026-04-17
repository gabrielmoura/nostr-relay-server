import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { BanUserDialog } from "@/components/features/ban-user-dialog"
import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { UserAvatarChip } from "@/components/shared/user-avatar-chip"
import { VirtualizedList } from "@/components/shared/virtualized-list"
import { Badge } from "@/components/ui/badge"
import { useInfiniteLoggedUsers } from "@/hooks/use-admin-data"

export function LoggedUsersPage() {
  const { t } = useTranslation()
  const query = useInfiniteLoggedUsers()
  const pages = query.data?.pages ?? []
  const users = pages.flatMap((page) => page.items)
  const total = pages[0]?.total ?? 0

  if (query.isLoading && users.length === 0) {
    return <LoadingPanel label={t("loggedUsers.loading")} />
  }

  if (query.isError) {
    return <ErrorPanel description={t("loggedUsers.errorDescription")} onRetry={() => void query.refetch()} title={t("loggedUsers.errorTitle")} />
  }
  return (
    <div className="space-y-6">
      <PageHeader description={t("loggedUsers.description")} title={t("loggedUsers.title")} />

      {users.length === 0 ? (
        <EmptyPanel description={t("loggedUsers.emptyDescription")} title={t("loggedUsers.emptyTitle")} />
      ) : (
        <VirtualizedList
          estimateSize={140}
          fetchMore={() => void query.fetchNextPage()}
          hasMore={query.hasNextPage}
          isFetchingMore={query.isFetchingNextPage}
          items={users}
          renderItem={(user) => (
            <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-card p-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <UserAvatarChip subtitle={t("overview.connectionsCount", { count: user.connectionCount })} user={user} />
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant={user.connectionState === "stable" ? "success" : "warning"}>{user.connectionState === "stable" ? t("loggedUsers.stable") : t("loggedUsers.warning")}</Badge>
                  {user.lastSeenAt ? <Badge variant="muted">{t("loggedUsers.lastSeen")} {new Date(user.lastSeenAt).toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit", timeZone: "UTC" })} UTC</Badge> : null}
                </div>
              </div>
              <div className="mt-4 flex flex-wrap gap-2">
                <Link className="inline-flex items-center rounded-[calc(var(--radius)-0.3rem)] border border-border px-3 py-2 text-sm hover:bg-muted" params={{ pubkey: user.pubkey }} to="/users/$pubkey">
                  {t("overview.viewDetails")}
                </Link>
                <Link className="inline-flex items-center rounded-[calc(var(--radius)-0.3rem)] border border-border px-3 py-2 text-sm hover:bg-muted" search={{ q: user.pubkey }} to="/events/search">
                  {t("overview.searchEvents")}
                </Link>
                <BanUserDialog defaultPubkey={user.pubkey} defaultReason="moderacao manual" triggerLabel={t("moderation.ban.shortTrigger")} triggerVariant="warning" />
              </div>
            </div>
          )}
          total={total}
        />
      )}
    </div>
  )
}
