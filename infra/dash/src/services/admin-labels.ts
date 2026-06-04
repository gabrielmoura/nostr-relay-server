import { adminApolloClient } from "@/graphql/client"
import { GraphQLApiError } from "@/graphql/helpers"
import { CreateLabelDocument, LabelsDocument, LabelsSummaryDocument } from "@/graphql/generated/operations"
import type { LabelTargetType } from "@/graphql/generated/operations"
import type { AdminLabelEvent, AdminLabelsFilters, AdminLabelsSummary, AdminPage, CreateAdminLabelPayload } from "@/types/admin"

import { ApiError } from "./admin"

type PageParams = {
  limit: number
  offset: number
}

function toGraphApiError(error: unknown): never {
  if (error instanceof GraphQLApiError) {
    throw new ApiError(error.message, error.status, error.details, error.requestId)
  }
  throw error
}

function toLabelTargetType(value?: string | null): LabelTargetType | undefined {
  const normalized = value?.trim().toLowerCase()
  if (normalized === "event") return "EVENT"
  if (normalized === "pubkey") return "PUBKEY"
  if (normalized === "address") return "ADDRESS"
  if (normalized === "reference") return "REFERENCE"
  if (normalized === "topic") return "TOPIC"
  return undefined
}

function toAdminLabelTargetType(value: string): AdminLabelEvent["target"]["type"] {
  const normalized = value.trim().toLowerCase()
  if (normalized === "event" || normalized === "pubkey" || normalized === "address" || normalized === "reference" || normalized === "topic") {
    return normalized
  }
  return "reference"
}

function buildLabelsFilter(filters: AdminLabelsFilters) {
  return {
    namespace: filters.namespace || null,
      labels: filters.labels?.length ? filters.labels : null,
      targetType: filters.target_type ? toLabelTargetType(filters.target_type) : null,
      target: filters.target || null,
      author: filters.author || null,
      q: filters.q || null,
  }
}

function mapLabelEvent(item: any): AdminLabelEvent {
  return {
    id: item.id,
    pubkey: item.pubkey,
    author_npub: item.authorNpub ?? undefined,
    created_at: item.createdAt,
    kind: item.kind,
    content: item.content,
    namespace: item.namespace,
    labels: item.labels ?? [],
    target: {
      type: toAdminLabelTargetType(String(item.target.type)),
      value: item.target.value,
      relay_hint: item.target.relayHint ?? undefined,
    },
    tags: (item.tags ?? []).map((tag: any) => tag.values),
  }
}

export async function getLabelsPage(filters: AdminLabelsFilters, params: PageParams) {
  try {
    const result = await adminApolloClient.query({
      query: LabelsDocument,
      variables: {
        filter: buildLabelsFilter(filters),
        page: { limit: params.limit, offset: params.offset },
      },
    })
    const data = result.data
    if (!data) {
      throw new ApiError("GraphQL query returned no data")
    }

    return {
      items: data.labels.items.map(mapLabelEvent),
      total: data.labels.pageInfo.total,
      limit: data.labels.pageInfo.limit,
      offset: data.labels.pageInfo.offset,
      has_more: data.labels.pageInfo.hasMore,
    } satisfies AdminPage<AdminLabelEvent>
  } catch (error) {
    toGraphApiError(error)
  }
}

export async function getLabelsSummary(filters: AdminLabelsFilters) {
  try {
    const result = await adminApolloClient.query({
      query: LabelsSummaryDocument,
      variables: {
        filter: buildLabelsFilter(filters),
      },
    })
    const data = result.data
    if (!data) {
      throw new ApiError("GraphQL query returned no data")
    }

    return {
      total_events: data.labelsSummary.totalEvents,
      total_targets: data.labelsSummary.totalTargets,
      namespaces: data.labelsSummary.namespaces.map((item: any) => ({ namespace: item.name, count: item.count })),
      labels: data.labelsSummary.labels.map((item: any) => ({ label: item.name, count: item.count })),
      target_types: data.labelsSummary.targetTypes.map((item: any) => ({ target_type: toAdminLabelTargetType(String(item.name)), count: item.count })),
    } satisfies AdminLabelsSummary
  } catch (error) {
    toGraphApiError(error)
  }
}

export async function createLabel(payload: CreateAdminLabelPayload) {
  try {
    const result = await adminApolloClient.mutate({
      mutation: CreateLabelDocument,
      variables: {
        input: {
          namespace: payload.namespace,
          labels: payload.labels,
          comment: payload.comment ?? null,
          target: {
            type: toLabelTargetType(payload.target.type) ?? "REFERENCE",
            value: payload.target.value,
            relayHint: payload.target.relay_hint ?? null,
          },
        },
      },
    })
    const data = result.data
    if (!data) {
      throw new ApiError("GraphQL mutation returned no data")
    }

    return {
      event: {
        id: data.createLabel.event.id,
        pubkey: data.createLabel.event.pubkey,
        created_at: data.createLabel.event.createdAt,
        kind: data.createLabel.event.kind,
        content: data.createLabel.event.content,
        tags: (data.createLabel.event.tags ?? []).map((tag: any) => tag.values),
        sig: data.createLabel.event.sig,
      },
      stored: data.createLabel.stored,
    }
  } catch (error) {
    toGraphApiError(error)
  }
}
