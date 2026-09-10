import type { ReactNode } from "react"
import { AlertTriangle, RefreshCw } from "lucide-react"
import { ErrorBoundary, type FallbackProps } from "react-error-boundary"

import { Button } from "@/components/ui/button"

interface EventBoundaryProps {
  eventId: string
  eventPayload: unknown
  children: ReactNode
}

interface EventRenderErrorFallbackProps extends FallbackProps {
  eventId: string
}

function EventRenderErrorFallback({ eventId, resetErrorBoundary }: EventRenderErrorFallbackProps) {
  return (
    <section aria-live="polite" className="relative overflow-hidden rounded-md border border-destructive/35 bg-card/30 p-3">
      <div className="absolute inset-y-0 left-0 w-1 bg-destructive/70" />
      <div className="ml-2 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 items-start gap-3">
          <AlertTriangle aria-hidden="true" className="mt-0.5 size-4 shrink-0 text-destructive" />
          <div className="min-w-0 space-y-1">
            <p className="text-sm font-semibold text-foreground">Não foi possível renderizar este evento</p>
            <p className="break-all font-mono text-xs text-muted-foreground">ID: {eventId}</p>
          </div>
        </div>
        <Button className="shrink-0" onClick={resetErrorBoundary} size="sm" variant="outline">
          <RefreshCw aria-hidden="true" />
          Tentar novamente
        </Button>
      </div>
    </section>
  )
}

function logEventRenderError(error: unknown, info: { componentStack?: string | null }, eventId: string, eventPayload: unknown) {
  console.error("Falha ao renderizar evento Nostr no painel de management", {
    error,
    componentStack: info.componentStack,
    eventId,
    eventPayload,
  })
}

export function EventBoundary({ eventId, eventPayload, children }: EventBoundaryProps) {
  return (
    <ErrorBoundary
      fallbackRender={(props) => <EventRenderErrorFallback {...props} eventId={eventId} />}
      onError={(error, info) => logEventRenderError(error, info, eventId, eventPayload)}
      resetKeys={[eventId]}
    >
      {children}
    </ErrorBoundary>
  )
}
