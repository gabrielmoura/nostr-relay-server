package http

import (
	"strconv"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	internalblossom "github.com/gabrielmoura/nostr-relay-server/internal/blossom"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	"github.com/gofiber/fiber/v2"
)

func BlossomPolicy() fiber.Handler {
	return func(c *fiber.Ctx) error {
		policy, ok, err := db.DbQueries.GetBlossomServerPolicy(c.UserContext())
		if err != nil {
			return internalServerError(c, err)
		}
		if !ok {
			policy = defaultBlossomServerPolicy()
		}
		return c.JSON(mapBlossomPolicyResponse(policy))
	}
}

func BlossomPolicyUpsert() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req adminBlossomPolicyRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		mode := strings.TrimSpace(req.Mode)
		switch mode {
		case "mandatory_review", "enabled_users", "free":
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid blossom policy mode"})
		}
		policy := dbmodel.BlossomServerPolicy{Mode: mode, UpdatedBy: blossomActorPubkey()}
		if err := db.DbQueries.UpsertBlossomServerPolicy(c.UserContext(), policy); err != nil {
			return internalServerError(c, err)
		}
		stored, _, err := db.DbQueries.GetBlossomServerPolicy(c.UserContext())
		if err != nil {
			return internalServerError(c, err)
		}
		_ = internalblossom.RecordAudit(c.UserContext(), blossomActorPubkey(), "policy.update", "policy", "blossom", map[string]string{"mode": mode})
		return c.JSON(mapBlossomPolicyResponse(stored))
	}
}

func BlossomPlans() fiber.Handler {
	return func(c *fiber.Ctx) error {
		items, err := db.DbQueries.ListBlossomPlans(c.UserContext(), strings.TrimSpace(c.Query("scope")))
		if err != nil {
			return internalServerError(c, err)
		}
		response := make([]adminBlossomPlanResponse, 0, len(items))
		for _, item := range items {
			response = append(response, mapBlossomPlanResponse(item))
		}
		return c.JSON(fiber.Map{"items": response})
	}
}

func BlossomPlanUpsert() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req adminBlossomPlanRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		if strings.TrimSpace(req.ID) == "" || strings.TrimSpace(req.Name) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "plan id and name are required"})
		}
		scope := strings.TrimSpace(req.Scope)
		if scope != "free" && scope != "enabled_users" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid blossom plan scope"})
		}
		plan := dbmodel.BlossomPlan{
			ID:          strings.TrimSpace(req.ID),
			Name:        strings.TrimSpace(req.Name),
			Scope:       scope,
			Description: strings.TrimSpace(req.Description),
			IsDefault:   req.IsDefault,
			UpdatedBy:   blossomActorPubkey(),
		}
		if req.StorageQuotaBytes != nil {
			plan.StorageQuotaBytes.Valid = true
			plan.StorageQuotaBytes.Int64 = *req.StorageQuotaBytes
		}
		if req.EgressQuotaBytes != nil {
			plan.EgressQuotaBytes.Valid = true
			plan.EgressQuotaBytes.Int64 = *req.EgressQuotaBytes
		}
		if err := db.DbQueries.UpsertBlossomPlan(c.UserContext(), plan); err != nil {
			return internalServerError(c, err)
		}
		_ = internalblossom.RecordAudit(c.UserContext(), blossomActorPubkey(), "plan.upsert", "plan", plan.ID, map[string]string{"scope": plan.Scope})
		items, err := db.DbQueries.ListBlossomPlans(c.UserContext(), "")
		if err != nil {
			return internalServerError(c, err)
		}
		for _, item := range items {
			if item.ID == plan.ID {
				return c.JSON(mapBlossomPlanResponse(item))
			}
		}
		return c.JSON(mapBlossomPlanResponse(plan))
	}
}

func BlossomPlanDelete() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := strings.TrimSpace(c.Params("id"))
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid plan id"})
		}
		if err := db.DbQueries.DeleteBlossomPlan(c.UserContext(), id); err != nil {
			return internalServerError(c, err)
		}
		_ = internalblossom.RecordAudit(c.UserContext(), blossomActorPubkey(), "plan.delete", "plan", id, nil)
		return c.JSON(fiber.Map{"ok": true, "id": id})
	}
}

func BlossomPlanAssign() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := strings.TrimSpace(c.Params("id"))
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid plan id"})
		}
		var req adminBlossomPlanAssignRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		pubkey, err := normalizePublicKey(req.Pubkey)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if err := db.DbQueries.AssignPlanToUser(c.UserContext(), id, pubkey, blossomActorPubkey()); err != nil {
			return internalServerError(c, err)
		}
		_ = internalblossom.RecordAudit(c.UserContext(), blossomActorPubkey(), "plan.assign", "pubkey", pubkey, map[string]string{"plan_id": id})
		return c.JSON(adminBlossomPlanAssignResponse{OK: true, PlanID: id, Pubkey: pubkey})
	}
}

func BlossomPlanAssignments() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := strings.TrimSpace(c.Params("id"))
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid plan id"})
		}
		items, err := db.DbQueries.ListPlanAssignments(c.UserContext(), id)
		if err != nil {
			return internalServerError(c, err)
		}
		response := make([]adminBlossomPlanAssignmentResponse, 0, len(items))
		for _, item := range items {
			response = append(response, mapBlossomPlanAssignmentResponse(item))
		}
		return c.JSON(fiber.Map{"items": response})
	}
}

func BlossomPlanUnassign() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := strings.TrimSpace(c.Params("id"))
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid plan id"})
		}
		pubkey, err := normalizePublicKey(c.Params("pubkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if err := db.DbQueries.UnassignPlanFromUser(c.UserContext(), id, pubkey); err != nil {
			return internalServerError(c, err)
		}
		_ = internalblossom.RecordAudit(c.UserContext(), blossomActorPubkey(), "plan.unassign", "pubkey", pubkey, map[string]string{"plan_id": id})
		return c.JSON(fiber.Map{"ok": true, "plan_id": id, "pubkey": pubkey})
	}
}

func BlossomUsers() fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := adminLimit(c)
		offset := adminOffset(c)
		sortBy := strings.TrimSpace(c.Query("sort_by"))
		sortDir := strings.TrimSpace(c.Query("sort_dir"))
		items, total, err := db.DbQueries.ListBlossomUsers(c.UserContext(), strings.TrimSpace(c.Query("q")), limit, offset, sortBy, sortDir)
		if err != nil {
			return internalServerError(c, err)
		}
		response := make([]adminBlossomUserResponse, 0, len(items))
		for _, item := range items {
			response = append(response, mapBlossomUserResponse(item))
		}
		return c.JSON(newAdminPage(response, int(total), limit, offset))
	}
}

func BlossomUserDetail() fiber.Handler {
	return func(c *fiber.Ctx) error {
		pubkey := strings.TrimSpace(c.Params("pubkey"))
		usage, ok, err := db.DbQueries.GetBlossomUserUsage(c.UserContext(), pubkey)
		if err != nil {
			return internalServerError(c, err)
		}
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "blossom user not found"})
		}
		files, _, err := db.DbQueries.ListBlossomObjects(c.UserContext(), dbmodel.BlossomObjectFilters{Pubkey: pubkey}, maxAdminLimit, 0)
		if err != nil {
			return internalServerError(c, err)
		}
		response := adminBlossomUserDetailResponse{adminBlossomUserResponse: mapBlossomUserResponse(usage)}
		response.Files = make([]adminBlossomObjectResponse, 0, len(files))
		for _, item := range files {
			response.Files = append(response.Files, mapBlossomObjectResponse(item))
		}
		return c.JSON(response)
	}
}

func BlossomWhitelistUpsert() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req adminBlossomWhitelistRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		pubkey, err := normalizePublicKey(req.Pubkey)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if err := db.DbQueries.UpsertBlossomPubkeyQuota(c.UserContext(), dbmodel.UpsertBlossomPubkeyQuotaParams{
			Pubkey:            pubkey,
			Enabled:           req.Enabled,
			StorageQuotaBytes: req.StorageQuotaBytes,
			EgressQuotaBytes:  req.EgressQuotaBytes,
			Notes:             strings.TrimSpace(req.Notes),
			CreatedBy:         blossomActorPubkey(),
		}); err != nil {
			return internalServerError(c, err)
		}
		_ = internalblossom.RecordAudit(c.UserContext(), blossomActorPubkey(), "quota.update", "pubkey", pubkey, map[string]string{"enabled": strconv.FormatBool(req.Enabled)})
		return c.JSON(req)
	}
}

func BlossomUserPurge() fiber.Handler {
	return func(c *fiber.Ctx) error {
		pubkey, err := normalizePublicKey(c.Params("pubkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		files, _, err := db.DbQueries.ListBlossomObjects(c.UserContext(), dbmodel.BlossomObjectFilters{Pubkey: pubkey}, maxAdminLimit, 0)
		if err != nil {
			return internalServerError(c, err)
		}
		hashes := make([]string, 0, len(files))
		for _, item := range files {
			hashes = append(hashes, item.Hash)
		}
		if _, err := hardDeleteBlossomObjects(c.UserContext(), hashes, "purge user data"); err != nil {
			return internalServerError(c, err)
		}
		_ = internalblossom.RecordAudit(c.UserContext(), blossomActorPubkey(), "user.purge", "pubkey", pubkey, map[string]string{"objects": strconv.Itoa(len(hashes))})
		return c.JSON(fiber.Map{"ok": true, "pubkey": pubkey})
	}
}

func BlossomMirror() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req adminBlossomMirrorRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		service := jobcore.Default()
		if service == nil || service.Dispatcher == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "job runtime is not initialized"})
		}
		id, err := service.Dispatcher.Dispatch(c.UserContext(), internalblossom.MirrorJob{SourceURL: strings.TrimSpace(req.SourceURL), ExpectedSHA256: strings.TrimSpace(req.ExpectedSHA256), RequestedBy: blossomActorPubkey()}, jobcore.WithQueue(config.Cfg.Jobs.DefaultQueue))
		if err != nil {
			return internalServerError(c, err)
		}
		_ = internalblossom.RecordAudit(c.UserContext(), blossomActorPubkey(), "mirror.enqueue", "job", id.String(), map[string]string{"source_url": req.SourceURL})
		return c.Status(fiber.StatusAccepted).JSON(adminBlossomMirrorResponse{OK: true, JobID: id.String(), Status: "queued"})
	}
}

func BlossomWorkers() fiber.Handler {
	return func(c *fiber.Ctx) error {
		service := jobcore.Default()
		if service == nil || service.Monitor == nil {
			return c.JSON([]adminBlossomWorkerResponse{})
		}
		snapshots, err := listAdminJobSnapshots(c.UserContext(), service.Monitor, "")
		if err != nil {
			return internalServerError(c, err)
		}
		statusFilter := strings.TrimSpace(c.Query("status"))
		jobTypeFilter := strings.TrimSpace(c.Query("job_type"))
		targetHashFilter := strings.TrimSpace(c.Query("target_hash"))
		items := make([]adminBlossomWorkerResponse, 0)
		for _, snapshot := range snapshots {
			if !strings.HasPrefix(snapshot.Name, "blossom.") {
				continue
			}
			if statusFilter != "" && snapshot.Status.String() != statusFilter {
				continue
			}
			item := blossomWorkerDetail(snapshot)
			if jobTypeFilter != "" && item.JobType != jobTypeFilter {
				continue
			}
			if targetHashFilter != "" && item.TargetHash != targetHashFilter {
				continue
			}
			items = append(items, item)
		}
		return c.JSON(items)
	}
}

func BlossomReports() fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := adminLimit(c)
		offset := adminOffset(c)
		items, total, err := db.DbQueries.ListBlossomReports(c.UserContext(), dbmodel.BlossomReportFilters{
			Query:      strings.TrimSpace(c.Query("q")),
			ReportType: strings.TrimSpace(c.Query("report_type")),
			Status:     strings.TrimSpace(c.Query("status")),
			ObjectHash: strings.TrimSpace(c.Query("object_hash")),
		}, limit, offset)
		if err != nil {
			return internalServerError(c, err)
		}
		response := make([]adminBlossomReportResponse, 0, len(items))
		for _, item := range items {
			response = append(response, mapBlossomReportResponse(item))
		}
		return c.JSON(newAdminPage(response, int(total), limit, offset))
	}
}

func BlossomResolveReport() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.ParseInt(strings.TrimSpace(c.Params("id")), 10, 64)
		if err != nil || id <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid report id"})
		}
		var req adminBlossomReportResolveRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		status := strings.TrimSpace(req.Status)
		if status != "dismissed" && status != "actioned" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid report resolution status"})
		}
		if err := db.DbQueries.ResolveBlossomReport(c.UserContext(), id, status, blossomActorPubkey(), strings.TrimSpace(req.Note)); err != nil {
			return internalServerError(c, err)
		}
		_ = internalblossom.RecordAudit(c.UserContext(), blossomActorPubkey(), "report.resolve", "report", strconv.FormatInt(id, 10), map[string]string{"status": status})
		return c.JSON(fiber.Map{"ok": true, "id": id, "status": status})
	}
}

func BlossomAnalytics() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reportSummary, err := db.DbQueries.GetBlossomReportSummary(c.UserContext(), dbmodel.BlossomReportFilters{})
		if err != nil {
			return internalServerError(c, err)
		}
		byMIME, err := db.DbQueries.GetBlossomObjectCountsByMIME(c.UserContext())
		if err != nil {
			return internalServerError(c, err)
		}
		byReview, err := db.DbQueries.GetBlossomObjectCountsByReviewState(c.UserContext())
		if err != nil {
			return internalServerError(c, err)
		}
		service := jobcore.Default()
		response := adminBlossomAnalyticsResponse{}
		response.Reports.Total = reportSummary.TotalReports
		response.Reports.Open = reportSummary.OpenReports
		response.Reports.Resolved = reportSummary.ResolvedReports
		for _, item := range reportSummary.ByType {
			response.Reports.ByType = append(response.Reports.ByType, adminCountByValue{Name: item.Name, Count: item.Count})
		}
		for _, item := range reportSummary.ByStatus {
			response.Reports.ByStatus = append(response.Reports.ByStatus, adminCountByValue{Name: item.Name, Count: item.Count})
		}
		for _, item := range byMIME {
			response.Objects.ByMime = append(response.Objects.ByMime, adminCountByValue{Name: item.Name, Count: item.Count})
		}
		for _, item := range byReview {
			response.Objects.ByReviewState = append(response.Objects.ByReviewState, adminCountByValue{Name: item.Name, Count: item.Count})
		}
		if service != nil && service.Monitor != nil {
			snapshots, err := listAdminJobSnapshots(c.UserContext(), service.Monitor, "")
			if err != nil {
				return internalServerError(c, err)
			}
			statusCounts := make(map[string]int64)
			typeCounts := make(map[string]int64)
			for _, snapshot := range snapshots {
				if !strings.HasPrefix(snapshot.Name, "blossom.") {
					continue
				}
				statusCounts[snapshot.Status.String()]++
				typeCounts[snapshot.Name]++
			}
			for key, value := range statusCounts {
				response.Workers.ByStatus = append(response.Workers.ByStatus, adminCountByValue{Name: key, Count: value})
			}
			for key, value := range typeCounts {
				response.Workers.ByType = append(response.Workers.ByType, adminCountByValue{Name: key, Count: value})
			}
		}
		return c.JSON(response)
	}
}

func BlossomAudit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := adminLimit(c)
		offset := adminOffset(c)
		items, total, err := db.DbQueries.ListBlossomAudit(c.UserContext(), strings.TrimSpace(c.Query("q")), limit, offset)
		if err != nil {
			return internalServerError(c, err)
		}
		response := make([]adminBlossomAuditResponse, 0, len(items))
		for _, item := range items {
			response = append(response, mapBlossomAuditResponse(item))
		}
		return c.JSON(newAdminPage(response, int(total), limit, offset))
	}
}
