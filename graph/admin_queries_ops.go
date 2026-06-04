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
	"github.com/gofiber/fiber/v2"
)

func (r *Resolver) labels(ctx context.Context, filter *model.AdminLabelFilterInput, page *model.OffsetPageInput) (*model.AdminLabelPage, error) {
	query := adminPageQuery(page)
	if filter != nil {
		if value := strings.TrimSpace(stringValue(filter.Namespace)); value != "" { query.Set("namespace", value) }
		if value := strings.TrimSpace(stringValue(filter.Target)); value != "" { query.Set("target", value) }
		if value := strings.TrimSpace(stringValue(filter.Author)); value != "" { query.Set("author", value) }
		if value := strings.TrimSpace(stringValue(filter.Q)); value != "" { query.Set("q", value) }
		if filter.TargetType != nil { query.Set("target_type", strings.ToLower(filter.TargetType.String())) }
		for _, label := range filter.Labels { query.Add("label", label) }
	}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/labels", path: "/labels", query: query, handlerFunc: httphandler.LabelsList()})
	if err != nil { return nil, err }
	items, pageInfo, err := decodeRESTPage[model.AdminLabelEvent](payload)
	if err != nil { return nil, err }
	for _, item := range items {
		item.Target.Type = model.LabelTargetType(strings.ToUpper(item.Target.Type.String()))
	}
	return &model.AdminLabelPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) labelsSummary(ctx context.Context, filter *model.AdminLabelFilterInput) (*model.AdminLabelsSummary, error) {
	query := url.Values{}
	if filter != nil {
		if value := strings.TrimSpace(stringValue(filter.Namespace)); value != "" { query.Set("namespace", value) }
		if value := strings.TrimSpace(stringValue(filter.Target)); value != "" { query.Set("target", value) }
		if value := strings.TrimSpace(stringValue(filter.Author)); value != "" { query.Set("author", value) }
		if value := strings.TrimSpace(stringValue(filter.Q)); value != "" { query.Set("q", value) }
		if filter.TargetType != nil { query.Set("target_type", strings.ToLower(filter.TargetType.String())) }
		for _, label := range filter.Labels { query.Add("label", label) }
	}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/labels/summary", path: "/labels/summary", query: query, handlerFunc: httphandler.LabelsSummary()})
	if err != nil { return nil, err }
	data, err := decodeRESTAny(payload)
	if err != nil { return nil, err }
	root := data.(map[string]any)
	root["namespaces"] = renameCountArrayKey(root["namespaces"], "namespace")
	root["labels"] = renameCountArrayKey(root["labels"], "label")
	root["targetTypes"] = renameCountArrayKey(root["targetTypes"], "targetType")
	encoded, err := jsonx.Marshal(root)
	if err != nil { return nil, err }
	var out model.AdminLabelsSummary
	if err := jsonx.Unmarshal(encoded, &out); err != nil { return nil, err }
	return &out, nil
}

func (r *Resolver) blossomOverview(ctx context.Context) (*model.AdminBlossomOverview, error) { return decodeAdminModel[model.AdminBlossomOverview](ctx, http.MethodGet, "/blossom/overview", "/blossom/overview", nil, nil, httphandler.BlossomOverview()) }
func (r *Resolver) blossomPolicy(ctx context.Context) (*model.AdminBlossomPolicy, error) { return decodeAdminModel[model.AdminBlossomPolicy](ctx, http.MethodGet, "/blossom/policy", "/blossom/policy", nil, nil, httphandler.BlossomPolicy()) }

func (r *Resolver) blossomPlans(ctx context.Context) ([]*model.AdminBlossomPlan, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/blossom/plans", path: "/blossom/plans", handlerFunc: httphandler.BlossomPlans()})
	if err != nil { return nil, err }
	items, err := decodeRESTItems[model.AdminBlossomPlan](payload)
	if err != nil { return nil, err }
	for _, item := range items { item.Scope = model.BlossomPlanScope(strings.ToUpper(item.Scope.String())) }
	return items, nil
}

func (r *Resolver) blossomPlanAssignments(ctx context.Context, planID string, page *model.OffsetPageInput) (*model.AdminBlossomPlanAssignmentPage, error) {
	path := fmt.Sprintf("/blossom/plans/%s/assignments", planID)
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/blossom/plans/:id/assignments", path: path, handlerFunc: httphandler.BlossomPlanAssignments()})
	if err != nil { return nil, err }
	items, err := decodeRESTItems[model.AdminBlossomPlanAssignment](payload)
	if err != nil { return nil, err }
	window, pageInfo := paginatePointers(items, page)
	return &model.AdminBlossomPlanAssignmentPage{PageInfo: pageInfo, Items: window}, nil
}

func (r *Resolver) blossomObjects(ctx context.Context, filter *model.AdminBlossomObjectFilterInput, page *model.OffsetPageInput) (*model.AdminBlossomObjectPage, error) {
	query := adminPageQuery(page)
	if filter != nil {
		if value := strings.TrimSpace(stringValue(filter.Sha256)); value != "" { query.Set("sha256", value) }
		if value := strings.TrimSpace(stringValue(filter.MimeType)); value != "" { query.Set("mime_type", value) }
		if value := strings.TrimSpace(stringValue(filter.Extension)); value != "" { query.Set("extension", value) }
		if value := strings.TrimSpace(stringValue(filter.ReviewState)); value != "" { query.Set("review_state", value) }
		if value := strings.TrimSpace(stringValue(filter.Pubkey)); value != "" { query.Set("pubkey", value) }
		if value := strings.TrimSpace(stringValue(filter.UploaderQuery)); value != "" { query.Set("uploader_q", value) }
	}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/blossom/objects", path: "/blossom/objects", query: query, handlerFunc: httphandler.BlossomObjects()})
	if err != nil { return nil, err }
	items, pageInfo, err := decodeRESTPage[model.AdminBlossomObject](payload)
	if err != nil { return nil, err }
	return &model.AdminBlossomObjectPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) blossomObject(ctx context.Context, hash string) (*model.AdminBlossomObject, error) {
	path := fmt.Sprintf("/blossom/objects/%s", hash)
	return decodeAdminModel[model.AdminBlossomObject](ctx, http.MethodGet, "/blossom/objects/:hash", path, nil, nil, httphandler.BlossomObjectDetail())
}

func (r *Resolver) blossomUsers(ctx context.Context, filter *model.AdminBlossomUserFilterInput, page *model.OffsetPageInput) (*model.AdminBlossomUserPage, error) {
	query := adminPageQuery(page)
	if filter != nil {
		if value := strings.TrimSpace(stringValue(filter.Q)); value != "" { query.Set("q", value) }
		if value := strings.TrimSpace(stringValue(filter.SortBy)); value != "" { query.Set("sort_by", value) }
		if filter.SortDir != nil { query.Set("sort_dir", strings.ToLower(filter.SortDir.String())) }
	}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/blossom/users", path: "/blossom/users", query: query, handlerFunc: httphandler.BlossomUsers()})
	if err != nil { return nil, err }
	items, pageInfo, err := decodeRESTPage[model.AdminBlossomUser](payload)
	if err != nil { return nil, err }
	return &model.AdminBlossomUserPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) blossomUser(ctx context.Context, pubkey string) (*model.AdminBlossomUserDetail, error) {
	path := fmt.Sprintf("/blossom/users/%s", pubkey)
	return decodeAdminModel[model.AdminBlossomUserDetail](ctx, http.MethodGet, "/blossom/users/:pubkey", path, nil, nil, httphandler.BlossomUserDetail())
}

func (r *Resolver) blossomReports(ctx context.Context, filter *model.AdminBlossomReportFilterInput, page *model.OffsetPageInput) (*model.AdminBlossomReportPage, error) {
	query := adminPageQuery(page)
	if filter != nil {
		if value := strings.TrimSpace(stringValue(filter.Q)); value != "" { query.Set("q", value) }
		if value := strings.TrimSpace(stringValue(filter.ReportType)); value != "" { query.Set("report_type", value) }
		if value := strings.TrimSpace(stringValue(filter.ObjectHash)); value != "" { query.Set("object_hash", value) }
		if filter.Status != nil { query.Set("status", strings.ToLower(filter.Status.String())) }
	}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/blossom/reports", path: "/blossom/reports", query: query, handlerFunc: httphandler.BlossomReports()})
	if err != nil { return nil, err }
	items, pageInfo, err := decodeRESTPage[model.AdminBlossomReport](payload)
	if err != nil { return nil, err }
	for _, item := range items { item.Status = model.BlossomReportStatus(strings.ToUpper(item.Status.String())) }
	return &model.AdminBlossomReportPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) blossomAnalytics(ctx context.Context) (map[string]any, error) {
	value, err := decodeAdminModel[map[string]any](ctx, http.MethodGet, "/blossom/analytics", "/blossom/analytics", nil, nil, httphandler.BlossomAnalytics())
	if err != nil {
		return nil, err
	}
	return *value, nil
}

func (r *Resolver) blossomWorkers(ctx context.Context, status *string, jobType *string, targetHash *string, page *model.OffsetPageInput) (*model.AdminBlossomWorkerPage, error) {
	query := url.Values{}
	if value := strings.TrimSpace(stringValue(status)); value != "" { query.Set("status", value) }
	if value := strings.TrimSpace(stringValue(jobType)); value != "" { query.Set("job_type", value) }
	if value := strings.TrimSpace(stringValue(targetHash)); value != "" { query.Set("target_hash", value) }
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/blossom/workers", path: "/blossom/workers", query: query, handlerFunc: httphandler.BlossomWorkers()})
	if err != nil { return nil, err }
	normalized, err := normalizeRESTPayload(payload)
	if err != nil { return nil, err }
	items := make([]map[string]any, 0)
	if err := jsonx.Unmarshal(normalized, &items); err != nil { return nil, err }
	window, pageInfo := paginateMaps(items, page)
	return &model.AdminBlossomWorkerPage{PageInfo: pageInfo, Items: window}, nil
}

func (r *Resolver) blossomAudit(ctx context.Context, page *model.OffsetPageInput) (*model.AdminBlossomAuditPage, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/blossom/audit", path: "/blossom/audit", query: adminPageQuery(page), handlerFunc: httphandler.BlossomAudit()})
	if err != nil { return nil, err }
	items, pageInfo, err := decodeRESTMapPage(payload)
	if err != nil { return nil, err }
	return &model.AdminBlossomAuditPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) nip86AllowedPubkeys(ctx context.Context, q *string, page *model.OffsetPageInput) (*model.AdminNip86PubkeyPage, error) { return r.nip86PubkeysPage(ctx, q, page, "/nip86/allowed-pubkeys", httphandler.NIP86AllowedPubKeys()) }
func (r *Resolver) nip86BlockedIps(ctx context.Context, q *string, page *model.OffsetPageInput) (*model.AdminNip86IPPage, error) { return r.nip86IPsPage(ctx, q, page, "/nip86/blocked-ips", httphandler.NIP86BlockedIPs()) }
func (r *Resolver) nip86BannedEvents(ctx context.Context, q *string, page *model.OffsetPageInput) (*model.AdminNip86EventPage, error) { return r.nip86EventsPage(ctx, q, page, "/nip86/banned-events", httphandler.NIP86BannedEvents()) }

func (r *Resolver) nip86RelayMetadata(ctx context.Context) (*model.AdminNip86RelayMetadata, error) { return decodeAdminModel[model.AdminNip86RelayMetadata](ctx, http.MethodGet, "/nip86/relay-metadata", "/nip86/relay-metadata", nil, nil, httphandler.NIP86RelayMetadata()) }

func (r *Resolver) jobs(ctx context.Context, filter *model.AdminJobFilterInput, page *model.OffsetPageInput) (*model.AdminJobPage, error) {
	query := adminPageQuery(page)
	if filter != nil {
		if value := strings.TrimSpace(stringValue(filter.Queue)); value != "" { query.Set("queue", value) }
		if value := strings.TrimSpace(stringValue(filter.JobName)); value != "" { query.Set("job_name", value) }
		for _, status := range filter.Statuses { query.Add("status", status) }
	}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/jobs", path: "/jobs", query: query, handlerFunc: httphandler.JobsList()})
	if err != nil { return nil, err }
	items, pageInfo, err := decodeRESTPage[model.AdminJob](payload)
	if err != nil { return nil, err }
	return &model.AdminJobPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) job(ctx context.Context, id string, queue *string) (*model.AdminJob, error) {
	query := url.Values{}
	if value := strings.TrimSpace(stringValue(queue)); value != "" { query.Set("queue", value) }
	path := fmt.Sprintf("/jobs/%s", id)
	return decodeAdminModel[model.AdminJob](ctx, http.MethodGet, "/jobs/:jobId", path, query, nil, httphandler.JobDetail())
}

func (r *Resolver) groups(ctx context.Context) ([]*model.AdminGroup, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/groups", path: "/groups", handlerFunc: httphandler.ListGroups()})
	if err != nil { return nil, err }
	items, _, err := decodeRESTPage[model.AdminGroup](payload)
	if err != nil { return nil, err }
	return items, nil
}

func (r *Resolver) wotSummary(ctx context.Context) (*model.AdminWoTSummary, error) { return decodeAdminModel[model.AdminWoTSummary](ctx, http.MethodGet, "/wot/summary", "/wot/summary", nil, nil, httphandler.WoTSummary()) }

func decodeAdminModel[T any](ctx context.Context, method, route, path string, query url.Values, body any, handler fiber.Handler) (*T, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: method, route: route, path: path, query: query, body: body, handlerFunc: handler})
	if err != nil { return nil, err }
	return decodeRESTModel[T](payload)
}

func (r *Resolver) nip86PubkeysPage(ctx context.Context, q *string, page *model.OffsetPageInput, path string, handler fiber.Handler) (*model.AdminNip86PubkeyPage, error) {
	query := adminPageQuery(page)
	if value := strings.TrimSpace(stringValue(q)); value != "" { query.Set("q", value) }
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: path, path: path, query: query, handlerFunc: handler})
	if err != nil { return nil, err }
	items, pageInfo, err := decodeRESTPage[model.AdminNip86PubkeyRecord](payload)
	if err != nil { return nil, err }
	return &model.AdminNip86PubkeyPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) nip86IPsPage(ctx context.Context, q *string, page *model.OffsetPageInput, path string, handler fiber.Handler) (*model.AdminNip86IPPage, error) {
	query := adminPageQuery(page)
	if value := strings.TrimSpace(stringValue(q)); value != "" { query.Set("q", value) }
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: path, path: path, query: query, handlerFunc: handler})
	if err != nil { return nil, err }
	items, pageInfo, err := decodeRESTPage[model.AdminNip86IPRecord](payload)
	if err != nil { return nil, err }
	return &model.AdminNip86IPPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) nip86EventsPage(ctx context.Context, q *string, page *model.OffsetPageInput, path string, handler fiber.Handler) (*model.AdminNip86EventPage, error) {
	query := adminPageQuery(page)
	if value := strings.TrimSpace(stringValue(q)); value != "" { query.Set("q", value) }
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: path, path: path, query: query, handlerFunc: handler})
	if err != nil { return nil, err }
	items, pageInfo, err := decodeRESTPage[model.AdminNip86EventRecord](payload)
	if err != nil { return nil, err }
	return &model.AdminNip86EventPage{PageInfo: pageInfo, Items: items}, nil
}
