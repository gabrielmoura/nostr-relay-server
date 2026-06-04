import { ApolloProvider } from "@apollo/client/react"
import React from "react"
import ReactDOM from "react-dom/client"
import { RouterProvider } from "@tanstack/react-router"
import { Toaster } from "sonner"

import { NostrProvider } from "@/components/shared/nostr-provider"
import { adminApolloClient } from "@/graphql/client"
import { router } from "@/router"
import "@/i18n"
import "@/index.css"

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ApolloProvider client={adminApolloClient}>
      <NostrProvider>
        <RouterProvider router={router} />
        <Toaster richColors position="top-right" closeButton />
      </NostrProvider>
    </ApolloProvider>
  </React.StrictMode>,
)
