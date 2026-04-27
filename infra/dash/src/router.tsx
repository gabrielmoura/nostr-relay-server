import { createRootRoute, createRoute, createRouter } from "@tanstack/react-router"

import { AppShell } from "@/components/layout/app-shell"
import { ActiveConnectionsPage } from "@/routes/active-connections-page"
import { BannedUsersPage } from "@/routes/banned-users-page"
import { EventSearchPage } from "@/routes/event-search-page"
import { EventDetailPage } from "@/routes/event-detail-page"
import { LoggedConnectionsPage } from "@/routes/logged-connections-page"
import { LoggedUsersPage } from "@/routes/logged-users-page"
import { OverviewPage } from "@/routes/overview-page"
import { ReportedEventsPage } from "@/routes/reported-events-page"
import { StreamStatusPage } from "@/routes/stream-status-page"
import { UserDetailPage } from "@/routes/user-detail-page"
import { NIP05Page } from "@/routes/nip05-page"
import { NIP86Page } from "@/routes/nip86-page"
import { UserSearchPage } from "@/routes/user-search-page"

const rootRoute = createRootRoute({
  component: AppShell,
})

const overviewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: OverviewPage,
})

const loggedUsersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/users/logged",
  component: LoggedUsersPage,
})

const bannedUsersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/users/banned",
  component: BannedUsersPage,
})

const userSearchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/users/search",
  component: UserSearchPage,
})

const nip05Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/nip05",
  component: NIP05Page,
})

const nip86Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/nip86",
  component: NIP86Page,
})

const userDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/users/$pubkey",
  component: UserDetailPage,
})

const activeConnectionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/connections/active",
  component: ActiveConnectionsPage,
})

const loggedConnectionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/connections/logged",
  component: LoggedConnectionsPage,
})

const eventSearchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/events/search",
  component: EventSearchPage,
})

const eventDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/events/$eventId",
  component: EventDetailPage,
})

const reportedEventsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/events/reported",
  component: ReportedEventsPage,
})

const streamStatusRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/stream",
  component: StreamStatusPage,
})

const routeTree = rootRoute.addChildren([
  overviewRoute,
  loggedUsersRoute,
  bannedUsersRoute,
  userSearchRoute,
  nip05Route,
  nip86Route,
  userDetailRoute,
  activeConnectionsRoute,
  loggedConnectionsRoute,
  eventSearchRoute,
  reportedEventsRoute,
  streamStatusRoute,
  eventDetailRoute,
])

const normalizedBasePath = import.meta.env.BASE_URL === "/" ? "/" : import.meta.env.BASE_URL.replace(/\/$/, "")

export const router = createRouter({ routeTree, basepath: normalizedBasePath })

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router
  }
}
