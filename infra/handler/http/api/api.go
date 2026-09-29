// Package api implements the opt-in public HTTP API on the relay's external
// listener. It deliberately has no dependency on WebSocket listeners.
package api

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/nip86"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

type eventSource func(context.Context, nostr.Filter, int) ([]*nostr.Event, error)

type Handler struct {
	cfg    *config.Config
	source eventSource
	sem    chan struct{}
}

type eventsResponse struct {
	Events []*nostr.Event `json:"events"`
	Count  int            `json:"count"`
	More   bool           `json:"more"`
}

func New(cfg *config.Config) *Handler {
	return NewWithSource(cfg, func(ctx context.Context, filter nostr.Filter, offset int) ([]*nostr.Event, error) {
		queries := db.DbQueries()
		if queries == nil {
			return nil, errors.New("database is not initialized")
		}
		events, _, err := queries.QueryEventsWindow(ctx, filter, offset)
		return events, err
	})
}

func NewWithSource(cfg *config.Config, source eventSource) *Handler {
	maxConcurrent := cfg.API.MaxConcurrent
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	return &Handler{cfg: cfg, source: source, sem: make(chan struct{}, maxConcurrent)}
}

// Register installs only documented public API routes. Static routes must be
// registered before identifier routes so they are never parsed as NIP-19.
func (h *Handler) Register(app fiber.Router) {
	app.Use(h.middleware)
	app.Get("/query", h.query)
	app.Get("/count", h.count)
	app.Get("/ids/:id", h.eventByHexID)
	app.Get("/:npub/:kind", h.eventsByAuthorAndKind)
	app.Get("/:identifier", h.identifier)
}

func (h *Handler) middleware(c *fiber.Ctx) error {
	if !h.cfg.API.Enabled {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if websocketUpgrade(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	if c.Method() != fiber.MethodGet {
		return c.Status(fiber.StatusMethodNotAllowed).JSON(fiber.Map{"error": "method not allowed"})
	}

	if h.cfg.API.NIP98.Enabled {
		result, err := nip86.ValidateNIP98(nip86.AuthInput{
			Authorization: c.Get(fiber.HeaderAuthorization),
			Method:        c.Method(),
			URL:           c.BaseURL() + c.OriginalURL(),
			Body:          nil,
		}, 60)
		if err != nil || !h.allowed(result.PubKey) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
	}

	select {
	case h.sem <- struct{}{}:
		defer func() { <-h.sem }()
		return c.Next()
	default:
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "api is busy"})
	}
}

func (h *Handler) allowed(pubkey string) bool {
	allowed := h.cfg.API.NIP98.AllowedPubkeys
	if len(allowed) == 0 {
		return true
	}
	for _, candidate := range allowed {
		if strings.EqualFold(strings.TrimSpace(candidate), pubkey) {
			return true
		}
	}
	return false
}

func (h *Handler) query(c *fiber.Ctx) error {
	filter, offset, err := parseFilter(c, h.maxLimit())
	if err != nil {
		return badRequest(c, err)
	}
	response, err := h.pageWithExcluded(c.UserContext(), filter, offset, excludedTags(c))
	if err != nil {
		return internalError(c, err)
	}
	return c.JSON(response)
}

func (h *Handler) count(c *fiber.Ctx) error {
	filter, _, err := parseFilter(c, h.maxLimit())
	if err != nil {
		return badRequest(c, err)
	}
	count, err := h.visibleCountWithExcluded(c.UserContext(), filter, excludedTags(c))
	if err != nil {
		return internalError(c, err)
	}
	return c.JSON(fiber.Map{"count": count})
}

func (h *Handler) eventByHexID(c *fiber.Ctx) error {
	id := c.Params("id")
	if !isHexID(id) {
		return badRequest(c, errors.New("id must be a 64-character hexadecimal event id"))
	}
	return h.respondFilter(c, nostr.Filter{IDs: []string{id}, Limit: 1})
}

func (h *Handler) identifier(c *fiber.Ctx) error {
	value := c.Params("identifier")
	filter, err := identifierFilter(value)
	if err != nil {
		return badRequest(c, err)
	}
	return h.respondFilter(c, filter)
}

func (h *Handler) eventsByAuthorAndKind(c *fiber.Ctx) error {
	pubkey, err := decodeNPub(c.Params("npub"))
	if err != nil {
		return badRequest(c, err)
	}
	kind, err := strconv.Atoi(c.Params("kind"))
	if err != nil || kind < 0 {
		return badRequest(c, errors.New("kind must be a non-negative integer"))
	}
	filter, offset, err := parseFilter(c, h.maxLimit())
	if err != nil {
		return badRequest(c, err)
	}
	filter.Authors = []string{pubkey}
	filter.Kinds = []int{kind}
	response, err := h.pageWithExcluded(c.UserContext(), filter, offset, excludedTags(c))
	if err != nil {
		return internalError(c, err)
	}
	return c.JSON(response)
}

func (h *Handler) respondFilter(c *fiber.Ctx, filter nostr.Filter) error {
	filter.Limit = 1
	response, err := h.page(c.UserContext(), filter, 0)
	if err != nil {
		return internalError(c, err)
	}
	return c.JSON(response)
}

func (h *Handler) maxLimit() int {
	if h.cfg.API.MaxLimit > 0 {
		return h.cfg.API.MaxLimit
	}
	return h.cfg.Security.Limits.MaxLimit
}

func badRequest(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
}

func internalError(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "query failed"})
}

func websocketUpgrade(c *fiber.Ctx) bool {
	return strings.Contains(strings.ToLower(c.Get(fiber.HeaderConnection)), "upgrade") || strings.EqualFold(c.Get(fiber.HeaderUpgrade), "websocket")
}

func identifierFilter(value string) (nostr.Filter, error) {
	value = strings.TrimSpace(value)
	switch {
	case strings.HasPrefix(value, "npub"):
		pubkey, err := decodeNPub(value)
		if err != nil {
			return nostr.Filter{}, err
		}
		return nostr.Filter{Authors: []string{pubkey}, Kinds: []int{0}, Limit: 1}, nil
	case strings.HasPrefix(value, "note"), strings.HasPrefix(value, "nevent"):
		id, err := decodeEventID(value)
		if err != nil {
			return nostr.Filter{}, err
		}
		return nostr.Filter{IDs: []string{id}, Limit: 1}, nil
	case strings.HasPrefix(value, "naddr"):
		prefix, decoded, err := nip19.Decode(value)
		if err != nil || prefix != "naddr" {
			return nostr.Filter{}, errors.New("invalid naddr")
		}
		pointer, ok := decoded.(nostr.EntityPointer)
		if !ok || !isHexID(pointer.PublicKey) || pointer.Kind < 0 {
			return nostr.Filter{}, errors.New("invalid naddr")
		}
		return nostr.Filter{Authors: []string{pointer.PublicKey}, Kinds: []int{pointer.Kind}, Tags: nostr.TagMap{"d": []string{pointer.Identifier}}, Limit: 1}, nil
	default:
		return nostr.Filter{}, errors.New("identifier must be npub, note, nevent, or naddr")
	}
}

func decodeNPub(value string) (string, error) {
	prefix, decoded, err := nip19.Decode(strings.TrimSpace(value))
	if err != nil || prefix != "npub" {
		return "", errors.New("invalid npub")
	}
	pubkey, ok := decoded.(string)
	if !ok || !isHexID(pubkey) {
		return "", errors.New("invalid npub")
	}
	return pubkey, nil
}

func decodeEventID(value string) (string, error) {
	prefix, decoded, err := nip19.Decode(strings.TrimSpace(value))
	if err != nil {
		return "", errors.New("invalid event identifier")
	}
	switch prefix {
	case "note":
		id, ok := decoded.(string)
		if !ok || !isHexID(id) {
			return "", errors.New("invalid note")
		}
		return id, nil
	case "nevent":
		pointer, ok := decoded.(nostr.EventPointer)
		if !ok || !isHexID(pointer.ID) {
			return "", errors.New("invalid nevent")
		}
		return pointer.ID, nil
	default:
		return "", errors.New("invalid event identifier")
	}
}

func isHexID(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !(char >= '0' && char <= '9') && !(char >= 'a' && char <= 'f') && !(char >= 'A' && char <= 'F') {
			return false
		}
	}
	return true
}
