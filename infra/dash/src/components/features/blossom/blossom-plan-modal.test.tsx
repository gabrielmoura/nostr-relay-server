// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, expect, it, vi } from "vitest"

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (_key: string, fallback: string) => fallback }) }))

import { BlossomPlanModal } from "./blossom-plan-modal"

afterEach(cleanup)

it("preserves zero and exact byte quotas when editing a plan", () => {
  const onSave = vi.fn()
  render(<BlossomPlanModal open onOpenChange={() => {}} onSave={onSave} saving={false}
    plan={{ id: "test", name: "Test", scope: "free", is_default: false, storage_quota_bytes: 0, egress_quota_bytes: 1 }} />)
  expect((screen.getByLabelText("Armazenamento (MB)") as HTMLInputElement).value).toBe("0")
  fireEvent.click(screen.getByRole("button", { name: "Salvar" }))
  expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ storage_quota_bytes: 0, egress_quota_bytes: 1 }))
})

it("rejects a blank name and negative quota before saving", () => {
  const onSave = vi.fn()
  render(<BlossomPlanModal open onOpenChange={() => {}} onSave={onSave} saving={false} />)
  fireEvent.change(screen.getByLabelText("ID tecnico"), { target: { value: "plan" } })
  fireEvent.change(screen.getByLabelText("Nome do plano"), { target: { value: "  " } })
  fireEvent.change(screen.getByLabelText("Armazenamento (MB)"), { target: { value: "-1" } })
  fireEvent.click(screen.getByRole("button", { name: "Salvar" }))
  expect(onSave).not.toHaveBeenCalled()
  expect(screen.getAllByRole("alert")).toHaveLength(2)
})
