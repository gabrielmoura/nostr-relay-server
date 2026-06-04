package graph

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/graph/model"
	httphandler "github.com/gabrielmoura/nostr-relay-server/infra/handler/http"
	jsonx "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
)

func (r *Resolver) events(ctx context.Context, filter *model.AdminEventSearchInput, page *model.OffsetPageInput) (*model.AdminEventPage, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/events/search", path: "/events/search", query: buildEventSearchQuery(filter, page), handlerFunc: httphandler.SearchEvents()})
	if err != nil {
		return nil, err
	}
	items, pageInfo, err := decodeRESTPage[model.NostrEvent](payload)
	if err != nil {
		return nil, err
	}
	return &model.AdminEventPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) eventAggregates(ctx context.Context, filter *model.AdminEventSearchInput) (*model.AdminEventAggregates, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/events/search/aggregates", path: "/events/search/aggregates", query: buildEventSearchQuery(filter, nil), handlerFunc: httphandler.SearchEventsAggregates()})
	if err != nil {
		return nil, err
	}
	return decodeRESTModel[model.AdminEventAggregates](payload)
}

func (r *Resolver) eventTimeline(ctx context.Context, filter *model.AdminEventSearchInput, bucket *model.TimelineBucket) (*model.AdminTimeline, error) {
	query := buildEventSearchQuery(filter, nil)
	if bucket != nil {
		query.Set("bucket", strings.ToLower(bucket.String()))
	}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/events/search/timeline", path: "/events/search/timeline", query: query, handlerFunc: httphandler.SearchEventsTimeline()})
	if err != nil {
		return nil, err
	}
	timeline, err := decodeRESTModel[model.AdminTimeline](payload)
	if err != nil {
		return nil, err
	}
	if timeline.Bucket != "" {
		timeline.Bucket = model.TimelineBucket(strings.ToUpper(timeline.Bucket.String()))
	}
	return timeline, nil
}

func (r *Resolver) eventDetail(ctx context.Context, id string) (*model.AdminEventDetail, error) {
	path := fmt.Sprintf("/events/%s", id)
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/events/:id", path: path, handlerFunc: httphandler.EventDetail()})
	if err != nil {
		return nil, err
	}
	return decodeRESTModel[model.AdminEventDetail](payload)
}

func (r *Resolver) eventReports(ctx context.Context, id string, page *model.OffsetPageInput) (*model.AdminEventReportPage, error) {
	path := fmt.Sprintf("/events/%s/reports", id)
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/events/:id/reports", path: path, query: adminPageQuery(page), handlerFunc: httphandler.EventReports()})
	if err != nil {
		return nil, err
	}
	items, pageInfo, err := decodeRESTPage[model.AdminEventReport](payload)
	if err != nil {
		return nil, err
	}
	return &model.AdminEventReportPage{PageInfo: pageInfo, Items: items}, nil
}

func buildReportedEventsQuery(filter *model.AdminReportedEventFilterInput, page *model.OffsetPageInput) map[string]string {
	query := map[string]string{}
	if page != nil && page.Limit != nil {
		query["limit"] = fmt.Sprintf("%d", *page.Limit)
	}
	if page != nil && page.Offset != nil {
		query["offset"] = fmt.Sprintf("%d", *page.Offset)
	}
	if filter == nil {
		return query
	}
	if value := strings.TrimSpace(stringValue(filter.Q)); value != "" {
		query["q"] = value
	}
	if len(filter.Types) > 0 {
		query["type"] = filter.Types[0]
	}
	return query
}

func (r *Resolver) reportedEvents(ctx context.Context, filter *model.AdminReportedEventFilterInput, page *model.OffsetPageInput) (*model.AdminReportedEventPage, error) {
	qmap := buildReportedEventsQuery(filter, page)
	query := make(url.Values)
	for key, value := range qmap {
		query.Set(key, value)
	}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/events/reported", path: "/events/reported", query: query, handlerFunc: httphandler.ReportedEvents()})
	if err != nil {
		return nil, err
	}
	items, pageInfo, err := decodeRESTPage[model.AdminReportedEvent](payload)
	if err != nil {
		return nil, err
	}
	return &model.AdminReportedEventPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) reportedEventsSummary(ctx context.Context, filter *model.AdminReportedEventFilterInput) (*model.AdminReportedEventsSummary, error) {
	qmap := buildReportedEventsQuery(filter, nil)
	query := make(url.Values)
	for key, value := range qmap {
		query.Set(key, value)
	}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/events/reported/summary", path: "/events/reported/summary", query: query, handlerFunc: httphandler.ReportedEventsSummary()})
	if err != nil {
		return nil, err
	}
	data, err := decodeRESTAny(payload)
	if err != nil {
		return nil, err
	}
	root := data.(map[string]any)
	root["timeline"] = renameCountArrayKey(root["timeline"], "bucket")
	root["topTargets"] = renameCountArrayKey(root["topTargets"], "targetEventId")
	remarshaled, err := jsonx.Marshal(root)
	if err != nil {
		return nil, err
	}
	var out model.AdminReportedEventsSummary
	if err := jsonx.Unmarshal(remarshaled, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func renameCountArrayKey(value any, from string) any {
	items, ok := value.([]any)
	if !ok {
		return value
	}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if current, exists := entry[from]; exists {
			entry["name"] = current
			delete(entry, from)
		}
	}
	return items
}
