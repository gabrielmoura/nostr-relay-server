import type { ComponentType, FormEvent } from "react"
import { useMemo, useState } from "react"
import { Link, Outlet, useLocation, useNavigate } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"
import { AtSign, Bell, Cable, Download, LayoutDashboard, Menu, Network, Radio, RefreshCw, Search, ShieldAlert, ShieldCheck, Tags, TriangleAlert, UserRound, Users } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet"
import { cn } from "@/lib/utils"

type NavItem = {
  to: string
  labelKey: string
  icon: ComponentType<{ className?: string }>
}

const primaryNav: NavItem[] = [
  { to: "/", labelKey: "layout.nav.overview", icon: LayoutDashboard },
  { to: "/sync", labelKey: "layout.nav.sync", icon: RefreshCw },
  { to: "/download", labelKey: "layout.nav.download", icon: Download },
  { to: "/groups", labelKey: "layout.nav.groups", icon: Users },
  { to: "/wot", labelKey: "layout.nav.wot", icon: Network },
  { to: "/users/logged", labelKey: "layout.nav.usersLogged", icon: Users },
  { to: "/users/banned", labelKey: "layout.nav.usersBanned", icon: ShieldAlert },
  { to: "/connections/active", labelKey: "layout.nav.connectionsActive", icon: Cable },
  { to: "/connections/logged", labelKey: "layout.nav.connectionsLogged", icon: UserRound },
  { to: "/events/search", labelKey: "layout.nav.eventsSearch", icon: Search },
  { to: "/labels", labelKey: "layout.nav.labels", icon: Tags },
  { to: "/nip05", labelKey: "layout.nav.nip05", icon: AtSign },
  { to: "/nip86", labelKey: "layout.nav.nip86", icon: ShieldCheck },
  { to: "/events/reported", labelKey: "layout.nav.eventsReported", icon: TriangleAlert },
  { to: "/stream", labelKey: "layout.nav.streams", icon: Radio },
]

function SideNav({ onNavigate }: { onNavigate?: () => void }) {
  const location = useLocation()
  const { t } = useTranslation()

  return (
    <div className="flex h-full flex-col gap-6">
      <div className="space-y-1 rounded-[var(--radius)] border border-primary/15 bg-[linear-gradient(135deg,rgba(14,165,233,0.08),rgba(255,255,255,0.98))] p-5 panel-shadow">
        <p className="font-heading text-lg font-semibold text-foreground">{t("layout.brandTitle")}</p>
        <p className="text-sm text-muted-foreground">{t("layout.brandDescription")}</p>
      </div>

      <nav className="space-y-2 rounded-[var(--radius)] border border-border bg-card/95 p-3 panel-shadow">
        {primaryNav.map((item) => {
          const active = location.pathname === item.to
          const Icon = item.icon

          return (
            <Link
              key={item.to}
              activeOptions={{ exact: item.to === "/" }}
              className={cn(
                "flex cursor-pointer items-center gap-3 rounded-[calc(var(--radius)-0.2rem)] px-3 py-2 text-sm transition-all duration-200",
                active
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground",
              )}
              onClick={onNavigate}
              to={item.to}
            >
              <Icon className="size-4" />
              <span>{t(item.labelKey)}</span>
            </Link>
          )
        })}
        <Link
          className={cn(
            "flex cursor-pointer items-center gap-3 rounded-[calc(var(--radius)-0.2rem)] px-3 py-2 text-sm transition-all duration-200",
            location.pathname === "/users/search"
              ? "bg-primary text-primary-foreground"
              : "text-muted-foreground hover:bg-muted hover:text-foreground",
          )}
          onClick={onNavigate}
          to="/users/search"
        >
          <Search className="size-4" />
          <span>{t("layout.nav.usersSearch")}</span>
        </Link>
      </nav>

      <div className="rounded-[var(--radius)] border border-sky-200 bg-sky-50 p-4 text-sm text-sky-900 panel-shadow">
        <div className="mb-1 flex items-center gap-2 font-heading font-semibold">
          <Bell className="size-4" />
          {t("layout.operationAlert")}
        </div>
        <p>{t("layout.operationAlertMessage")}</p>
      </div>
    </div>
  )
}

export function AppShell() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [globalSearch, setGlobalSearch] = useState("")
  const [mobileOpen, setMobileOpen] = useState(false)

  const searchHint = useMemo(() => {
    const value = globalSearch.trim().toLowerCase()
    if (!value) {
      return t("layout.hintNpubEventIp")
    }

    return value.startsWith("npub") || value.startsWith("@") ? t("layout.hintProfile") : t("layout.hintEvents")
  }, [globalSearch, t])

  const handleGlobalSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const query = globalSearch.trim()
    if (!query) {
      return
    }

    const target = query.startsWith("npub") || query.startsWith("@") ? "/users/search" : "/events/search"
    navigate({ to: target, search: { q: query } as never })
  }

  return (
    <div className="surface-grid min-h-dvh">
      <div className="mx-auto flex min-h-dvh w-full max-w-[1600px] gap-6 px-4 py-4 lg:px-6">
        <aside className="hidden w-80 shrink-0 lg:block">
          <div className="sticky top-4 h-[calc(100dvh-2rem)]">
            <SideNav />
          </div>
        </aside>

        <div className="flex min-w-0 flex-1 flex-col gap-4">
          <header className="sticky top-4 z-30 rounded-[var(--radius)] border border-border bg-card/95 px-4 py-3 backdrop-blur panel-shadow">
            <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
              <div className="flex items-center gap-3">
                <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
                  <SheetTrigger asChild>
                    <Button className="lg:hidden" size="icon" variant="outline">
                      <Menu className="size-4" />
                      <span className="sr-only">{t("layout.openNavigation")}</span>
                    </Button>
                  </SheetTrigger>
                  <SheetContent className="p-4" side="left">
                    <SideNav onNavigate={() => setMobileOpen(false)} />
                  </SheetContent>
                </Sheet>

                <div>
                  <p className="font-heading text-xl font-semibold text-foreground">{t("layout.adminPanel")}</p>
                  <p className="text-sm text-muted-foreground">{t("layout.adminPanelDescription")}</p>
                </div>
              </div>

              <form className="flex w-full max-w-xl flex-wrap items-center gap-2" onSubmit={handleGlobalSearch}>
                <div className="relative flex-1">
                  <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    aria-label={t("layout.globalSearchAria")}
                    className="pl-9"
                    onChange={(event) => setGlobalSearch(event.target.value)}
                    placeholder={t("layout.globalSearchPlaceholder", { hint: searchHint })}
                    value={globalSearch}
                  />
                </div>
                <Button type="submit">{t("common.search")}</Button>
                <div className="ml-auto flex items-center gap-1">
                  <span className="text-xs text-muted-foreground">{t("common.language")}</span>
                  <Button onClick={() => void i18n.changeLanguage("en")} size="sm" type="button" variant={i18n.resolvedLanguage?.startsWith("en") ? "default" : "outline"}>
                    EN
                  </Button>
                  <Button onClick={() => void i18n.changeLanguage("pt-BR")} size="sm" type="button" variant={i18n.resolvedLanguage?.startsWith("pt") ? "default" : "outline"}>
                    PT
                  </Button>
                </div>
              </form>
            </div>
          </header>

          <main className="min-w-0 flex-1">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  )
}
