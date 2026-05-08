package http

import (
	"fmt"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

type adminLabelTargetResponse struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	RelayHint string `json:"relay_hint,omitempty"`
}

type adminLabelEventResponse struct {
	ID         string                   `json:"id"`
	Pubkey     string                   `json:"pubkey"`
	AuthorNpub string                   `json:"author_npub,omitempty"`
	CreatedAt  int64                    `json:"created_at"`
	Kind       int                      `json:"kind"`
	Content    string                   `json:"content"`
	Namespace  string                   `json:"namespace"`
	Labels     []string                 `json:"labels"`
	Target     adminLabelTargetResponse `json:"target"`
	Tags       nostr.Tags               `json:"tags"`
}

type adminLabelCountResponse struct {
	Count int64 `json:"count"`

	Namespace  string `json:"namespace,omitempty"`
	Label      string `json:"label,omitempty"`
	TargetType string `json:"target_type,omitempty"`
}

type adminLabelsSummaryResponse struct {
	TotalEvents  int64                     `json:"total_events"`
	TotalTargets int64                     `json:"total_targets"`
	Namespaces   []adminLabelCountResponse `json:"namespaces"`
	Labels       []adminLabelCountResponse `json:"labels"`
	TargetTypes  []adminLabelCountResponse `json:"target_types"`
}

type adminCreateLabelTargetRequest struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	RelayHint string `json:"relay_hint"`
}

type adminCreateLabelRequest struct {
	Namespace string                        `json:"namespace"`
	Labels    []string                      `json:"labels"`
	Comment   string                        `json:"comment"`
	Target    adminCreateLabelTargetRequest `json:"target"`
}

type adminCreateLabelResponse struct {
	Event  *nostr.Event `json:"event"`
	Stored bool         `json:"stored"`
}

func LabelsList() fiber.Handler {
	return func(c *fiber.Ctx) error {
		filters, err := adminLabelFiltersFromRequest(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		limit := adminLimit(c)
		offset := adminOffset(c)
		items, total, err := db.DbQueries.GetLabels(c.UserContext(), filters, limit, offset)
		if err != nil {
			return internalServerError(c, err)
		}

		response := make([]adminLabelEventResponse, 0, len(items))
		for _, item := range items {
			response = append(response, toAdminLabelEventResponse(item))
		}

		return c.JSON(newAdminPage(response, int(total), limit, offset))
	}
}

func LabelsSummary() fiber.Handler {
	return func(c *fiber.Ctx) error {
		filters, err := adminLabelFiltersFromRequest(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		summary, err := db.DbQueries.GetLabelsSummary(c.UserContext(), filters)
		if err != nil {
			return internalServerError(c, err)
		}

		return c.JSON(adminLabelsSummaryResponse{
			TotalEvents:  summary.TotalEvents,
			TotalTargets: summary.TotalTargets,
			Namespaces:   toAdminLabelCountResponses(summary.Namespaces, "namespace"),
			Labels:       toAdminLabelCountResponses(summary.Labels, "label"),
			TargetTypes:  toAdminLabelCountResponses(summary.TargetTypes, "target_type"),
		})
	}
}

func CreateLabel() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req adminCreateLabelRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		evt, err := buildAdminLabelEvent(req, time.Now().UTC())
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		if err := evt.Sign(config.Cfg.RelayInformation.PrivKey); err != nil {
			return internalServerError(c, fmt.Errorf("sign label event: %w", err))
		}

		if err := db.DbQueries.InsertEvent(c.UserContext(), evt); err != nil {
			if err == dbmodel.ErrDupEvent {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "label event already exists"})
			}
			return internalServerError(c, err)
		}

		return c.Status(fiber.StatusCreated).JSON(adminCreateLabelResponse{Event: evt, Stored: true})
	}
}

func adminLabelFiltersFromRequest(c *fiber.Ctx) (dbmodel.AdminLabelFilters, error) {
	filters := dbmodel.AdminLabelFilters{
		Namespace:  strings.TrimSpace(c.Query("namespace")),
		TargetType: strings.TrimSpace(c.Query("target_type")),
		Target:     strings.TrimSpace(c.Query("target")),
		Author:     strings.TrimSpace(c.Query("author")),
		Query:      strings.TrimSpace(c.Query("q")),
	}
	for _, rawLabel := range c.Request().URI().QueryArgs().PeekMulti("label") {
		label := strings.TrimSpace(string(rawLabel))
		if label == "" {
			continue
		}
		filters.Labels = append(filters.Labels, label)
	}

	if filters.TargetType != "" {
		normalizedType, err := normalizeAdminLabelTargetType(filters.TargetType)
		if err != nil {
			return dbmodel.AdminLabelFilters{}, err
		}
		filters.TargetType = normalizedType
	}

	if filters.Author != "" {
		normalizedAuthor, err := normalizePublicKey(filters.Author)
		if err != nil {
			return dbmodel.AdminLabelFilters{}, err
		}
		filters.Author = normalizedAuthor
	}

	if filters.Target != "" && filters.TargetType != "" {
		normalizedTarget, err := normalizeAdminLabelTargetValue(filters.TargetType, filters.Target)
		if err != nil {
			return dbmodel.AdminLabelFilters{}, err
		}
		filters.Target = normalizedTarget
	}

	return filters, nil
}

func buildAdminLabelEvent(req adminCreateLabelRequest, now time.Time) (*nostr.Event, error) {
	if strings.TrimSpace(config.Cfg.RelayInformation.PrivKey) == "" {
		return nil, fmt.Errorf("relay_information.priv_key is required")
	}

	namespace := strings.TrimSpace(req.Namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if len(req.Labels) == 0 {
		return nil, fmt.Errorf("at least one label is required")
	}

	targetType, err := normalizeAdminLabelTargetType(req.Target.Type)
	if err != nil {
		return nil, err
	}

	targetValue := strings.TrimSpace(req.Target.Value)
	if targetValue == "" {
		return nil, fmt.Errorf("target value is required")
	}
	targetValue, err = normalizeAdminLabelTargetValue(targetType, targetValue)
	if err != nil {
		return nil, err
	}

	tags := nostr.Tags{{"L", namespace}}
	seenLabels := make(map[string]struct{}, len(req.Labels))
	for _, value := range req.Labels {
		label := strings.ToLower(strings.TrimSpace(value))
		if label == "" {
			continue
		}
		if _, ok := seenLabels[label]; ok {
			continue
		}
		seenLabels[label] = struct{}{}
		tags = append(tags, nostr.Tag{"l", label, namespace})
	}
	if len(tags) == 1 {
		return nil, fmt.Errorf("at least one valid label is required")
	}

	targetTag := nostr.Tag{adminLabelTagName(targetType), targetValue}
	relayHint := strings.TrimSpace(req.Target.RelayHint)
	if relayHint != "" && (targetType == "event" || targetType == "pubkey") {
		targetTag = append(targetTag, relayHint)
	}
	tags = append(tags, targetTag)

	return &nostr.Event{
		CreatedAt: nostr.Timestamp(now.Unix()),
		Kind:      1985,
		Tags:      tags,
		Content:   strings.TrimSpace(req.Comment),
	}, nil
}

func normalizeAdminLabelTargetValue(targetType string, targetValue string) (string, error) {
	switch targetType {
	case "pubkey":
		return normalizePublicKey(targetValue)
	case "event":
		return normalizeEventID(targetValue)
	case "address":
		return normalizeAddressValue(targetValue)
	default:
		return targetValue, nil
	}
}

func normalizeAdminLabelTargetType(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "event", "pubkey", "address", "reference", "topic":
		return strings.ToLower(strings.TrimSpace(value)), nil
	default:
		return "", fmt.Errorf("invalid target type")
	}
}

func adminLabelTagName(targetType string) string {
	switch targetType {
	case "event":
		return "e"
	case "pubkey":
		return "p"
	case "address":
		return "a"
	case "reference":
		return "r"
	default:
		return "t"
	}
}

func toAdminLabelEventResponse(item dbmodel.AdminLabelRecord) adminLabelEventResponse {
	return adminLabelEventResponse{
		ID:         item.Event.ID,
		Pubkey:     item.Event.PubKey,
		AuthorNpub: npubFromPublicKey(item.Event.PubKey),
		CreatedAt:  int64(item.Event.CreatedAt),
		Kind:       item.Event.Kind,
		Content:    item.Event.Content,
		Namespace:  item.Namespace,
		Labels:     item.Labels,
		Target: adminLabelTargetResponse{
			Type:      item.Target.Type,
			Value:     item.Target.Value,
			RelayHint: item.Target.RelayHint,
		},
		Tags: item.Event.Tags,
	}
}

func toAdminLabelCountResponses(items []dbmodel.AdminLabelCount, kind string) []adminLabelCountResponse {
	response := make([]adminLabelCountResponse, 0, len(items))
	for _, item := range items {
		count := adminLabelCountResponse{Count: item.Count}
		switch kind {
		case "namespace":
			count.Namespace = item.Key
		case "label":
			count.Label = item.Key
		case "target_type":
			count.TargetType = item.Key
		}
		response = append(response, count)
	}
	return response
}
