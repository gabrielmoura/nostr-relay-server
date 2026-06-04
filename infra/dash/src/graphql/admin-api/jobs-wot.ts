import {
  AddTrustedPubkeyDocument,
  CancelJobDocument,
  DeleteJobsHistoryDocument,
  DownloadEventsDocument,
  DownloadJobDocument,
  DownloadJobsDocument,
  GroupsDocument,
  JobDocument,
  JobsDocument,
  RemoveTrustedPubkeyDocument,
  ResumeJobDocument,
  RetryJobDocument,
  StartNegentropySyncDocument,
  WotSummaryDocument,
} from "@/graphql/generated/operations"

import { emptyToNull, extractDeletedCount, graphMutation, graphQuery, jobsFilterVars, keysToSnake, listToPage, mapDownloadJob, pageFromGraph, pageVars, type RequestContext } from "./core"

export async function handleJobsAndOpsRequest<T>(ctx: RequestContext): Promise<T | undefined> {
  const { method, pathname, search, body } = ctx

  switch (true) {
    case method === "POST" && pathname === "/sync/negentropy":
      return graphMutation(StartNegentropySyncDocument, { input: { remote: body?.remote, direction: body?.direction, filter: Array.isArray(body?.filter) ? body.filter : body?.filter ? [body.filter] : null, timeoutSeconds: body?.timeout ?? null } }, (data) => ({ status: data.startNegentropySync.status, remote: data.startNegentropySync.remote, message: data.startNegentropySync.message, job_id: data.startNegentropySync.jobId })) as Promise<T>
    case method === "POST" && pathname === "/events/download":
      return graphMutation(DownloadEventsDocument, { input: { relays: body?.relays ?? [], publicKey: body?.public_key ?? null, kinds: body?.kinds ?? null, filter: body?.filter ?? null, timeoutSeconds: body?.timeout ?? null } }, (data) => ({ status: data.downloadEvents.status, job_id: data.downloadEvents.jobId, relays: data.downloadEvents.relays, message: data.downloadEvents.message })) as Promise<T>
    case method === "GET" && pathname === "/events/download/jobs":
      return graphQuery(DownloadJobsDocument, undefined, (data) => ({ items: data.jobs.items.map(mapDownloadJob) })) as Promise<T>
    case method === "GET" && pathname.match(/^\/events\/download\/jobs\//) !== null: {
      const jobId = decodeURIComponent(pathname.split("/")[4] ?? "")
      return graphQuery(DownloadJobDocument, { id: jobId }, (data) => mapDownloadJob(data.job)) as Promise<T>
    }
    case method === "GET" && pathname === "/jobs":
      return graphQuery(JobsDocument, { filter: jobsFilterVars(search), page: pageVars(search) }, (data) => pageFromGraph(data.jobs, (item) => keysToSnake(item))) as Promise<T>
    case method === "GET" && pathname.match(/^\/jobs\//) !== null: {
      const jobId = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphQuery(JobDocument, { id: jobId, queue: emptyToNull(search.get("queue")) }, (data) => keysToSnake(data.job)) as Promise<T>
    }
    case method === "POST" && pathname.match(/^\/jobs\/[^/]+\/retry$/) !== null: {
      const jobId = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphMutation(RetryJobDocument, { id: jobId, queue: body?.queue ?? null }, () => ({ ok: true, id: jobId, queue: body?.queue ?? null })) as Promise<T>
    }
    case method === "POST" && pathname.match(/^\/jobs\/[^/]+\/resume$/) !== null: {
      const jobId = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphMutation(ResumeJobDocument, { id: jobId, queue: body?.queue ?? null }, () => ({ ok: true, id: jobId, queue: body?.queue ?? null })) as Promise<T>
    }
    case method === "POST" && pathname.match(/^\/jobs\/[^/]+\/cancel$/) !== null: {
      const jobId = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphMutation(CancelJobDocument, { id: jobId, queue: body?.queue ?? null }, () => ({ ok: true, id: jobId, queue: body?.queue ?? null })) as Promise<T>
    }
    case method === "DELETE" && pathname === "/jobs":
      return graphMutation(DeleteJobsHistoryDocument, { input: { jobName: search.get("job_name"), statuses: search.getAll("status") } }, (data) => ({ deleted: extractDeletedCount(data.deleteJobsHistory.message) })) as Promise<T>
    case method === "GET" && pathname === "/groups":
			return graphQuery(GroupsDocument, undefined, (data) => listToPage(data.groups.map((item: any) => keysToSnake(item)), search)) as Promise<T>
    case method === "GET" && pathname === "/wot/summary":
      return graphQuery(WotSummaryDocument, undefined, (data) => keysToSnake(data.wotSummary)) as Promise<T>
    case method === "POST" && pathname === "/wot/trusted":
      return graphMutation(AddTrustedPubkeyDocument, { pubkey: body?.pubkey }, () => ({ pubkey: body?.pubkey, added: true })) as Promise<T>
    case method === "DELETE" && pathname.match(/^\/wot\/trusted\//) !== null: {
      const pubkey = decodeURIComponent(pathname.split("/")[3] ?? "")
      return graphMutation(RemoveTrustedPubkeyDocument, { pubkey }, () => ({ pubkey, removed: true })) as Promise<T>
    }
    default:
      return undefined
  }
}
