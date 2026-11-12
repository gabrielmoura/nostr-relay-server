// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, expect, it, vi } from "vitest"

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }))
vi.mock("@tanstack/react-router", () => ({ Link: ({ children }: { children: React.ReactNode }) => <a href="/user">{children}</a> }))
vi.mock("@/components/shared/virtualized-list", () => ({ VirtualizedList: ({ items, renderItem }: { items: unknown[]; renderItem: (item: unknown) => React.ReactNode }) => <div>{items.map((item, index) => <div key={index}>{renderItem(item)}</div>)}</div> }))
vi.mock("@/components/features/ban-user-dialog", () => ({ BanUserDialog: () => <button>Ban</button> }))
vi.mock("@/hooks/use-admin-data", () => ({
  useInfiniteConnections: () => ({ data: { pages: [{ total: 1, items: [{ ws_id: "connection-id", ip: "127.0.0.1", authed: "user-public-key", subscription_count: 1, connected_at: "2024-01-01T00:00:00Z", last_seen_at: "2024-01-02T00:00:00Z", user_agent: "Example client" }] }] }, isLoading: false, isError: false }),
  useDisconnectConnectionMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useInfiniteLoggedUsers: () => ({ data: { pages: [{ total: 1, items: [{ pubkey: "user-public-key", npub: "npub-test", displayName: "Alice", handle: "alice", nip05: "alice@example.org", website: "https://example.org", metadata: "Profile text", bot: true, connectionCount: 1, connectionState: "stable" }] }] }, isLoading: false, isError: false }),
}))

import { ActiveConnectionsPage } from "./active-connections-page"
import { LoggedConnectionsPage } from "./logged-connections-page"
import { LoggedUsersPage } from "./logged-users-page"

afterEach(cleanup)

it.each([ActiveConnectionsPage, LoggedConnectionsPage])("shows connection details without guessing the network origin", (Page) => {
  render(<Page />)
  expect(screen.getByText("Origem de rede: não identificada")).toBeTruthy()
  expect(screen.getByText(/Conectado:/)).toBeTruthy()
  expect(screen.getByText(/Example client/)).toBeTruthy()
  expect(screen.getByRole("link")).toBeTruthy()
})

it("shows available profile details and an unknown network for logged users", () => {
  render(<LoggedUsersPage />)
  expect(screen.getByText("NIP-05: alice@example.org")).toBeTruthy()
  expect(screen.getByText("Site informado: https://example.org")).toBeTruthy()
  expect(screen.getByText("Profile text")).toBeTruthy()
  expect(screen.getByText("Origem de rede: não identificada")).toBeTruthy()
})
