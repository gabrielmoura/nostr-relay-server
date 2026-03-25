const defaultAdminBaseUrl = "/admin"

export const env = {
  adminBaseUrl: (import.meta.env.VITE_ADMIN_API_URL as string | undefined) ?? defaultAdminBaseUrl,
  adminToken: (import.meta.env.VITE_ADMIN_TOKEN as string | undefined) ?? "",
  mockOnFailure: ((import.meta.env.VITE_MOCK_ON_FAILURE as string | undefined) ?? "false") === "true",
}
