import { useRef, useState } from "react"
import { KeyRound, ShieldCheck, UploadCloud } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { importDMs } from "./importer"
import { importRelayTransport } from "./relay-client"
import { extensionSigner, secretSigner } from "./signer"
import type { ImportProgress, ImportSigner } from "./types"

declare global { interface Window { nostr?: { getPublicKey(): Promise<string>; signEvent(event: any): Promise<any> } } }

export interface DMImportPanelProps { mode: "admin" | "public"; bootstrapRelays?: string[]; localRelayURL?: string; localAuthRelayURL?: string }
const initial: ImportProgress = { relays: 0, completedRelays: 0, failedRelays: 0, found: 0, imported: 0, duplicates: 0, rejected: 0, uncertain: 0, errors: [], incomplete: false, phase: "idle" }
const configuredBootstrap = (import.meta.env.VITE_DM_IMPORT_BOOTSTRAP_RELAYS ?? "wss://relay.damus.io,wss://nos.lol").split(",").map((value: string) => value.trim()).filter(Boolean)
const configuredLocalRelay = import.meta.env.VITE_PUBLIC_RELAY_URL ?? (() => {
  const url = new URL(window.location.href)
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:"
  if (url.port === "4870") url.port = "4869"
  url.pathname = "/"
  return url.toString()
})()

export function DMImportPanel({ mode, bootstrapRelays = configuredBootstrap, localRelayURL = configuredLocalRelay, localAuthRelayURL }: DMImportPanelProps) {
  const [nsec, setNsec] = useState("")
  const [progress, setProgress] = useState(initial)
  const [running, setRunning] = useState(false)
  const controllerRef = useRef<AbortController | undefined>(undefined)
  const start = async (kind: "extension" | "secret") => {
    let signer: ImportSigner | undefined
    try {
      signer = kind === "extension" ? (window.nostr ? extensionSigner(window.nostr) : undefined) : secretSigner(nsec)
      if (!signer) throw new Error("Extensão NIP-07 não encontrada")
      setNsec("")
      setRunning(true); setProgress(initial)
      const controller = new AbortController()
      controllerRef.current = controller
      await importDMs({ transport: importRelayTransport, signer, bootstrap: bootstrapRelays, localRelay: localRelayURL, localAuthRelay: localAuthRelayURL, onProgress: setProgress, signal: controller.signal })
    } catch (error) { setProgress((current) => ({ ...current, phase: "error", errors: [...current.errors, error instanceof Error ? error.message : "falha"] })) }
    finally { controllerRef.current = undefined; signer?.dispose(); setNsec(""); setRunning(false) }
  }
  return <section className="space-y-6" aria-label="Importação de eventos"><Card className="overflow-hidden"><CardHeader className="border-b bg-primary/5"><div className="flex items-start gap-3"><div className="rounded-lg bg-primary p-2 text-primary-foreground"><UploadCloud className="size-5" /></div><div><CardTitle>Importar dados Nostr</CardTitle><CardDescription>{mode === "admin" ? "Copie perfil, configurações, notas e envelopes cifrados para este relay." : "A chave permanece apenas na memória deste navegador."}</CardDescription></div></div></CardHeader><CardContent className="grid gap-5 p-6 md:grid-cols-2"><div className="space-y-3 rounded-lg border bg-muted/30 p-4"><div className="flex items-center gap-2 font-medium"><ShieldCheck className="size-4 text-success" />Extensão NIP-07</div><p className="text-sm text-muted-foreground">Use a extensão para assinar a autenticação de cada relay.</p><Button className="w-full" disabled={running || !window.nostr} onClick={() => void start("extension")} type="button">Importar com extensão</Button></div><div className="space-y-3 rounded-lg border bg-muted/30 p-4"><div className="flex items-center gap-2 font-medium"><KeyRound className="size-4 text-primary" />Chave nsec</div><Label htmlFor="dm-import-nsec">nsec ou chave hexadecimal</Label><Input autoComplete="off" id="dm-import-nsec" onChange={(event) => setNsec(event.target.value)} placeholder="nsec1…" type="password" value={nsec}/><Button className="w-full" disabled={running || !nsec} onClick={() => void start("secret")} type="button">Importar com nsec</Button></div></CardContent></Card><Card><CardHeader><div className="flex flex-wrap items-center justify-between gap-3"><div><CardTitle>Progresso: {progress.phase}</CardTitle><CardDescription>{progress.completedRelays}/{progress.relays} relays concluídos</CardDescription></div>{running && <Button onClick={() => controllerRef.current?.abort()} type="button" variant="outline">Cancelar importação</Button>}</div></CardHeader><CardContent className="grid grid-cols-2 gap-3 text-center text-sm sm:grid-cols-5"><Metric value={progress.found} label="encontrados" /><Metric value={progress.imported} label="importados" /><Metric value={progress.duplicates} label="duplicados" /><Metric value={progress.rejected} label="rejeitados" /><Metric value={progress.uncertain} label="incertos" />{progress.incomplete && <p className="col-span-2 rounded border border-warning/40 bg-warning/10 p-3 text-left text-warning-foreground sm:col-span-5">A coleta atingiu um limite, foi cancelada ou encontrou timestamps que não permitem avançar com segurança. O resumo pode estar incompleto.</p>}{progress.errors.length > 0 && <ul className="col-span-2 space-y-1 text-left text-destructive sm:col-span-5">{progress.errors.map((error) => <li key={error}>{error}</li>)}</ul>}</CardContent></Card></section>
}

function Metric({ value, label }: { value: number; label: string }) {
  return <div className="rounded bg-muted p-3"><strong className="block text-lg">{value}</strong>{label}</div>
}
