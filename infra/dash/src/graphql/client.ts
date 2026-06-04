import { ApolloClient, HttpLink, InMemoryCache } from "@apollo/client"

import { env } from "@/lib/env"

const httpLink = new HttpLink({
  uri: `${env.adminBaseUrl}/graphql`,
  fetch: async (input, init) => {
    const headers = new Headers(init?.headers)
    if (env.adminToken) {
      headers.set("X-Admin-Token", env.adminToken)
    }

    return fetch(input, {
      ...init,
      headers,
    })
  },
})

export const adminApolloClient = new ApolloClient({
  cache: new InMemoryCache(),
  link: httpLink,
  devtools: {
    enabled: true,
  },
  defaultOptions: {
    query: {
      fetchPolicy: "no-cache",
    },
    watchQuery: {
      fetchPolicy: "no-cache",
    },
  },
})
