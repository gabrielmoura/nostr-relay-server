package blossom

import (
	stdbase64 "encoding/base64"
	"github.com/emmansun/base64"
	"github.com/gabrielmoura/nostr-relay-server/config"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	errors2 "github.com/gabrielmoura/nostr-relay-server/internal/errors"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const prefixAuthBlossomNostr = "Nostr "

// processAuth processes the Authorization header to extract Nostr event tags and public key.
func processAuth(c *fiber.Ctx) (nostr.Tags, string, error) {
	//if config.Cfg.Ws.Auth {
	authHeader := c.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, prefixAuthBlossomNostr) {
		return nil, "", errors2.ErrorAuthHeaderRequired
	}
	token := strings.TrimPrefix(authHeader, prefixAuthBlossomNostr)

	decodedBytes, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		log.Logger.Error("Decode error", zap.Error(err))
		return nil, "", errors2.ErrorDecodeAuthorization
	}

	var event nostr.Event
	if err := json.Unmarshal(decodedBytes, &event); err != nil {
		log.Logger.Error("Unmarshal error", zap.Error(err))
		return nil, "", errors2.ErrorUnmarshalAuthorization
	}

	if event.Kind != nostr.KindBlobs {
		return nil, "", errors2.ErrorInvalidEventKind
	}

	if ok, err := event.CheckSignature(); !ok || err != nil {
		return nil, "", errors2.ErrorInvalidSignature
	}

	return event.Tags, event.PubKey, nil
}

func ternaryString(condition string, fallback string) string {
	if condition != "" {
		return condition
	}
	return fallback
}

func normalizeBlobID(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 64 {
		return trimmed
	}
	if idx := strings.Index(trimmed, "."); idx == 64 {
		return trimmed[:idx]
	}
	return trimmed
}

func evaluateUploadPolicy(policy dbmodel.BlossomServerPolicy, quota *dbmodel.BlossomPubkeyQuota, usedBytes int64, uploadSize int64) (reviewState string, blocked bool, blockedReason string, err error) {
	mode := strings.TrimSpace(policy.Mode)
	if mode == "" {
		mode = "free"
	}
	switch mode {
	case "enabled_users":
		if quota == nil || !quota.Enabled {
			return "", false, "", fiber.NewError(fiber.StatusForbidden, "uploader is not enabled for Blossom uploads")
		}
	case "mandatory_review", "free":
	default:
		return "", false, "", fiber.NewError(fiber.StatusBadRequest, "invalid blossom policy mode")
	}
	limit := effectiveStorageQuota(policy, quota)
	if limit != nil && usedBytes+uploadSize > *limit {
		return "", false, "", fiber.NewError(fiber.StatusForbidden, "storage quota exceeded")
	}
	if mode == "mandatory_review" {
		return "pending_review", true, "pending manual review", nil
	}
	return "ready", false, "", nil
}

func effectiveStorageQuota(policy dbmodel.BlossomServerPolicy, quota *dbmodel.BlossomPubkeyQuota) *int64 {
	if quota != nil && quota.StorageQuotaBytes.Valid {
		value := quota.StorageQuotaBytes.Int64
		return &value
	}
	if strings.TrimSpace(policy.Mode) == "enabled_users" {
		if policy.EnabledUserDefaultStorageQuotaBytes.Valid {
			value := policy.EnabledUserDefaultStorageQuotaBytes.Int64
			return &value
		}
		return nil
	}
	if policy.DefaultStorageQuotaBytes.Valid {
		value := policy.DefaultStorageQuotaBytes.Int64
		return &value
	}
	return nil
}

func timeFromUnix(createdAt nostr.Timestamp) time.Time {
	if createdAt <= 0 {
		return time.Now().UTC()
	}
	return time.Unix(int64(createdAt), 0).UTC()
}

func mediaURLWithExtension(hash string, extension string) string {
	base := strings.TrimRight(config.Cfg.Store.MediaPath, "/") + "/" + hash
	trimmed := strings.TrimSpace(extension)
	if trimmed == "" {
		return base
	}
	if !strings.HasPrefix(trimmed, ".") {
		trimmed = "." + trimmed
	}
	return base + trimmed
}

func processBlossomActionAuth(c *fiber.Ctx, action string, expectedHash string) (string, error) {
	authHeader := strings.TrimSpace(c.Get("Authorization"))
	if authHeader == "" || !strings.HasPrefix(authHeader, prefixAuthBlossomNostr) {
		return "", errors2.ErrorAuthHeaderRequired
	}

	event, err := decodeBlossomAuthEvent(strings.TrimPrefix(authHeader, prefixAuthBlossomNostr))
	if err != nil {
		return "", err
	}
	if event.Kind != 24242 {
		return "", errors2.ErrorInvalidEventKind
	}
	if ok, err := event.CheckSignature(); !ok || err != nil {
		return "", errors2.ErrorInvalidSignature
	}
	if event.CreatedAt > nostr.Now() {
		return "", fiber.NewError(fiber.StatusUnauthorized, "authorization created_at is in the future")
	}
	if strings.TrimSpace(event.Content) == "" {
		return "", fiber.NewError(fiber.StatusUnauthorized, "authorization content is required")
	}

	if !hasBlossomAuthAction(event.Tags, action) {
		return "", fiber.NewError(fiber.StatusUnauthorized, "authorization action tag mismatch")
	}
	if err := validateBlossomAuthExpiration(event.Tags); err != nil {
		return "", err
	}
	if err := validateBlossomAuthServer(c, event.Tags); err != nil {
		return "", err
	}
	if err := validateBlossomAuthHash(event.Tags, expectedHash); err != nil {
		return "", err
	}

	return event.PubKey, nil
}

func decodeBlossomAuthEvent(token string) (nostr.Event, error) {
	decodedBytes, err := stdbase64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		decodedBytes, err = stdbase64.URLEncoding.DecodeString(strings.TrimSpace(token))
	}
	if err != nil {
		log.Logger.Error("Decode error", zap.Error(err))
		return nostr.Event{}, errors2.ErrorDecodeAuthorization
	}

	var event nostr.Event
	if err := json.Unmarshal(decodedBytes, &event); err != nil {
		log.Logger.Error("Unmarshal error", zap.Error(err))
		return nostr.Event{}, errors2.ErrorUnmarshalAuthorization
	}
	return event, nil
}

func hasBlossomAuthAction(tags nostr.Tags, action string) bool {
	for _, tag := range tags.GetAll([]string{"t", ""}) {
		if strings.EqualFold(strings.TrimSpace(tag.Value()), action) {
			return true
		}
	}
	return false
}

func validateBlossomAuthExpiration(tags nostr.Tags) error {
	expirationTag := tags.GetFirst([]string{"expiration", ""})
	if expirationTag == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "authorization expiration tag is required")
	}

	expiresAt, err := strconv.ParseInt(strings.TrimSpace(expirationTag.Value()), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "authorization expiration tag is invalid")
	}
	if expiresAt <= time.Now().Unix() {
		return fiber.NewError(fiber.StatusUnauthorized, "authorization token expired")
	}
	return nil
}

func validateBlossomAuthHash(tags nostr.Tags, expectedHash string) error {
	expectedHash = strings.ToLower(strings.TrimSpace(expectedHash))
	if expectedHash == "" {
		return nil
	}
	for _, tag := range tags.GetAll([]string{"x", ""}) {
		if strings.EqualFold(strings.TrimSpace(tag.Value()), expectedHash) {
			return nil
		}
	}
	return fiber.NewError(fiber.StatusUnauthorized, "authorization hash tag mismatch")
}

func validateBlossomAuthServer(c *fiber.Ctx, tags nostr.Tags) error {
	serverTags := tags.GetAll([]string{"server", ""})
	if len(serverTags) == 0 {
		return nil
	}

	host := requestHostname(c)
	for _, tag := range serverTags {
		if strings.EqualFold(strings.TrimSpace(tag.Value()), host) {
			return nil
		}
	}
	return fiber.NewError(fiber.StatusUnauthorized, "authorization server tag mismatch")
}

func requestHostname(c *fiber.Ctx) string {
	parsed, err := url.Parse(c.BaseURL())
	if err == nil && parsed.Host != "" {
		return strings.ToLower(parsed.Hostname())
	}
	if host, _, err := net.SplitHostPort(c.Hostname()); err == nil {
		return strings.ToLower(host)
	}
	return strings.ToLower(c.Hostname())
}
