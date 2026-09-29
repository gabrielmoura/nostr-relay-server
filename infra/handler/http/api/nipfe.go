package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/event"
	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
	"github.com/gabrielmoura/nostr-relay-server/internal/nip86"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

// NIPFE handles exactly one NIP-01 frame and never creates a subscription.
// Its newline-delimited output makes every response frame independently
// decodable by HTTP clients and proxies.
func NIPFE(cfg *config.Config) fiber.Handler {
	maxConcurrent := cfg.API.NIPFE.MaxConcurrent
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	sem := make(chan struct{}, maxConcurrent)
	return func(c *fiber.Ctx) error {
		if !cfg.API.NIPFE.Enabled {
			return c.Next()
		}
		if c.Method() != fiber.MethodPost {
			return c.Next()
		}
		body := c.Body()
		if len(body) == 0 || (cfg.API.NIPFE.MaxBodyBytes > 0 && len(body) > cfg.API.NIPFE.MaxBodyBytes) {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "invalid request body"})
		}
		authed, err := validateNIPFEAuth(c, body)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
		default:
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "nip-fe is busy"})
		}
		ctx := c.UserContext()
		if cfg.API.NIPFE.TimeoutSeconds > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(cfg.API.NIPFE.TimeoutSeconds)*time.Second)
			defer cancel()
		}
		frames, status := executeNIPFE(ctx, body, authed)
		payload := bytes.Buffer{}
		for _, frame := range frames {
			encoded, marshalErr := json.Marshal(frame)
			if marshalErr != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "response encoding failed"})
			}
			payload.Write(encoded)
			payload.WriteByte('\n')
		}
		c.Set(fiber.HeaderContentType, "application/x-ndjson")
		return c.Status(status).Send(payload.Bytes())
	}
}

func validateNIPFEAuth(c *fiber.Ctx, body []byte) (string, error) {
	header := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
	if header == "" {
		return "", nil
	}
	result, err := nip86.ValidateNIP98(nip86.AuthInput{
		Authorization: header,
		Method:        fiber.MethodPost,
		URL:           c.BaseURL() + c.OriginalURL(),
		Body:          body,
	}, 60)
	if err != nil {
		return "", err
	}
	return result.PubKey, nil
}

func executeNIPFE(ctx context.Context, body []byte, authed string) ([]any, int) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	var frame []json.RawMessage
	if err := decoder.Decode(&frame); err != nil || len(frame) == 0 {
		return []any{[]any{"CLOSED", "", "invalid: expected one NIP-01 frame"}}, fiber.StatusBadRequest
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return []any{[]any{"CLOSED", "", "invalid: expected one NIP-01 frame"}}, fiber.StatusBadRequest
	}
	var command string
	if err := json.Unmarshal(frame[0], &command); err != nil {
		return []any{[]any{"CLOSED", "", "invalid: command is required"}}, fiber.StatusBadRequest
	}
	switch command {
	case "REQ":
		return executeREQ(ctx, frame, authed)
	case "COUNT":
		return executeCOUNT(ctx, frame, authed)
	case "EVENT":
		return executeEVENT(ctx, frame, authed)
	default:
		return []any{[]any{"CLOSED", "", "invalid: unsupported command"}}, fiber.StatusBadRequest
	}
}

func executeREQ(ctx context.Context, frame []json.RawMessage, authed string) ([]any, int) {
	if len(frame) < 3 {
		return []any{[]any{"CLOSED", "", "invalid: REQ requires id and filter"}}, fiber.StatusBadRequest
	}
	var id string
	if json.Unmarshal(frame[1], &id) != nil || id == "" {
		return []any{[]any{"CLOSED", "", "invalid: REQ id is required"}}, fiber.StatusBadRequest
	}
	frames := make([]any, 0)
	seen := make(map[string]struct{})
	for _, raw := range frame[2:] {
		var filter nostr.Filter
		if json.Unmarshal(raw, &filter) != nil {
			return []any{[]any{"CLOSED", id, "invalid: filter"}}, fiber.StatusBadRequest
		}
		events, err := oneShotEvents(ctx, authed, filter)
		if err != nil {
			return []any{[]any{"CLOSED", id, "error: query failed"}}, fiber.StatusInternalServerError
		}
		for _, evt := range events {
			if _, exists := seen[evt.ID]; exists {
				continue
			}
			seen[evt.ID] = struct{}{}
			frames = append(frames, nostr.EventEnvelope{SubscriptionID: &id, Event: *evt})
		}
	}
	frames = append(frames, nostr.EOSEEnvelope(id))
	return frames, fiber.StatusOK
}

func executeCOUNT(ctx context.Context, frame []json.RawMessage, authed string) ([]any, int) {
	if len(frame) < 3 {
		return []any{[]any{"CLOSED", "", "invalid: COUNT requires id and filter"}}, fiber.StatusBadRequest
	}
	var id string
	if json.Unmarshal(frame[1], &id) != nil || id == "" {
		return []any{[]any{"CLOSED", "", "invalid: COUNT id is required"}}, fiber.StatusBadRequest
	}
	var total int64
	for _, raw := range frame[2:] {
		var filter nostr.Filter
		if json.Unmarshal(raw, &filter) != nil {
			return []any{[]any{"CLOSED", id, "invalid: filter"}}, fiber.StatusBadRequest
		}
		events, err := oneShotEvents(ctx, authed, filter)
		if err != nil {
			return []any{[]any{"CLOSED", id, "error: query failed"}}, fiber.StatusInternalServerError
		}
		total += int64(len(events))
	}
	return []any{[]any{"COUNT", id, map[string]int64{"count": total}}}, fiber.StatusOK
}

func executeEVENT(ctx context.Context, frame []json.RawMessage, authed string) ([]any, int) {
	if len(frame) != 2 {
		return []any{[]any{"OK", "", false, "invalid: EVENT requires one event"}}, fiber.StatusBadRequest
	}
	var evt nostr.Event
	if json.Unmarshal(frame[1], &evt) != nil {
		return []any{[]any{"OK", "", false, "invalid: event"}}, fiber.StatusBadRequest
	}
	frames := event.ProcessOneShot(ctx, authed, &evt)
	if len(frames) == 0 {
		return []any{[]any{"OK", evt.ID, false, "error: no event response"}}, fiber.StatusInternalServerError
	}
	return frames, fiber.StatusOK
}

func oneShotEvents(ctx context.Context, authed string, filter nostr.Filter) ([]*nostr.Event, error) {
	if filter.Limit <= 0 {
		filter.Limit = config.Cfg.Relay.QueryLimit
	}
	if filter.Limit <= 0 {
		return nil, errors.New("invalid filter limit")
	}
	queries := db.DbQueries()
	if queries == nil {
		return nil, errors.New("database is not initialized")
	}
	query := queries.QueryEventsChan
	events, handled, err := groups.QueryEvents(ctx, authed, filter, query)
	if err != nil {
		return nil, err
	}
	if !handled {
		events, err = query(ctx, filter)
		if err != nil {
			return nil, err
		}
	}
	result := make([]*nostr.Event, 0, filter.Limit)
	for evt := range events {
		result = append(result, evt)
	}
	return result, nil
}
