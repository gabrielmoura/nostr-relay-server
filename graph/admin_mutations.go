package graph

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gabrielmoura/nostr-relay-server/graph/model"
	storedb "github.com/gabrielmoura/nostr-relay-server/infra/db"
	httphandler "github.com/gabrielmoura/nostr-relay-server/infra/handler/http"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	jsonx "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gofiber/fiber/v2"
)

func (r *Resolver) disconnectConnection(ctx context.Context, wsid string, reason *string) (*model.MutationAck, error) {
	body := map[string]any{"reason": strings.TrimSpace(stringValue(reason))}
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/connections/:wsid/disconnect", path: fmt.Sprintf("/connections/%s/disconnect", wsid), body: body, handlerFunc: httphandler.DisconnectConnection()})
	if err != nil { return nil, err }
	return adminMutationAck(true, &wsid, "connection disconnected"), nil
}

func (r *Resolver) banUser(ctx context.Context, pubkey string, input model.AdminBanUserInput) (*model.AdminBanStatus, error) {
	body := map[string]any{"reason": strings.TrimSpace(stringValue(input.Reason)), "related_ids": input.RelatedIds}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/users/:pubkey/ban", path: fmt.Sprintf("/users/%s/ban", pubkey), body: body, handlerFunc: httphandler.BanUser()})
	if err != nil { return nil, err }
	return decodeRESTModel[model.AdminBanStatus](payload)
}

func (r *Resolver) unbanUser(ctx context.Context, pubkey string) (*model.AdminBanStatus, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodDelete, route: "/users/:pubkey/ban", path: fmt.Sprintf("/users/%s/ban", pubkey), handlerFunc: httphandler.UnbanUser()})
	if err != nil { return nil, err }
	status, err := decodeRESTModel[model.AdminBanStatus](payload)
	if err != nil { return nil, err }
	return status, nil
}

func (r *Resolver) upsertNip05(ctx context.Context, input model.AdminNip05UpsertInput) (*model.AdminNip05Identity, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/nip05", path: "/nip05", body: input, handlerFunc: httphandler.NIP05Upsert()})
	if err != nil { return nil, err }
	return decodeRESTModel[model.AdminNip05Identity](payload)
}

func (r *Resolver) deleteNip05(ctx context.Context, name string) (*model.MutationAck, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodDelete, route: "/nip05/:name", path: fmt.Sprintf("/nip05/%s", name), handlerFunc: httphandler.NIP05Delete()})
	if err != nil { return nil, err }
	return adminMutationAck(true, &name, "identity deleted"), nil
}

func (r *Resolver) importEvents(ctx context.Context, files []*graphql.Upload) (*model.AdminImportEventsPayload, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/events/import", path: "/events/import", uploads: files, handlerFunc: httphandler.ImportEventsJSONL()})
	if err != nil { return nil, err }
	return decodeRESTModel[model.AdminImportEventsPayload](payload)
}

func (r *Resolver) fetchEventFromRelays(ctx context.Context, id string, input *model.AdminFetchEventInput) (*model.AdminFetchEventPayload, error) {
	body := map[string]any{}
	if input != nil { body["relays"] = input.Relays }
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/events/:id/fetch", path: fmt.Sprintf("/events/%s/fetch", id), body: body, handlerFunc: httphandler.FetchEventFromRelays()})
	if err != nil { return nil, err }
	return decodeRESTModel[model.AdminFetchEventPayload](payload)
}

func (r *Resolver) createLabel(ctx context.Context, input model.AdminCreateLabelInput) (*model.AdminCreateLabelPayload, error) {
	body := map[string]any{"namespace": input.Namespace, "labels": input.Labels, "comment": stringValue(input.Comment), "target": map[string]any{"type": strings.ToLower(input.Target.Type.String()), "value": input.Target.Value, "relay_hint": strings.TrimSpace(stringValue(input.Target.RelayHint))}}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/labels", path: "/labels", body: body, handlerFunc: httphandler.CreateLabel()})
	if err != nil { return nil, err }
	return decodeRESTModel[model.AdminCreateLabelPayload](payload)
}

func (r *Resolver) upsertBlossomPolicy(ctx context.Context, input model.AdminBlossomPolicyInput) (*model.AdminBlossomPolicy, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPut, route: "/blossom/policy", path: "/blossom/policy", body: map[string]any{"mode": input.Mode}, handlerFunc: httphandler.BlossomPolicyUpsert()})
	if err != nil { return nil, err }
	return decodeRESTModel[model.AdminBlossomPolicy](payload)
}

func (r *Resolver) upsertBlossomPlan(ctx context.Context, input model.AdminBlossomPlanInput) (*model.AdminBlossomPlan, error) {
	body := map[string]any{"id": input.ID, "name": input.Name, "scope": strings.ToLower(input.Scope.String()), "storage_quota_bytes": input.StorageQuotaBytes, "egress_quota_bytes": input.EgressQuotaBytes, "description": stringValue(input.Description), "is_default": input.IsDefault}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPut, route: "/blossom/plans", path: "/blossom/plans", body: body, handlerFunc: httphandler.BlossomPlanUpsert()})
	if err != nil { return nil, err }
	out, err := decodeRESTModel[model.AdminBlossomPlan](payload)
	if err != nil { return nil, err }
	out.Scope = model.BlossomPlanScope(strings.ToUpper(out.Scope.String()))
	return out, nil
}

func (r *Resolver) deleteBlossomPlan(ctx context.Context, id string) (*model.MutationAck, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodDelete, route: "/blossom/plans/:id", path: fmt.Sprintf("/blossom/plans/%s", id), handlerFunc: httphandler.BlossomPlanDelete()})
	if err != nil { return nil, err }
	return adminMutationAck(true, &id, "plan deleted"), nil
}

func (r *Resolver) assignBlossomPlan(ctx context.Context, planID string, pubkey string) (*model.MutationAck, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/blossom/plans/:id/assign", path: fmt.Sprintf("/blossom/plans/%s/assign", planID), body: map[string]any{"pubkey": pubkey}, handlerFunc: httphandler.BlossomPlanAssign()})
	if err != nil { return nil, err }
	return adminMutationAck(true, &planID, "plan assigned"), nil
}

func (r *Resolver) unassignBlossomPlan(ctx context.Context, planID string, pubkey string) (*model.MutationAck, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodDelete, route: "/blossom/plans/:id/assign/:pubkey", path: fmt.Sprintf("/blossom/plans/%s/assign/%s", planID, pubkey), handlerFunc: httphandler.BlossomPlanUnassign()})
	if err != nil { return nil, err }
	return adminMutationAck(true, &planID, "plan unassigned"), nil
}

func (r *Resolver) reviewBlossomObjects(ctx context.Context, input model.AdminBlossomBulkReviewInput) (*model.MutationAck, error) {
	body := map[string]any{"hashes": input.Hashes, "action": strings.ToLower(input.Action.String()), "reason": strings.TrimSpace(stringValue(input.Reason))}
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/blossom/objects/bulk-review", path: "/blossom/objects/bulk-review", body: body, handlerFunc: httphandler.BlossomBulkReview()})
	if err != nil { return nil, err }
	return adminMutationAck(true, nil, "objects updated"), nil
}

func (r *Resolver) upsertBlossomWhitelist(ctx context.Context, input model.AdminBlossomWhitelistInput) (*model.AdminBlossomUser, error) {
	body := map[string]any{"pubkey": input.Pubkey, "enabled": input.Enabled, "storage_quota_bytes": input.StorageQuotaBytes, "egress_quota_bytes": input.EgressQuotaBytes, "notes": stringValue(input.Notes)}
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/blossom/users/whitelist", path: "/blossom/users/whitelist", body: body, handlerFunc: httphandler.BlossomWhitelistUpsert()})
	if err != nil { return nil, err }
	detail, err := r.blossomUser(ctx, input.Pubkey)
	if err != nil { return nil, err }
	return detail.User, nil
}

func (r *Resolver) purgeBlossomUser(ctx context.Context, pubkey string) (*model.MutationAck, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/blossom/users/:pubkey/purge", path: fmt.Sprintf("/blossom/users/%s/purge", pubkey), handlerFunc: httphandler.BlossomUserPurge()})
	if err != nil { return nil, err }
	return adminMutationAck(true, &pubkey, "user purged"), nil
}

func (r *Resolver) mirrorBlossomObject(ctx context.Context, input model.AdminBlossomMirrorInput) (*model.AdminAsyncJob, error) {
	body := map[string]any{"source_url": input.SourceURL, "expected_sha256": input.ExpectedSha256}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/blossom/mirror", path: "/blossom/mirror", body: body, handlerFunc: httphandler.BlossomMirror()})
	if err != nil { return nil, err }
	job, err := decodeRESTModel[model.AdminAsyncJob](payload)
	if err != nil { return nil, err }
	return job, nil
}

func (r *Resolver) resolveBlossomReport(ctx context.Context, id string, input model.AdminBlossomResolveReportInput) (*model.AdminBlossomReport, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/blossom/reports/:id/resolve", path: fmt.Sprintf("/blossom/reports/%s/resolve", id), body: map[string]any{"status": strings.ToLower(input.Status.String()), "note": stringValue(input.Note)}, handlerFunc: httphandler.BlossomResolveReport()})
	if err != nil { return nil, err }
	reportID, err := strconv.ParseInt(id, 10, 64)
	if err != nil { return nil, err }
	record, ok, err := db.DbQueries.GetBlossomReportByID(ctx, reportID)
	if err != nil { return nil, err }
	if !ok { return nil, fmt.Errorf("resolved report %s not found", id) }
	return blossomReportRowToModel(record), nil
}

func (r *Resolver) createNip86AllowedPubkey(ctx context.Context, pubkey string, input *model.AdminNip86ReasonInput) (*model.MutationAck, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/nip86/allowed-pubkeys/:pubkey", path: fmt.Sprintf("/nip86/allowed-pubkeys/%s", pubkey), body: map[string]any{"reason": strings.TrimSpace(stringValue(reasonInput(input)))}, handlerFunc: httphandler.NIP86CreateAllowedPubKey()})
	if err != nil { return nil, err }
	return adminMutationAck(true, &pubkey, "pubkey allowed"), nil
}

func (r *Resolver) deleteNip86AllowedPubkey(ctx context.Context, pubkey string) (*model.MutationAck, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodDelete, route: "/nip86/allowed-pubkeys/:pubkey", path: fmt.Sprintf("/nip86/allowed-pubkeys/%s", pubkey), handlerFunc: httphandler.NIP86DeleteAllowedPubKey()})
	if err != nil { return nil, err }
	return adminMutationAck(true, &pubkey, "pubkey removed from allowlist"), nil
}

func (r *Resolver) createNip86BlockedIP(ctx context.Context, ip string, input *model.AdminNip86ReasonInput) (*model.MutationAck, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/nip86/blocked-ips/:ip", path: fmt.Sprintf("/nip86/blocked-ips/%s", ip), body: map[string]any{"reason": strings.TrimSpace(stringValue(reasonInput(input)))}, handlerFunc: httphandler.NIP86CreateBlockedIP()})
	if err != nil { return nil, err }
	return adminMutationAck(true, &ip, "ip blocked"), nil
}

func (r *Resolver) deleteNip86BlockedIP(ctx context.Context, ip string) (*model.MutationAck, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodDelete, route: "/nip86/blocked-ips/:ip", path: fmt.Sprintf("/nip86/blocked-ips/%s", ip), handlerFunc: httphandler.NIP86DeleteBlockedIP()})
	if err != nil { return nil, err }
	return adminMutationAck(true, &ip, "ip unblocked"), nil
}

func (r *Resolver) createNip86BannedEvent(ctx context.Context, eventID string, input *model.AdminNip86ReasonInput) (*model.MutationAck, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/nip86/banned-events/:id", path: fmt.Sprintf("/nip86/banned-events/%s", eventID), body: map[string]any{"reason": strings.TrimSpace(stringValue(reasonInput(input)))}, handlerFunc: httphandler.NIP86CreateBannedEvent()})
	if err != nil { return nil, err }
	return adminMutationAck(true, &eventID, "event banned"), nil
}

func (r *Resolver) deleteNip86BannedEvent(ctx context.Context, eventID string) (*model.MutationAck, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodDelete, route: "/nip86/banned-events/:id", path: fmt.Sprintf("/nip86/banned-events/%s", eventID), handlerFunc: httphandler.NIP86DeleteBannedEvent()})
	if err != nil { return nil, err }
	return adminMutationAck(true, &eventID, "event unbanned"), nil
}

func (r *Resolver) updateNip86RelayMetadata(ctx context.Context, input model.AdminNip86RelayMetadataInput) (*model.AdminNip86RelayMetadata, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/nip86/relay-metadata", path: "/nip86/relay-metadata", body: input, handlerFunc: httphandler.NIP86UpdateRelayMetadata()})
	if err != nil { return nil, err }
	return decodeRESTModel[model.AdminNip86RelayMetadata](payload)
}

func (r *Resolver) startNegentropySync(ctx context.Context, input model.AdminNegentropySyncInput) (*model.AdminAsyncJob, error) {
	body := map[string]any{"remote": input.Remote, "direction": stringValue(input.Direction), "filter": input.Filter, "timeout": input.TimeoutSeconds}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/sync/negentropy", path: "/sync/negentropy", body: body, handlerFunc: httphandler.NegentropySync()})
	if err != nil { return nil, err }
	return decodeRESTModel[model.AdminAsyncJob](payload)
}

func (r *Resolver) downloadEvents(ctx context.Context, input model.AdminDownloadEventsInput) (*model.AdminAsyncJob, error) {
	body := map[string]any{"relays": input.Relays, "public_key": stringValue(input.PublicKey), "kinds": input.Kinds, "filter": input.Filter, "timeout": input.TimeoutSeconds}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/events/download", path: "/events/download", body: body, handlerFunc: httphandler.DownloadEvents()})
	if err != nil { return nil, err }
	return decodeRESTModel[model.AdminAsyncJob](payload)
}

func (r *Resolver) retryJob(ctx context.Context, id string, queue *string) (*model.MutationAck, error) { return r.jobMutation(ctx, "/jobs/:jobId/retry", fmt.Sprintf("/jobs/%s/retry", id), id, queue, httphandler.RetryJob(), "job retried") }
func (r *Resolver) cancelJob(ctx context.Context, id string, queue *string) (*model.MutationAck, error) { return r.jobMutation(ctx, "/jobs/:jobId/cancel", fmt.Sprintf("/jobs/%s/cancel", id), id, queue, httphandler.CancelJob(), "job canceled") }
func (r *Resolver) resumeJob(ctx context.Context, id string, queue *string) (*model.MutationAck, error) { return r.jobMutation(ctx, "/jobs/:jobId/resume", fmt.Sprintf("/jobs/%s/resume", id), id, queue, httphandler.ResumeJob(), "job resumed") }

func (r *Resolver) deleteJobsHistory(ctx context.Context, input model.AdminDeleteJobsHistoryInput) (*model.MutationAck, error) {
	query := url.Values{}
	query.Set("job_name", input.JobName)
	for _, status := range input.Statuses { query.Add("status", status) }
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodDelete, route: "/jobs", path: "/jobs", query: query, handlerFunc: httphandler.DeleteJobsHistory()})
	if err != nil { return nil, err }
	deleted := struct{ Deleted int32 `json:"deleted"` }{}
	normalized, err := normalizeRESTPayload(payload)
	if err != nil { return nil, err }
	if err := jsonx.Unmarshal(normalized, &deleted); err != nil { return nil, err }
	msg := fmt.Sprintf("deleted %d jobs", deleted.Deleted)
	return adminMutationAck(true, nil, msg), nil
}

func (r *Resolver) addTrustedPubkey(ctx context.Context, pubkey string) (*model.AdminWoTSummary, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: "/wot/trusted", path: "/wot/trusted", body: map[string]any{"pubkey": pubkey}, handlerFunc: httphandler.AddTrustedPubkey()})
	if err != nil { return nil, err }
	return r.wotSummary(ctx)
}

func (r *Resolver) removeTrustedPubkey(ctx context.Context, pubkey string) (*model.AdminWoTSummary, error) {
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodDelete, route: "/wot/trusted/:pubkey", path: fmt.Sprintf("/wot/trusted/%s", pubkey), handlerFunc: httphandler.RemoveTrustedPubkey()})
	if err != nil { return nil, err }
	return r.wotSummary(ctx)
}

func (r *Resolver) jobMutation(ctx context.Context, route, path, id string, queue *string, handler fiber.Handler, message string) (*model.MutationAck, error) {
	body := map[string]any{}
	if value := strings.TrimSpace(stringValue(queue)); value != "" { body["queue"] = value }
	_, err := executeAdminRequest(ctx, adminRequest{method: http.MethodPost, route: route, path: path, body: body, handlerFunc: handler})
	if err != nil { return nil, err }
	return adminMutationAck(true, &id, message), nil
}

func reasonInput(input *model.AdminNip86ReasonInput) *string {
	if input == nil {
		return nil
	}
	return input.Reason
}

func blossomReportRowToModel(item storedb.BlossomReportRow) *model.AdminBlossomReport {
	modelItem := &model.AdminBlossomReport{ID: strconv.FormatInt(item.ID, 10), EventID: item.EventID, ObjectHash: item.ObjectHash, ReporterPubkey: item.ReporterPubkey, Status: model.BlossomReportStatus(strings.ToUpper(item.Status))}
	if item.TargetEventID != "" { modelItem.TargetEventID = &item.TargetEventID }
	if item.TargetPubkey != "" { modelItem.TargetPubkey = &item.TargetPubkey }
	if item.ReportType != "" { modelItem.ReportType = &item.ReportType }
	if item.Reason != "" { modelItem.Reason = &item.Reason }
	if item.ResolvedBy != "" { modelItem.ResolvedBy = &item.ResolvedBy }
	if item.ResolvedNote != "" { modelItem.ResolvedNote = &item.ResolvedNote }
	if item.CreatedAt.Valid { value := item.CreatedAt.Time.UTC().Format(time.RFC3339); modelItem.CreatedAt = &value }
	if item.ResolvedAt.Valid { value := item.ResolvedAt.Time.UTC().Format(time.RFC3339); modelItem.ResolvedAt = &value }
	return modelItem
}
