import {
  AssignBlossomPlanDocument,
  BlossomAnalyticsDocument,
  BlossomAuditDocument,
  BlossomObjectDocument,
  BlossomObjectsDocument,
  BlossomOverviewDocument,
  BlossomPlanAssignmentsDocument,
  BlossomPlansDocument,
  BlossomPolicyDocument,
  BlossomReportsDocument,
  BlossomUserDocument,
  BlossomUsersDocument,
  BlossomWorkersDocument,
  DeleteBlossomPlanDocument,
  MirrorBlossomObjectDocument,
  PurgeBlossomUserDocument,
  ResolveBlossomReportDocument,
  ReviewBlossomObjectsDocument,
  UnassignBlossomPlanDocument,
  UpsertBlossomPlanDocument,
  UpsertBlossomPolicyDocument,
  UpsertBlossomWhitelistDocument,
} from "@/graphql/generated/operations"

import {
  blossomObjectFilterVars,
  blossomReportFilterVars,
  blossomUserFilterVars,
  emptyToNull,
  graphMutation,
  graphQuery,
  keysToSnake,
  normalizeBlossomObject,
  normalizeBlossomObjectRecord,
  normalizeBlossomOverview,
  normalizeBlossomPlan,
  normalizeBlossomPolicy,
  normalizeBlossomReport,
  normalizeBlossomUser,
  pageFromGraph,
  pageVars,
  type RequestContext,
} from "./core"

export async function handleBlossomRequest<T>(ctx: RequestContext): Promise<T | undefined> {
	const { method, pathname, search, body } = ctx

	switch (true) {
		case method === "GET" && pathname === "/blossom/overview":
			return graphQuery(BlossomOverviewDocument, undefined, (data) => normalizeBlossomOverview(data.blossomOverview)) as Promise<T>
		case method === "GET" && pathname === "/blossom/policy":
			return graphQuery(BlossomPolicyDocument, undefined, (data) => normalizeBlossomPolicy(data.blossomPolicy)) as Promise<T>
		case method === "PUT" && pathname === "/blossom/policy":
			return graphMutation(UpsertBlossomPolicyDocument, { input: { mode: body?.mode } }, (data) => normalizeBlossomPolicy(data.upsertBlossomPolicy)) as Promise<T>
		case method === "GET" && pathname === "/blossom/plans":
			return graphQuery(BlossomPlansDocument, undefined, (data) => ({ items: data.blossomPlans.map(normalizeBlossomPlan) })) as Promise<T>
		case method === "PUT" && pathname === "/blossom/plans":
			return graphMutation(UpsertBlossomPlanDocument, { input: { id: body?.id, name: body?.name, scope: String(body?.scope ?? "").toUpperCase(), storageQuotaBytes: body?.storage_quota_bytes ?? null, egressQuotaBytes: body?.egress_quota_bytes ?? null, description: body?.description ?? null, isDefault: body?.is_default } }, (data) => normalizeBlossomPlan(data.upsertBlossomPlan)) as Promise<T>
		case method === "DELETE" && pathname.match(/^\/blossom\/plans\//) !== null && !pathname.includes("/assign/") && !pathname.endsWith("/assign"): {
			const id = decodeURIComponent(pathname.split("/")[3] ?? "")
			return graphMutation(DeleteBlossomPlanDocument, { id }, () => ({ ok: true, id })) as Promise<T>
		}
		case method === "POST" && pathname.match(/^\/blossom\/plans\/[^/]+\/assign$/) !== null: {
			const planId = decodeURIComponent(pathname.split("/")[3] ?? "")
			return graphMutation(AssignBlossomPlanDocument, { planId, pubkey: body?.pubkey }, () => ({ ok: true, plan_id: planId, pubkey: body?.pubkey })) as Promise<T>
		}
		case method === "DELETE" && pathname.match(/^\/blossom\/plans\/[^/]+\/assign\/[^/]+$/) !== null: {
			const planId = decodeURIComponent(pathname.split("/")[3] ?? "")
			const pubkey = decodeURIComponent(pathname.split("/")[5] ?? "")
			return graphMutation(UnassignBlossomPlanDocument, { planId, pubkey }, () => ({ ok: true, plan_id: planId, pubkey })) as Promise<T>
		}
		case method === "GET" && pathname.match(/^\/blossom\/plans\/[^/]+\/assignments$/) !== null: {
			const planId = decodeURIComponent(pathname.split("/")[3] ?? "")
			return graphQuery(BlossomPlanAssignmentsDocument, { planId }, (data) => ({ items: data.blossomPlanAssignments.items.map((item: any) => keysToSnake(item)) })) as Promise<T>
		}
		case method === "GET" && pathname === "/blossom/objects":
			return graphQuery(BlossomObjectsDocument, { filter: blossomObjectFilterVars(search), page: pageVars(search) }, (data) => pageFromGraph(data.blossomObjects, normalizeBlossomObject)) as Promise<T>
		case method === "GET" && pathname.match(/^\/blossom\/objects\/[^/]+$/) !== null: {
			const hash = decodeURIComponent(pathname.split("/")[3] ?? "")
			return graphQuery(BlossomObjectDocument, { hash }, (data) => normalizeBlossomObject(data.blossomObject)) as Promise<T>
		}
		case method === "POST" && pathname === "/blossom/objects/bulk-review":
			return graphMutation(ReviewBlossomObjectsDocument, { input: { hashes: body?.hashes ?? [], action: String(body?.action ?? "").toUpperCase(), reason: body?.reason ?? null } }, () => ({ ok: true, updated: Array.isArray(body?.hashes) ? body.hashes.length : 0 })) as Promise<T>
		case method === "GET" && pathname === "/blossom/users":
			return graphQuery(BlossomUsersDocument, { filter: blossomUserFilterVars(search), page: pageVars(search) }, (data) => pageFromGraph(data.blossomUsers, normalizeBlossomUser)) as Promise<T>
		case method === "GET" && pathname.match(/^\/blossom\/users\/[^/]+$/) !== null: {
			const pubkey = decodeURIComponent(pathname.split("/")[3] ?? "")
			return graphQuery(BlossomUserDocument, { pubkey }, (data) => ({ ...normalizeBlossomUser(data.blossomUser.user), files: data.blossomUser.files.map(normalizeBlossomObjectRecord) })) as Promise<T>
		}
		case method === "POST" && pathname === "/blossom/users/whitelist":
			return graphMutation(UpsertBlossomWhitelistDocument, { input: { pubkey: body?.pubkey, enabled: body?.enabled, storageQuotaBytes: body?.storage_quota_bytes ?? null, egressQuotaBytes: body?.egress_quota_bytes ?? null, notes: body?.notes ?? null } }, (data) => normalizeBlossomUser(data.upsertBlossomWhitelist)) as Promise<T>
		case method === "POST" && pathname.match(/^\/blossom\/users\/[^/]+\/purge$/) !== null: {
			const pubkey = decodeURIComponent(pathname.split("/")[3] ?? "")
			return graphMutation(PurgeBlossomUserDocument, { pubkey }, () => ({ ok: true, pubkey })) as Promise<T>
		}
		case method === "POST" && pathname === "/blossom/mirror":
			return graphMutation(MirrorBlossomObjectDocument, { input: { sourceURL: body?.source_url, expectedSHA256: body?.expected_sha256 } }, (data) => ({ ok: true, job_id: data.mirrorBlossomObject.jobId, status: data.mirrorBlossomObject.status })) as Promise<T>
		case method === "GET" && pathname === "/blossom/workers":
			return graphQuery(BlossomWorkersDocument, { status: emptyToNull(search.get("status")), jobType: emptyToNull(search.get("job_type")), targetHash: emptyToNull(search.get("target_hash")) }, (data) => (data.blossomWorkers.items ?? []).map((item: any) => keysToSnake(item))) as Promise<T>
		case method === "GET" && pathname === "/blossom/reports":
			return graphQuery(BlossomReportsDocument, { filter: blossomReportFilterVars(search), page: pageVars(search) }, (data) => pageFromGraph(data.blossomReports, normalizeBlossomReport)) as Promise<T>
		case method === "POST" && pathname.match(/^\/blossom\/reports\/[^/]+\/resolve$/) !== null: {
			const id = decodeURIComponent(pathname.split("/")[3] ?? "")
			return graphMutation(ResolveBlossomReportDocument, { id, input: { status: String(body?.status ?? "").toUpperCase(), note: body?.note ?? null } }, (data) => ({ ok: true, id: data.resolveBlossomReport.id, status: String(data.resolveBlossomReport.status).toLowerCase() })) as Promise<T>
		}
		case method === "GET" && pathname === "/blossom/analytics":
			return graphQuery(BlossomAnalyticsDocument, undefined, (data) => keysToSnake(data.blossomAnalytics)) as Promise<T>
		case method === "GET" && pathname === "/blossom/audit":
			return graphQuery(BlossomAuditDocument, { page: pageVars(search) }, (data) => pageFromGraph(data.blossomAudit, (item: any) => keysToSnake(item))) as Promise<T>
		default:
			return undefined
	}
}
