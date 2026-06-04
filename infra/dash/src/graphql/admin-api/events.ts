import {
  CreateLabelDocument,
  EventAggregatesDocument,
  EventDetailDocument,
  EventReportsDocument,
  EventTimelineDocument,
  EventsDocument,
  FetchEventFromRelaysDocument,
  LabelsDocument,
  LabelsSummaryDocument,
  ReportedEventsDocument,
  ReportedEventsSummaryDocument,
} from "@/graphql/generated/operations"

import { bucketVar, eventFilterVars, graphMutation, graphQuery, keysToSnake, labelTargetTypeVar, labelsFilterVars, mapEventRecord, mapLabelEvent, pageFromGraph, pageVars, reportedFilterVars, type RequestContext } from "./core"

export async function handleEventsRequest<T>(ctx: RequestContext): Promise<T | undefined> {
  const { method, pathname, search, body } = ctx

  switch (true) {
    case method === "GET" && pathname === "/events/search":
      return graphQuery(EventsDocument, { filter: eventFilterVars(search), page: pageVars(search) }, (data) => pageFromGraph(data.events, mapEventRecord)) as Promise<T>
    case method === "GET" && pathname === "/events/search/aggregates":
			return graphQuery(EventAggregatesDocument, { filter: eventFilterVars(search) }, (data) => ({ total: data.eventAggregates.total, kinds: data.eventAggregates.kinds.map((item: any) => ({ kind: item.kind, count: item.count })), top_authors: data.eventAggregates.topAuthors.map((item: any) => ({ pubkey: item.pubkey, display_name: item.displayName, count: item.count })), top_tags: data.eventAggregates.topTags.map((item: any) => ({ tag: item.tag, count: item.count })), trends: keysToSnake(data.eventAggregates.trends) })) as Promise<T>
    case method === "GET" && pathname === "/events/search/timeline":
      return graphQuery(EventTimelineDocument, { filter: eventFilterVars(search), bucket: bucketVar(search.get("bucket")) }, (data) => ({ bucket: String(data.eventTimeline.bucket).toLowerCase(), points: data.eventTimeline.points })) as Promise<T>
    case method === "GET" && pathname.match(/^\/events\/[^/]+$/) !== null: {
      const eventId = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphQuery(EventDetailDocument, { id: eventId }, (data) => ({ event: mapEventRecord(data.eventDetail.event), identifiers: keysToSnake(data.eventDetail.identifiers), author: keysToSnake(data.eventDetail.author), hashtags: data.eventDetail.hashtags, image_urls: data.eventDetail.imageUrls })) as Promise<T>
    }
    case method === "POST" && pathname.match(/^\/events\/[^/]+\/fetch$/) !== null: {
      const eventId = decodeURIComponent(pathname.split("/")[2] ?? "")
      return graphMutation(FetchEventFromRelaysDocument, { id: eventId, input: { relays: body?.relays ?? [] } }, (data) => ({ event_id: data.fetchEventFromRelays.eventId, source_relay: data.fetchEventFromRelays.sourceRelay, found: data.fetchEventFromRelays.found, persisted: data.fetchEventFromRelays.persisted, relays_tried: data.fetchEventFromRelays.relaysTried, relay_results: keysToSnake(data.fetchEventFromRelays.relayResults), message: data.fetchEventFromRelays.message })) as Promise<T>
    }
    case method === "GET" && pathname.match(/^\/events\/[^/]+\/reports$/) !== null: {
      const eventId = decodeURIComponent(pathname.split("/")[2] ?? "")
			return graphQuery(EventReportsDocument, { id: eventId, page: pageVars(search) }, (data) => ({ items: data.eventReports.items.map((item: any) => keysToSnake(item)), total: data.eventReports.pageInfo.total })) as Promise<T>
    }
    case method === "GET" && pathname === "/events/reported":
      return graphQuery(ReportedEventsDocument, { filter: reportedFilterVars(search), page: pageVars(search) }, (data) => pageFromGraph(data.reportedEvents, (item) => ({ ...keysToSnake(item), target_event: item.targetEvent ? mapEventRecord(item.targetEvent) : undefined }))) as Promise<T>
    case method === "GET" && pathname === "/events/reported/summary":
			return graphQuery(ReportedEventsSummaryDocument, { filter: reportedFilterVars(search) }, (data) => ({ total_events: data.reportedEventsSummary.totalEvents, total_reports: data.reportedEventsSummary.totalReports, unique_target_authors: data.reportedEventsSummary.uniqueTargetAuthors, timeline: data.reportedEventsSummary.timeline.map((item: any) => ({ bucket: item.name, count: item.count })), report_types: data.reportedEventsSummary.reportTypes.map((item: any) => ({ name: item.name, count: item.count })), top_authors: data.reportedEventsSummary.topAuthors.map((item: any) => ({ pubkey: item.pubkey, display_name: item.displayName, count: item.count })), top_targets: data.reportedEventsSummary.topTargets.map((item: any) => ({ target_event_id: item.targetEventId, count: item.count })) })) as Promise<T>
    case method === "GET" && pathname === "/labels":
      return graphQuery(LabelsDocument, { filter: labelsFilterVars(search), page: pageVars(search) }, (data) => pageFromGraph(data.labels, mapLabelEvent)) as Promise<T>
    case method === "GET" && pathname === "/labels/summary":
			return graphQuery(LabelsSummaryDocument, { filter: labelsFilterVars(search) }, (data) => ({ total_events: data.labelsSummary.totalEvents, total_targets: data.labelsSummary.totalTargets, namespaces: data.labelsSummary.namespaces.map((item: any) => ({ namespace: item.name, count: item.count })), labels: data.labelsSummary.labels.map((item: any) => ({ label: item.name, count: item.count })), target_types: data.labelsSummary.targetTypes.map((item: any) => ({ target_type: String(item.name).toLowerCase(), count: item.count })) })) as Promise<T>
    case method === "POST" && pathname === "/labels":
      return graphMutation(CreateLabelDocument, { input: { namespace: body?.namespace, labels: body?.labels ?? [], comment: body?.comment ?? null, target: { type: labelTargetTypeVar(body?.target?.type ?? null), value: body?.target?.value, relayHint: body?.target?.relay_hint ?? null } } }, (data) => ({ event: mapEventRecord(data.createLabel.event), stored: data.createLabel.stored })) as Promise<T>
    default:
      return undefined
  }
}
