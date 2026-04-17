import { useState } from "react"
import { useTranslation } from "react-i18next"

import { PageHeader } from "@/components/shared/page-header"
import { EmptyPanel, ErrorPanel, LoadingPanel } from "@/components/shared/state-panels"
import { UserAvatarChip } from "@/components/shared/user-avatar-chip"
import { VirtualizedList } from "@/components/shared/virtualized-list"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { useInfiniteBannedUsers } from "@/hooks/use-admin-data"
import { formatDateTime } from "@/lib/utils"
import { UnbanUserAlert } from "@/components/features/unban-user-alert"

export function BannedUsersPage() {
  const { t } = useTranslation()
  const [search, setSearch] = useState("")
  const query = useInfiniteBannedUsers(search)
  const pages = query.data?.pages ?? []
  const users = pages.flatMap((page) => page.items)
  const total = pages[0]?.total ?? 0

  if (query.isLoading && users.length === 0) {
    return <LoadingPanel label={t("bannedUsers.loading")} />
  }

  if (query.isError) {
    return <ErrorPanel description={t("bannedUsers.errorDescription")} onRetry={() => void query.refetch()} title={t("bannedUsers.errorTitle")} />
  }
  return (
    <div className="space-y-6">
      <PageHeader description={t("bannedUsers.description")} title={t("bannedUsers.title")} />

      <Input onChange={(event) => setSearch(event.target.value)} placeholder={t("bannedUsers.searchPlaceholder")} value={search} />

      {users.length === 0 ? (
        <EmptyPanel description={t("bannedUsers.emptyDescription")} title={t("bannedUsers.emptyTitle")} />
      ) : (
        <VirtualizedList
          estimateSize={122}
          fetchMore={() => void query.fetchNextPage()}
          hasMore={query.hasNextPage}
          isFetchingMore={query.isFetchingNextPage}
          items={users}
          renderItem={(user) => (
            <div className="rounded-[calc(var(--radius)-0.25rem)] border border-border bg-card p-4">
              <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                <div className="space-y-3">
                  <UserAvatarChip subtitle={user.reason} user={user} />
                  <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                    <Badge variant="danger">{user.reason}</Badge>
                    {user.created_at ? <Badge variant="muted">{formatDateTime(user.created_at)}</Badge> : null}
                  </div>
                </div>
                <UnbanUserAlert pubkey={user.pubkey} />
              </div>
            </div>
          )}
          total={total}
        />
      )}
    </div>
  )
}
