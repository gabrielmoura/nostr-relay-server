// @vitest-environment jsdom

import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => ({
      "common.howToEnable": "How to enable",
      "common.configurationKey": "Configuration key",
      "common.restartRequired": "Restart the relay server after changing the configuration, then check again.",
      "common.retry": "Check again",
    }[key] ?? key),
  }),
}))

import { FeatureDisabledPanel } from "@/components/shared/feature-disabled-panel"

describe("FeatureDisabledPanel", () => {
  afterEach(cleanup)

  it("shows the recovery steps and a refresh action", () => {
    render(
      <FeatureDisabledPanel
        configKey="privacy.enabled"
        description="Privacy monitoring is unavailable."
        howToEnable={"privacy:\n  enabled: true"}
        title="Privacy feature is disabled"
      />,
    )

    expect(screen.getByRole("heading", { name: "Privacy feature is disabled" })).toBeTruthy()
    expect(screen.getByText("How to enable")).toBeTruthy()
    expect(screen.getByText("privacy.enabled")).toBeTruthy()
    expect(screen.getByText("privacy:\n  enabled: true")).toBeTruthy()
    expect(screen.getByRole("button", { name: "Check again" })).toBeTruthy()
  })
})
