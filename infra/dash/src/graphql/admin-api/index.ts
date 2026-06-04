import { buildRequestContext, GraphQLApiError, graphUpload, type RequestInitLike } from "./core"
import { handleBlossomRequest } from "./blossom"
import { handleEventsRequest } from "./events"
import { handleIdentityRequest } from "./nip05-nip86"
import { handleJobsAndOpsRequest } from "./jobs-wot"
import { handleUsersRequest } from "./users"

export { GraphQLApiError }

export async function graphqlRequest<T>(path: string, init?: RequestInitLike): Promise<T> {
	const ctx = await buildRequestContext(path, init)

	const handlers = [
		handleUsersRequest,
		handleEventsRequest,
		handleIdentityRequest,
		handleJobsAndOpsRequest,
		handleBlossomRequest,
	]

	for (const handler of handlers) {
		const result = await handler<T>(ctx)
		if (result !== undefined) {
			return result
		}
	}

	if (ctx.method === "POST" && ctx.pathname === "/events/import") {
		return graphUpload<T>(
			"mutation ImportEvents($files: [Upload!]!) { importEvents(files: $files) { files { filename total inserted duplicates invalid error } } }",
			ctx.init?.body as FormData,
			(data) => data.importEvents,
		)
	}

	throw new GraphQLApiError(`GraphQL mapping not implemented for ${ctx.method} ${ctx.pathname}`)
}
