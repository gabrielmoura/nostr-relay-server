import { expect, it } from "vitest"

import { blossomObjectFilterVars } from "./helpers"

it("passes the general Blossom search to the GraphQL filter", () => {
  const variables = blossomObjectFilterVars(new URLSearchParams({ q: "image/png", review_state: "ready" }))
  expect(variables.q).toBe("image/png")
  expect(variables.reviewState).toBe("ready")
})
