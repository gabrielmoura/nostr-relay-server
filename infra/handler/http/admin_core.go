package http

import (
	"errors"
	"regexp"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	promdto "github.com/prometheus/client_model/go"
)

const (
	defaultAdminLimit = 100
	maxAdminLimit     = 250
	adminMaxJSONBody  = 4 << 20
)

var errAdminEventNotFoundOnRelays = errors.New("event not found on provided relays")

var defaultAdminFetchRelays = []string{
	"wss://relay.damus.io",
	"wss://relay.primal.net",
	"wss://nos.lol",
	"wss://relay.nostr.band",
	"wss://nostr.mom",
}

var imageURLPattern = regexp.MustCompile(`https?://[^\s"'<>]+\.(?:png|jpg|jpeg|gif|webp|avif)`)

var (
	eventIDPattern   = regexp.MustCompile(`^[a-f0-9]{64}$`)
	publicKeyPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

func AdminTokenMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg == nil || !cfg.AdminAPIRequiresToken() {
			return c.Next()
		}
		if c.Get("X-Admin-Token") != cfg.AdminToken {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid admin token"})
		}
		return c.Next()
	}
}

func AdminIndex() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString("Admin Interface")
	}
}

func adminLimit(c *fiber.Ctx) int {
	limit := c.QueryInt("limit", defaultAdminLimit)
	if limit <= 0 {
		return defaultAdminLimit
	}
	if limit > maxAdminLimit {
		return maxAdminLimit
	}
	return limit
}

func adminOffset(c *fiber.Ctx) int {
	offset := c.QueryInt("offset", 0)
	if offset < 0 {
		return 0
	}
	return offset
}

func parseAdminJSONBody(c *fiber.Ctx, out any) error {
	body := c.Body()
	if len(body) == 0 {
		return nil
	}
	if len(body) > adminMaxJSONBody {
		return errors.New("request body too large")
	}

	return json.Unmarshal(body, out)
}

func internalServerError(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
}

func formatUnix(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func counterValue(counter prometheus.Counter) float64 {
	metric := &promdto.Metric{}
	if err := counter.Write(metric); err != nil {
		return 0
	}
	if metric.Counter == nil {
		return 0
	}
	return metric.Counter.GetValue()
}
