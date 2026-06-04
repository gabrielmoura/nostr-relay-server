import { adminApolloClient } from "@/graphql/client"
import { GraphQLApiError, graphUpload, keysToSnake, mapEventRecord } from "@/graphql/helpers"
import { EventDetailDocument, FetchEventFromRelaysDocument } from "@/graphql/generated/operations"
import { toNevent, toNote, toNprofile, toNpub } from "@/lib/nostr"
import type { EventDetail, FetchEventFromRelaysResponse, ImportEventsResponse } from "@/types/admin"

import { ApiError } from "./admin"

function toGraphApiError(error: unknown): never {
  if (error instanceof GraphQLApiError) {
    throw new ApiError(error.message, error.status, error.details, error.requestId)
  }
  throw error
}

export async function getEventDetail(eventID: string) {
  try {
    const result = await adminApolloClient.query({ query: EventDetailDocument, variables: { id: eventID } })
    const data = result.data
    if (!data) {
      throw new ApiError("GraphQL query returned no data")
    }

    const response = {
      event: mapEventRecord(data.eventDetail.event),
      identifiers: keysToSnake(data.eventDetail.identifiers),
      author: keysToSnake(data.eventDetail.author),
      hashtags: data.eventDetail.hashtags,
      image_urls: data.eventDetail.imageUrls,
    } satisfies EventDetail

    const pubkey = response.event?.pubkey ?? response.author?.pubkey ?? ""

    return {
      ...response,
      identifiers: {
        note: response.identifiers?.note ?? toNote(response.event.id),
        nevent: response.identifiers?.nevent ?? toNevent(response.event.id, pubkey),
        npub: response.identifiers?.npub ?? toNpub(pubkey),
        nprofile: response.identifiers?.nprofile ?? toNprofile(pubkey),
      },
    }
  } catch (error) {
    toGraphApiError(error)
  }
}

export async function fetchEventFromRelays(eventID: string, relays: string[]) {
  try {
    const result = await adminApolloClient.mutate({
      mutation: FetchEventFromRelaysDocument,
      variables: { id: eventID, input: { relays } },
    })
    const data = result.data
    if (!data) {
      throw new ApiError("GraphQL mutation returned no data")
    }

    return {
      event_id: data.fetchEventFromRelays.eventId ?? "",
      source_relay: data.fetchEventFromRelays.sourceRelay ?? "",
      found: data.fetchEventFromRelays.found,
      persisted: data.fetchEventFromRelays.persisted,
      relays_tried: data.fetchEventFromRelays.relaysTried,
      relay_results: keysToSnake(data.fetchEventFromRelays.relayResults),
      message: data.fetchEventFromRelays.message ?? undefined,
    } satisfies FetchEventFromRelaysResponse
  } catch (error) {
    toGraphApiError(error)
  }
}

export async function importEventsFiles(files: File[]) {
  try {
    const formData = new FormData()
    for (const file of files) {
      formData.append("files", file)
    }

    return await graphUpload<ImportEventsResponse>(
      "mutation ImportEvents($files: [Upload!]!) { importEvents(files: $files) { files { filename total inserted duplicates invalid error } } }",
      formData,
      (data: any) => data.importEvents,
    )
  } catch (error) {
    if (error instanceof GraphQLApiError) {
      throw new ApiError(error.message, error.status, error.details, error.requestId)
    }
    throw error
  }
}

export function getEventTags(event: { tags: string[][] }) {
  return event.tags.filter((entry: string[]) => entry[0] === "t").map((entry: string[]) => entry[1])
}
