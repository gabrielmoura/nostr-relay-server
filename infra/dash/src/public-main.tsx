import { useEffect, useState } from "react"
import ReactDOM from "react-dom/client"
import { Copy, Database, Globe2, Layers3, Radio, RefreshCw, Server, SquareTerminal } from "lucide-react"

import { DMImportPanel } from "@/features/dm-import/dm-import-panel"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { getRelayInformation, relayWebSocketURL, type RelayInformation } from "@/services/public-relay"
import "@/index.css"

function PublicRelayPage() {
  const [information, setInformation] = useState<RelayInformation>()
  const [error, setError] = useState("")
  const [copied, setCopied] = useState(false)
  const relayURL = relayWebSocketURL(window.location)

  const reload = () => {
    setError("")
    void getRelayInformation()
      .then(setInformation)
      .catch((cause: unknown) => setError(cause instanceof Error ? cause.message : "Falha ao carregar NIP-11"))
  }

  useEffect(reload, [])

  const copy = async () => {
    await navigator.clipboard?.writeText(relayURL)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1800)
  }

  const nips = information?.supported_nips?.join(", ") || "Carregando capacidades…"
  const endpoints = information?.public_endpoints ?? []

  return (
    <main className="public-relay min-h-dvh overflow-hidden">
      <div aria-hidden="true" className="public-relay__grid" />
      <div aria-hidden="true" className="public-relay__orb public-relay__orb--violet" />
      <div aria-hidden="true" className="public-relay__orb public-relay__orb--cyan" />

      <header className="public-relay__header">
        <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-5 py-4 sm:px-8 lg:px-10">
          <a className="group flex items-center gap-3" href="/" aria-label="NRServer">
            <span className="public-relay__mark"><Radio className="size-5" /><i /></span>
            <span>
              <span className="flex items-center gap-2 font-heading font-semibold tracking-tight text-white">
                NRServer <small>Nostr Relay</small>
              </span>
              <span className="hidden text-xs text-slate-400 sm:block">Infraestrutura descentralizada, simples de operar.</span>
            </span>
          </a>
          <span className="public-relay__online"><i /> relay online</span>
        </div>
      </header>

      <section className="relative mx-auto grid max-w-7xl items-center gap-12 px-5 pb-20 pt-16 sm:px-8 md:pt-20 lg:grid-cols-[1.08fr_.92fr] lg:px-10 lg:pb-24 lg:pt-28">
        <div>
          <span className="public-relay__eyebrow"><i /> interface pública disponível</span>
          <h1 className="mt-6 max-w-4xl font-heading text-5xl font-semibold leading-[1.02] tracking-[-.055em] text-white sm:text-6xl lg:text-7xl">
            Seu ponto de acesso à <span>rede Nostr.</span>
          </h1>
          <p className="mt-7 max-w-2xl text-lg leading-8 text-slate-300 sm:text-xl">
            {information?.description || "Conecte-se ao relay, consulte suas capacidades em tempo real e importe seus eventos diretamente pelo navegador."}
          </p>
          <div className="mt-9 flex flex-wrap gap-3 text-sm text-slate-300">
            <span className="public-relay__pill"><Globe2 className="size-4" /> NIP-11 ao vivo</span>
            <span className="public-relay__pill"><Server className="size-4" /> WebSocket Nostr</span>
            <span className="public-relay__pill"><Database className="size-4" /> Eventos no seu relay</span>
          </div>
        </div>

        <div className="public-relay__terminal-wrap">
          <div className="public-relay__terminal">
            <div className="flex items-center justify-between border-b border-white/10 px-5 py-4">
              <span className="flex gap-2"><i className="bg-rose-400" /><i className="bg-amber-300" /><i className="bg-emerald-400" /></span>
              <span className="font-mono text-[11px] uppercase tracking-[.2em] text-slate-400">NRServer / relay</span>
              <SquareTerminal className="size-4 text-slate-400" />
            </div>
            <div className="p-5 sm:p-7">
              <div className="public-relay__code">
                <p><b>relay</b> websocket ready</p>
                <p><b>nip-11</b> {information ? "capabilities loaded" : "loading capabilities"}</p>
                <p><b>events</b> import available in browser</p>
                <div />
                <p className="flex items-center justify-between gap-3"><span className="text-slate-400">status</span><strong><i /> operational</strong></p>
              </div>
              <div className="mt-5 rounded-2xl border border-cyan-300/15 bg-cyan-300/[.055] p-4">
                <p className="mb-2 font-mono text-[11px] uppercase tracking-[.18em] text-cyan-200">WebSocket URL</p>
                <code className="block break-all text-sm text-cyan-50">{relayURL}</code>
                <Button className="mt-4 w-full border-violet-300/30 bg-violet-500/20 text-violet-100 hover:bg-violet-500/35" onClick={() => void copy()} size="sm" type="button" variant="outline">
                  <Copy /> {copied ? "Copiado" : "Copiar endereço"}
                </Button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="public-relay__section">
        <div className="mx-auto max-w-7xl px-5 py-16 sm:px-8 lg:px-10">
          <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
            <div>
              <p className="public-relay__section-label">Visão geral</p>
              <h2 className="mt-3 font-heading text-3xl font-semibold tracking-tight text-white">Capacidades do relay</h2>
              <p className="mt-3 max-w-2xl text-slate-400">As informações abaixo são consultadas do documento NIP-11 desta instância.</p>
            </div>
            <Button className="border-white/15 bg-white/5 text-slate-200 hover:bg-white/10 hover:text-white" onClick={reload} size="sm" type="button" variant="outline"><RefreshCw /> Atualizar dados</Button>
          </div>

          <div className="grid gap-4 lg:grid-cols-[1.15fr_.85fr]">
            <Card className="public-relay__card">
              <CardHeader><div className="public-relay__icon"><Layers3 /></div><CardTitle>NIPs suportados</CardTitle><CardDescription>Capacidades anunciadas pelo relay nesta sessão.</CardDescription></CardHeader>
              <CardContent><p className="font-mono text-sm leading-7 text-slate-200">{nips}</p></CardContent>
            </Card>
            <Card className="public-relay__card">
              <CardHeader><div className="public-relay__icon public-relay__icon--cyan"><Server /></div><CardTitle>Limites atuais</CardTitle><CardDescription>Políticas reportadas pelo NIP-11.</CardDescription></CardHeader>
              <CardContent><p className="break-words font-mono text-xs leading-6 text-slate-300">{information?.limitation ? JSON.stringify(information.limitation) : "Carregando limites…"}</p></CardContent>
            </Card>
          </div>
          {error && <p role="alert" className="mt-4 rounded-2xl border border-rose-400/30 bg-rose-400/10 px-4 py-3 text-sm text-rose-100">{error}</p>}
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-5 py-16 sm:px-8 lg:px-10">
        <div className="mb-8">
          <p className="public-relay__section-label">API pública</p>
          <h2 className="mt-3 font-heading text-3xl font-semibold tracking-tight text-white">Conecte seu cliente</h2>
          <p className="mt-3 max-w-2xl text-slate-400">WebSocket Nostr e documento NIP-11; apenas endpoints efetivamente públicos desta instância são exibidos.</p>
        </div>
        <Card className="public-relay__card">
          <CardContent className="p-0">
            {endpoints.map((endpoint) => <div className="public-relay__endpoint" key={`${endpoint.method}:${endpoint.path}`}><code>{endpoint.method} {endpoint.path}</code><span>{endpoint.description}</span></div>)}
            {information && endpoints.length === 0 && <div className="public-relay__endpoint"><span>Nenhum endpoint HTTP adicional foi anunciado por esta instância.</span></div>}
          </CardContent>
        </Card>
      </section>

      <section className="public-relay__import">
        <div className="mx-auto max-w-7xl px-5 py-16 sm:px-8 lg:px-10">
          <div className="mb-8 max-w-2xl">
            <p className="public-relay__section-label">Importação no navegador</p>
            <h2 className="mt-3 font-heading text-3xl font-semibold tracking-tight text-white">Traga seus dados para cá</h2>
            <p className="mt-3 text-slate-400">Sua chave fica somente na memória do navegador. Os eventos são copiados sem serem alterados ou descriptografados.</p>
          </div>
          <DMImportPanel mode="public" localAuthRelayURL={information?.auth_relay_url} localRelayURL={relayURL} />
        </div>
      </section>
    </main>
  )
}

ReactDOM.createRoot(document.getElementById("root")!).render(<PublicRelayPage />)
