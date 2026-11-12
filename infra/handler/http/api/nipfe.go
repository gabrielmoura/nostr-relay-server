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
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/event"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dmimport"
	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
	"github.com/gabrielmoura/nostr-relay-server/internal/nip86"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

var findNIPFEEvents = oneShotEvents
var findNIPFECountPage = oneShotCountPage

const defaultNIPFEMaxResponseBytes = 8 << 20
const defaultNIPFEMaxFilters = 16
const defaultNIPFEMaxResponseEvents = 2000
const defaultNIPFEMaxScan = 2000

type nipFEWorkBudget struct {
	remainingScans int
}

type nipFEWorkBudgetKey struct{}

type nipFEConfigKey struct{}

func newNIPFEWorkBudget(cfg *config.Config) *nipFEWorkBudget {
	maxScans := defaultNIPFEMaxScan
	if cfg != nil && cfg.API.MaxScan > 0 {
		maxScans = cfg.API.MaxScan
	}
	return &nipFEWorkBudget{remainingScans: maxScans}
}

func (budget *nipFEWorkBudget) pageLimit(requested int) (int, error) {
	if budget == nil {
		return requested, nil
	}
	if budget.remainingScans <= 0 {
		return 0, errScanLimit
	}
	return min(requested, budget.remainingScans), nil
}

func (budget *nipFEWorkBudget) consume(scanned, limit int) error {
	if scanned < 0 || scanned > limit {
		return errors.New("invalid NIP-FE page size")
	}
	if budget == nil {
		return nil
	}
	budget.remainingScans -= scanned
	return nil
}

func nipFEWorkBudgetFromContext(ctx context.Context) *nipFEWorkBudget {
	budget, _ := ctx.Value(nipFEWorkBudgetKey{}).(*nipFEWorkBudget)
	return budget
}

func nipFEConfigFromContext(ctx context.Context) *config.Config {
	cfg, _ := ctx.Value(nipFEConfigKey{}).(*config.Config)
	if cfg != nil {
		return cfg
	}
	return config.Cfg
}

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
		frames, status := executeNIPFEWithConfig(ctx, body, authed, cfg)
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
	return executeNIPFEWithConfig(ctx, body, authed, config.Cfg)
}

func executeNIPFEWithConfig(ctx context.Context, body []byte, authed string, cfg *config.Config) ([]any, int) {
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
	maxFilters, maxBytes, maxEvents := defaultNIPFEMaxFilters, defaultNIPFEMaxResponseBytes, defaultNIPFEMaxResponseEvents
	if cfg != nil {
		if cfg.Security.Limits.MaxFiltersPerReq > 0 {
			maxFilters = cfg.Security.Limits.MaxFiltersPerReq
		}
		if cfg.API.NIPFE.MaxResponseBytes > 0 {
			maxBytes = cfg.API.NIPFE.MaxResponseBytes
		}
		if cfg.API.NIPFE.MaxResponseEvents > 0 {
			maxEvents = cfg.API.NIPFE.MaxResponseEvents
		}
	}
	if (command == "REQ" || command == "COUNT") && len(frame)-2 > maxFilters {
		return []any{[]any{"CLOSED", "", "invalid: too many filters"}}, fiber.StatusBadRequest
	}
	switch command {
	case "REQ":
		ctx = context.WithValue(ctx, nipFEWorkBudgetKey{}, newNIPFEWorkBudget(cfg))
		ctx = context.WithValue(ctx, nipFEConfigKey{}, cfg)
		return executeREQ(ctx, frame, authed, maxBytes, maxEvents)
	case "COUNT":
		ctx = context.WithValue(ctx, nipFEWorkBudgetKey{}, newNIPFEWorkBudget(cfg))
		ctx = context.WithValue(ctx, nipFEConfigKey{}, cfg)
		return executeCOUNT(ctx, frame, authed, cfg)
	case "EVENT":
		return executeEVENT(ctx, frame, authed)
	default:
		return []any{[]any{"CLOSED", "", "invalid: unsupported command"}}, fiber.StatusBadRequest
	}
}

func executeREQ(ctx context.Context, frame []json.RawMessage, authed string, maxBytes, maxEvents int) ([]any, int) {
	if len(frame) < 3 {
		return []any{[]any{"CLOSED", "", "invalid: REQ requires id and filter"}}, fiber.StatusBadRequest
	}
	var id string
	if json.Unmarshal(frame[1], &id) != nil || id == "" {
		return []any{[]any{"CLOSED", "", "invalid: REQ id is required"}}, fiber.StatusBadRequest
	}
	frames := make([]any, 0)
	seen := make(map[string]struct{})
	responseBytes := 0
	for _, raw := range frame[2:] {
		var filter nostr.Filter
		if json.Unmarshal(raw, &filter) != nil {
			return []any{[]any{"CLOSED", id, "invalid: filter"}}, fiber.StatusBadRequest
		}
		events, err := findNIPFEEvents(ctx, authed, filter)
		if err != nil {
			if errors.Is(err, errScanLimit) {
				return []any{[]any{"CLOSED", id, "error: request work limit exceeded"}}, fiber.StatusServiceUnavailable
			}
			return []any{[]any{"CLOSED", id, "error: query failed"}}, fiber.StatusInternalServerError
		}
		events = filterNIPFEVisibleEvents(events, authed)
		for _, evt := range events {
			if _, exists := seen[evt.ID]; exists {
				continue
			}
			if len(seen) >= maxEvents {
				return []any{[]any{"CLOSED", id, "error: response exceeds event limit"}}, fiber.StatusRequestEntityTooLarge
			}
			envelope := nostr.EventEnvelope{SubscriptionID: &id, Event: *evt}
			encoded, err := json.Marshal(envelope)
			if err != nil {
				return []any{[]any{"CLOSED", id, "error: response encoding failed"}}, fiber.StatusInternalServerError
			}
			responseBytes += len(encoded) + 1
			if responseBytes > maxBytes {
				return []any{[]any{"CLOSED", id, "error: response exceeds limit"}}, fiber.StatusRequestEntityTooLarge
			}
			seen[evt.ID] = struct{}{}
			frames = append(frames, envelope)
		}
	}
	eose := nostr.EOSEEnvelope(id)
	encoded, err := json.Marshal(eose)
	if err != nil {
		return []any{[]any{"CLOSED", id, "error: response encoding failed"}}, fiber.StatusInternalServerError
	}
	if responseBytes+len(encoded)+1 > maxBytes {
		return []any{[]any{"CLOSED", id, "error: response exceeds limit"}}, fiber.StatusRequestEntityTooLarge
	}
	frames = append(frames, eose)
	return frames, fiber.StatusOK
}

func executeCOUNT(ctx context.Context, frame []json.RawMessage, authed string, cfg *config.Config) ([]any, int) {
	if len(frame) < 3 {
		return []any{[]any{"CLOSED", "", "invalid: COUNT requires id and filter"}}, fiber.StatusBadRequest
	}
	var id string
	if json.Unmarshal(frame[1], &id) != nil || id == "" {
		return []any{[]any{"CLOSED", "", "invalid: COUNT id is required"}}, fiber.StatusBadRequest
	}
	seen := make(map[string]struct{})
	for _, raw := range frame[2:] {
		var filter nostr.Filter
		if json.Unmarshal(raw, &filter) != nil {
			return []any{[]any{"CLOSED", id, "invalid: filter"}}, fiber.StatusBadRequest
		}
		// COUNT concerns the complete matching set, regardless of REQ's limit.
		// Bound work and fail explicitly if the scan cannot be completed.
		batchSize := 100
		if cfg != nil && cfg.Relay.QueryLimit > 0 {
			batchSize = min(batchSize, cfg.Relay.QueryLimit)
		}
		for offset := 0; ; {
			limit, budgetErr := nipFEWorkBudgetFromContext(ctx).pageLimit(batchSize)
			if budgetErr != nil {
				return []any{[]any{"CLOSED", id, "error: request work limit exceeded"}}, fiber.StatusServiceUnavailable
			}
			filter.Limit = limit
			events, rawCount, err := findNIPFECountPage(ctx, authed, filter, offset)
			if err != nil {
				return []any{[]any{"CLOSED", id, "error: query failed"}}, fiber.StatusInternalServerError
			}
			if err := nipFEWorkBudgetFromContext(ctx).consume(rawCount, limit); err != nil {
				return []any{[]any{"CLOSED", id, "error: query returned an invalid page"}}, fiber.StatusInternalServerError
			}
			for _, event := range filterNIPFEVisibleEvents(events, authed) {
				seen[event.ID] = struct{}{}
			}
			if rawCount < limit {
				break
			}
			offset += rawCount
		}
	}
	return []any{[]any{"COUNT", id, map[string]int64{"count": int64(len(seen))}}}, fiber.StatusOK
}

func oneShotCountPage(ctx context.Context, authed string, filter nostr.Filter, offset int) ([]*nostr.Event, int, error) {
	queries := db.DbQueries
	if queries == nil {
		return nil, 0, errors.New("database is not initialized")
	}
	candidates, err := queries.QueryEventsWindowOnly(ctx, filter, offset)
	if err != nil {
		return nil, 0, err
	}
	visible, err := groups.FilterEvents(ctx, authed, filter, candidates)
	if err != nil {
		return nil, 0, err
	}
	return visible, len(candidates), nil
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
	cfg := nipFEConfigFromContext(ctx)
	if cfg == nil || cfg.Relay.QueryLimit <= 0 {
		return nil, errors.New("invalid filter limit")
	}
	limit := filter.Limit
	if limit <= 0 || limit > cfg.Relay.QueryLimit {
		limit = cfg.Relay.QueryLimit
	}
	batchSize := min(100, cfg.Relay.QueryLimit)
	result := make([]*nostr.Event, 0, min(limit, batchSize))
	budget := nipFEWorkBudgetFromContext(ctx)
	if budget == nil {
		budget = newNIPFEWorkBudget(cfg)
	}
	for offset := 0; ; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch, err := budget.pageLimit(batchSize)
		if err != nil {
			return nil, err
		}
		filter.Limit = batch
		events, rawCount, err := findNIPFECountPage(ctx, authed, filter, offset)
		if err != nil {
			return nil, err
		}
		if err := budget.consume(rawCount, batch); err != nil {
			return nil, err
		}
		for _, event := range filterNIPFEVisibleEvents(events, authed) {
			result = append(result, event)
			if len(result) == limit {
				return result, nil
			}
		}
		if rawCount < batch {
			return result, nil
		}
		offset += rawCount
	}
}

// filterNIPFEVisibleEvents applies the same private-delivery gate as WebSocket
// REQ. NIP-FE is a finite transport, not an authorization bypass for stored DM
// envelopes.
func filterNIPFEVisibleEvents(events []*nostr.Event, authed string) []*nostr.Event {
	visible := make([]*nostr.Event, 0, len(events))
	for _, evt := range events {
		// NIP-FE does not implement the narrowly validated Concord stream
		// filter (kind 1059 with an exact author). Keep its finite HTTP reads
		// on the recipient/author delivery rule instead of widening access.
		if evt != nil && !hasProtectedTag(evt) && dmimport.VisibleToWithConcord(authed, evt, false) {
			visible = append(visible, evt)
		}
	}
	return visible
}
