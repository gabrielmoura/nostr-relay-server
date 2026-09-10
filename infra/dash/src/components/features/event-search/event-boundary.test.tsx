// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { EventBoundary } from "./event-boundary"

let renderAttempts = 0

function BrokenEvent(): never {
  renderAttempts += 1
  throw new Error("Objects are not valid as a React child")
}

describe("EventBoundary", () => {
  afterEach(() => {
    cleanup()
    renderAttempts = 0
    vi.restoreAllMocks()
  })

  it("isolates a malformed event and keeps adjacent cards rendered", () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined)

    render(
      <div>
        <EventBoundary eventId="malformed-event" eventPayload={{ id: "malformed-event", content: { invalid: true } }}>
          <BrokenEvent />
        </EventBoundary>
        <article>Evento íntegro</article>
      </div>,
    )

    expect(screen.getByText("Não foi possível renderizar este evento")).not.toBeNull()
    expect(screen.getByText("Evento íntegro")).not.toBeNull()
    expect(console.error).toHaveBeenCalledWith(
      "Falha ao renderizar evento Nostr no painel de management",
      expect.objectContaining({ eventId: "malformed-event" }),
    )
  })

  it("retries only the failed card", () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined)
    const { getByRole } = render(
      <EventBoundary eventId="event-1" eventPayload={{ id: "event-1" }}>
        <BrokenEvent />
      </EventBoundary>,
    )

    fireEvent.click(getByRole("button", { name: "Tentar novamente" }))

    expect(renderAttempts).toBeGreaterThan(1)
  })
})
