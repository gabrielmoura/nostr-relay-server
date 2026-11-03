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

export function createAdminCache() {
  return new InMemoryCache({
    typePolicies: {
      AdminBlossomObject: { keyFields: ["hash"] },
      Query: {
        fields: {
          blossomObjects: {
            keyArgs: ["filter"],
            merge(existing, incoming, { args, readField }) {
              const offset = args?.page?.offset ?? 0
              if (offset === 0 || !existing) return incoming
              const items = existing.items.slice(0, offset)
              const seen = new Set(items.map((item: unknown) => readField<string>("hash", item as never)))
              for (const item of incoming.items) {
                const hash = readField<string>("hash", item)
                if (!seen.has(hash)) {
                  items.push(item)
                  seen.add(hash)
                }
              }
              // Preserve the next server offset even when a volatile list
              // contains an object already present in an earlier page.
              return { ...incoming, items, pageInfo: { ...incoming.pageInfo, offset: offset + incoming.items.length - items.length } }
            },
          },
        },
      },
    },
  })
}

export const adminApolloClient = new ApolloClient({
  cache: createAdminCache(),
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
