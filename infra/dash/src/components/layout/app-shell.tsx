import type { ComponentType, FormEvent } from "react"
import { useMemo, useState } from "react"
import { Link, Outlet, useLocation, useNavigate } from "@tanstack/react-router"
import { Bell, Cable, LayoutDashboard, Menu, Radio, Search, ShieldAlert, TriangleAlert, UserRound, Users } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet"
import { cn } from "@/lib/utils"

type NavItem = {
  to: string
  label: string
  icon: ComponentType<{ className?: string }>
}

const primaryNav: NavItem[] = [
  { to: "/", label: "Overview", icon: LayoutDashboard },
  { to: "/users/logged", label: "Usuarios logados", icon: Users },
  { to: "/users/banned", label: "Usuarios banidos", icon: ShieldAlert },
  { to: "/connections/active", label: "Conexoes ativas", icon: Cable },
  { to: "/connections/logged", label: "Conexoes logadas", icon: UserRound },
  { to: "/events/search", label: "Busca de eventos", icon: Search },
  { to: "/events/reported", label: "Eventos reportados", icon: TriangleAlert },
  { to: "/stream", label: "Streams", icon: Radio },
]

function SideNav({ onNavigate }: { onNavigate?: () => void }) {
  const location = useLocation()

  return (
    <div className="flex h-full flex-col gap-6">
      <div className="space-y-1 rounded-[var(--radius)] border border-border bg-card p-5 panel-shadow">
        <p className="font-heading text-lg font-semibold text-foreground">Relay Nostr Admin</p>
        <p className="text-sm text-muted-foreground">Observabilidade, moderacao e operacao em tempo real</p>
      </div>

      <nav className="space-y-2 rounded-[var(--radius)] border border-border bg-card p-3 panel-shadow">
        {primaryNav.map((item) => {
          const active = location.pathname === item.to
          const Icon = item.icon

          return (
            <Link
              key={item.to}
              activeOptions={{ exact: item.to === "/" }}
              className={cn(
                "flex items-center gap-3 rounded-[calc(var(--radius)-0.2rem)] px-3 py-2 text-sm transition-all duration-200",
                active
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground",
              )}
              onClick={onNavigate}
              to={item.to}
            >
              <Icon className="size-4" />
              <span>{item.label}</span>
            </Link>
          )
        })}
        <Link
          className={cn(
            "flex items-center gap-3 rounded-[calc(var(--radius)-0.2rem)] px-3 py-2 text-sm transition-all duration-200",
            location.pathname === "/users/search"
              ? "bg-primary text-primary-foreground"
              : "text-muted-foreground hover:bg-muted hover:text-foreground",
          )}
          onClick={onNavigate}
          to="/users/search"
        >
          <Search className="size-4" />
          <span>Busca de usuarios</span>
        </Link>
      </nav>

      <div className="rounded-[var(--radius)] border border-orange-200 bg-orange-50 p-4 text-sm text-orange-800 panel-shadow">
        <div className="mb-1 flex items-center gap-2 font-heading font-semibold">
          <Bell className="size-4" />
          Alerta de operacao
        </div>
        <p>Erro ao atualizar presenca. Reconectando stream de sessao em segundo plano.</p>
      </div>
    </div>
  )
}

export function AppShell() {
  const navigate = useNavigate()
  const [globalSearch, setGlobalSearch] = useState("")
  const [mobileOpen, setMobileOpen] = useState(false)

  const searchHint = useMemo(() => {
    const value = globalSearch.trim().toLowerCase()
    if (!value) {
      return "npub, evento, IP"
    }

    return value.startsWith("npub") || value.startsWith("@") ? "perfil" : "eventos"
  }, [globalSearch])

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
                      <span className="sr-only">Abrir navegacao</span>
                    </Button>
                  </SheetTrigger>
                  <SheetContent className="p-4" side="left">
                    <SideNav onNavigate={() => setMobileOpen(false)} />
                  </SheetContent>
                </Sheet>

                <div>
                  <p className="font-heading text-xl font-semibold text-foreground">Painel administrativo</p>
                  <p className="text-sm text-muted-foreground">Controle operacional do relay com rotas modulares e estados reais.</p>
                </div>
              </div>

              <form className="flex w-full max-w-xl items-center gap-2" onSubmit={handleGlobalSearch}>
                <div className="relative flex-1">
                  <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    aria-label="Busca global"
                    className="pl-9"
                    onChange={(event) => setGlobalSearch(event.target.value)}
                    placeholder={`Buscar global: ${searchHint}`}
                    value={globalSearch}
                  />
                </div>
                <Button type="submit">Buscar</Button>
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
