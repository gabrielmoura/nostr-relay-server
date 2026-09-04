// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }))
vi.mock("@/components/shared/page-header", () => ({ PageHeader: ({ title }: { title: string }) => <h1>{title}</h1> }))
vi.mock("@/components/shared/state-panels", () => ({
  LoadingPanel: ({ label }: { label?: string }) => <div data-testid="loading">{label}</div>,
  ErrorPanel: ({ onRetry }: { onRetry?: () => void }) => (
    <div data-testid="error">
      <button onClick={onRetry} type="button">retry</button>
    </div>
  ),
}))
vi.mock("@/components/shared/feature-disabled-panel", () => ({
  FeatureDisabledPanel: () => <div data-testid="disabled" />,
}))
vi.mock("@/components/privacy/privacy-networks-panel", () => ({
  PrivacyNetworksPanel: ({ networks }: { networks: unknown[] }) => <output data-testid="networks">{JSON.stringify(networks)}</output>,
}))
vi.mock("@/hooks/use-admin-data", () => ({ usePrivacyStatus: vi.fn() }))

import { usePrivacyStatus } from "@/hooks/use-admin-data"
import { PrivacyPage } from "@/components/privacy/privacy-page"

const mockUsePrivacyStatus = vi.mocked(usePrivacyStatus)
const refetch = vi.fn()

const enabledPrivacy = {
  enabled: true,
  persistence: true,
  state_dir: "data/privacy",
  networks: [{
    id: "yggdrasil" as const,
    name: "Yggdrasil",
    mode: "native",
    enabled: true,
    started: true,
    status: "error" as const,
    addresses: [],
    metrics: { tx_bytes: 0, rx_bytes: 0, peers: 0, connections: 0 },
    error: "peer unavailable",
    uptime_ms: 0,
  }],
}

function setQuery(overrides: Record<string, unknown>) {
  mockUsePrivacyStatus.mockReturnValue({
    data: enabledPrivacy,
    isLoading: false,
    isError: false,
    error: null,
    isFetching: false,
    refetch,
    ...overrides,
  } as ReturnType<typeof usePrivacyStatus>)
}

describe("PrivacyPage", () => {
  beforeEach(() => {
    refetch.mockReset()
  })

  afterEach(cleanup)

  it("renders only loading while the initial request is pending", () => {
    setQuery({ data: undefined, isLoading: true, isError: true })

    render(<PrivacyPage />)

    expect(screen.getByTestId("loading").textContent).toContain("privacy.loading")
    expect(screen.queryByTestId("error")).toBeNull()
  })

  it("renders an error for failed requests without data and consumes retry rejection", async () => {
    refetch.mockRejectedValueOnce(new Error("retry failed"))
    setQuery({ data: undefined, isError: true })

    render(<PrivacyPage />)
    fireEvent.click(screen.getByRole("button", { name: "retry" }))
    await Promise.resolve()

    expect(refetch).toHaveBeenCalledOnce()
    expect(screen.queryByTestId("loading")).toBeNull()
  })

  it("renders disabled, empty, and degraded zero-metric statuses as content", () => {
    setQuery({ data: { ...enabledPrivacy, enabled: false } })
    const { rerender } = render(<PrivacyPage />)
    expect(screen.getByTestId("disabled")).not.toBeNull()

    setQuery({ data: { ...enabledPrivacy, networks: [] } })
    rerender(<PrivacyPage />)
    expect(screen.getByTestId("networks").textContent).toContain("[]")

    setQuery({})
    rerender(<PrivacyPage />)
    expect(screen.getByTestId("networks").textContent).toContain('"tx_bytes":0')
    expect(screen.getByTestId("networks").textContent).toContain('"status":"error"')
  })
})
