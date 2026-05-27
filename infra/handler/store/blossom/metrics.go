package blossom

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	errors2 "github.com/gabrielmoura/nostr-relay-server/internal/errors"
	"github.com/gofiber/fiber/v2"
)

func observeBlossomRequest(route string, method string, startedAt time.Time) {
	metrics.NostrBlossomHTTPRequestTotal.WithLabelValues(route, method).Inc()
	metrics.NostrBlossomHTTPRequestDurationSeconds.WithLabelValues(route, method).Observe(time.Since(startedAt).Seconds())
}

func observeBlossomError(route string, method string, status int, category string) {
	metrics.NostrBlossomHTTPErrorsTotal.WithLabelValues(route, method, category, strconv.Itoa(status)).Inc()
}

func blossomErrorCategory(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	if errors.Is(err, errors2.ErrorAuthHeaderRequired) {
		return "auth_missing"
	}
	if errors.Is(err, errors2.ErrorDecodeAuthorization) || errors.Is(err, errors2.ErrorUnmarshalAuthorization) {
		return "auth_invalid"
	}
	if errors.Is(err, errors2.ErrorInvalidSignature) {
		return "auth_signature_invalid"
	}
	if errors.Is(err, errors2.ErrorInvalidEventKind) {
		return "auth_invalid"
	}
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		message := strings.ToLower(strings.TrimSpace(fiberErr.Message))
		switch {
		case strings.Contains(message, "quota"):
			return "quota_exceeded"
		case strings.Contains(message, "not enabled") || strings.Contains(message, "policy"):
			return "policy_denied"
		case strings.Contains(message, "file type"):
			return "mime_rejected"
		case strings.Contains(message, "hash"):
			return "hash_mismatch"
		case strings.Contains(message, "expired") || strings.Contains(message, "action tag mismatch") || strings.Contains(message, "server tag mismatch"):
			return "auth_invalid"
		case fiberErr.Code == fiber.StatusNotFound:
			return "not_found"
		case fiberErr.Code == fiber.StatusRequestedRangeNotSatisfiable:
			return "range_invalid"
		}
	}
	return fallback
}
