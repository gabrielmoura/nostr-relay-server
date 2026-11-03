// @vitest-environment jsdom
import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { DMImportPanel } from "./dm-import-panel"

describe("DMImportPanel", () => {
  it("renders in public mode without admin providers", () => {
    render(<DMImportPanel mode="public" bootstrapRelays={[]} localRelayURL="ws://local" />)
    expect(screen.getByText("Importar dados Nostr")).toBeTruthy()
    expect(screen.getByLabelText("nsec ou chave hexadecimal")).toBeTruthy()
    expect(screen.getByRole("button", { name: "Importar DMs com nsec" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Importar outros eventos com nsec" })).toBeTruthy()
  })
})
