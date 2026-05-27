package blossom

import (
	"net/url"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	internalblossom "github.com/gabrielmoura/nostr-relay-server/internal/blossom"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gofiber/fiber/v2"
)

type mirrorRequest struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type mirrorResponse struct {
	Status string `json:"status"`
	JobID  string `json:"job_id"`
	SHA256 string `json:"sha256"`
	URL    string `json:"url"`
}

func MirrorHandler(c *fiber.Ctx) error {
	startedAt := time.Now()
	statusCode := fiber.StatusAccepted
	errorCategory := ""
	defer func() {
		observeBlossomRequest("/mirror", c.Method(), startedAt)
		if statusCode >= 400 {
			observeBlossomError("/mirror", c.Method(), statusCode, errorCategory)
		}
	}()

	if c.Method() != fiber.MethodPut {
		statusCode = fiber.StatusMethodNotAllowed
		errorCategory = "invalid_method"
		return c.Status(fiber.StatusMethodNotAllowed).SendString("Invalid request method")
	}

	var req mirrorRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		statusCode = fiber.StatusBadRequest
		errorCategory = "invalid_request"
		return c.Status(fiber.StatusBadRequest).SendString("Invalid mirror payload")
	}

	req.URL = strings.TrimSpace(req.URL)
	req.SHA256 = normalizeMirrorHash(req.SHA256)
	if err := validateMirrorRequest(req); err != nil {
		statusCode = err.(*fiber.Error).Code
		errorCategory = blossomErrorCategory(err, "invalid_request")
		return err
	}

	pubKey, err := processBlossomActionAuth(c, "upload", req.SHA256)
	if err != nil {
		status := fiber.StatusUnauthorized
		if fiberErr, ok := err.(*fiber.Error); ok {
			status = fiberErr.Code
		}
		statusCode = status
		errorCategory = blossomErrorCategory(err, "auth_invalid")
		return c.Status(status).SendString(err.Error())
	}

	service := jobcore.Default()
	if service == nil || service.Dispatcher == nil {
		statusCode = fiber.StatusServiceUnavailable
		errorCategory = "internal"
		return c.Status(fiber.StatusServiceUnavailable).SendString("Job runtime is not initialized")
	}

	id, err := service.Dispatcher.Dispatch(
		c.UserContext(),
		internalblossom.MirrorJob{
			SourceURL:      req.URL,
			ExpectedSHA256: req.SHA256,
			RequestedBy:    pubKey,
		},
		jobcore.WithQueue(config.Cfg.Jobs.DefaultQueue),
	)
	if err != nil {
		statusCode = fiber.StatusInternalServerError
		errorCategory = "internal"
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to enqueue mirror job")
	}

	return c.Status(fiber.StatusAccepted).JSON(mirrorResponse{
		Status: "queued",
		JobID:  id.String(),
		SHA256: req.SHA256,
		URL:    req.URL,
	})
}

func validateMirrorRequest(req mirrorRequest) error {
	if req.URL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "mirror url is required")
	}
	parsed, err := url.Parse(req.URL)
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		return fiber.NewError(fiber.StatusBadRequest, "mirror url must be absolute")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fiber.NewError(fiber.StatusBadRequest, "mirror url scheme must be http or https")
	}
	if len(req.SHA256) != 64 {
		return fiber.NewError(fiber.StatusBadRequest, "mirror sha256 must be 64 hex chars")
	}
	for _, r := range req.SHA256 {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return fiber.NewError(fiber.StatusBadRequest, "mirror sha256 must be lowercase hex")
		}
	}
	return nil
}

func normalizeMirrorHash(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
