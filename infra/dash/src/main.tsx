import React from "react"
import ReactDOM from "react-dom/client"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { RouterProvider } from "@tanstack/react-router"
import { Toaster } from "sonner"

import { NostrProvider } from "@/components/shared/nostr-provider"
import { router } from "@/router"
import "@/i18n"
import "@/index.css"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 15_000,
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <NostrProvider>
        <RouterProvider router={router} />
        <Toaster richColors position="top-right" closeButton />
      </NostrProvider>
    </QueryClientProvider>
  </React.StrictMode>,
)
