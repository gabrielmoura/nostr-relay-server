package http

import (
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/db"
	internaldb "github.com/gabrielmoura/nostr-relay-server/internal/db"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	intnip86 "github.com/gabrielmoura/nostr-relay-server/internal/nip86"
	"github.com/gofiber/fiber/v2"
)

type adminNIP86ReasonRequest struct {
	Reason string `json:"reason"`
}

type adminNIP86RelayMetadataRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func NIP86AllowedPubKeys() fiber.Handler {
	return func(c *fiber.Ctx) error {
		items, ok := executeNIP86List[db.NIP86PubKeyRecord](c, intnip86.MethodListAllowedPubKeys)
		if !ok {
			return nil
		}
		filtered := filterAllowedPubkeys(items, strings.TrimSpace(c.Query("q")))
		limit := adminLimit(c)
		offset := adminOffset(c)
		return c.JSON(newAdminPage(paginate(filtered, limit, offset), len(filtered), limit, offset))
	}
}

func NIP86CreateAllowedPubKey() fiber.Handler {
	return func(c *fiber.Ctx) error {
		pubkey, err := normalizePublicKey(c.Params("pubkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		var req adminNIP86ReasonRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		return executeNIP86Mutation(c, intnip86.MethodAllowPubKey, pubkey, req.Reason)
	}
}

func NIP86DeleteAllowedPubKey() fiber.Handler {
	return func(c *fiber.Ctx) error {
		pubkey, err := normalizePublicKey(c.Params("pubkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return executeNIP86Mutation(c, intnip86.MethodUnallowPubKey, pubkey)
	}
}

func NIP86BlockedIPs() fiber.Handler {
	return func(c *fiber.Ctx) error {
		items, ok := executeNIP86List[db.NIP86IPRecord](c, intnip86.MethodListBlockedIPs)
		if !ok {
			return nil
		}
		filtered := filterBlockedIPs(items, strings.TrimSpace(c.Query("q")))
		limit := adminLimit(c)
		offset := adminOffset(c)
		return c.JSON(newAdminPage(paginate(filtered, limit, offset), len(filtered), limit, offset))
	}
}

func NIP86CreateBlockedIP() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := strings.TrimSpace(c.Params("ip"))
		var req adminNIP86ReasonRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		return executeNIP86Mutation(c, intnip86.MethodBlockIP, ip, req.Reason)
	}
}

func NIP86DeleteBlockedIP() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return executeNIP86Mutation(c, intnip86.MethodUnblockIP, strings.TrimSpace(c.Params("ip")))
	}
}

func NIP86BannedEvents() fiber.Handler {
	return func(c *fiber.Ctx) error {
		items, ok := executeNIP86List[db.NIP86EventRecord](c, intnip86.MethodListBannedEvents)
		if !ok {
			return nil
		}
		filtered := filterBannedEvents(items, strings.TrimSpace(c.Query("q")))
		limit := adminLimit(c)
		offset := adminOffset(c)
		return c.JSON(newAdminPage(paginate(filtered, limit, offset), len(filtered), limit, offset))
	}
}

func NIP86CreateBannedEvent() fiber.Handler {
	return func(c *fiber.Ctx) error {
		eventID := strings.TrimSpace(c.Params("id"))
		var req adminNIP86ReasonRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		return executeNIP86Mutation(c, intnip86.MethodBanEvent, eventID, req.Reason)
	}
}

func NIP86DeleteBannedEvent() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return executeNIP86Mutation(c, intnip86.MethodAllowEvent, strings.TrimSpace(c.Params("id")))
	}
}

func NIP86RelayMetadata() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if intnip86.S == nil || !intnip86.S.Enabled() {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "nip86 is disabled"})
		}
		record, exists, err := internaldb.DbQueries.GetRelayMetadata(c.UserContext(), strings.TrimSpace(config.Cfg.RelayInformation.URL))
		if err != nil {
			return internalServerError(c, err)
		}
		if !exists {
			return c.JSON(fiber.Map{
				"relay_url":   config.Cfg.RelayInformation.URL,
				"name":        config.Cfg.RelayInformation.Name,
				"description": config.Cfg.RelayInformation.Description,
			})
		}
		return c.JSON(record)
	}
}

func NIP86UpdateRelayMetadata() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req adminNIP86RelayMetadataRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		if ok, err := executeNIP86MutationAndDecodeBool(c, intnip86.MethodChangeRelayName, req.Name); err != nil || !ok {
			return err
		}
		if ok, err := executeNIP86MutationAndDecodeBool(c, intnip86.MethodChangeRelayDesc, req.Description); err != nil || !ok {
			return err
		}
		return c.JSON(fiber.Map{"updated": true, "name": req.Name, "description": req.Description})
	}
}

func executeNIP86List[T any](c *fiber.Ctx, method string) ([]T, bool) {
	resp, ok := executeNIP86(c, method)
	if !ok {
		return nil, false
	}
	payload, err := json.Marshal(resp.Result)
	if err != nil {
		_ = internalServerError(c, err)
		return nil, false
	}
	items := make([]T, 0)
	if err := json.Unmarshal(payload, &items); err != nil {
		_ = internalServerError(c, err)
		return nil, false
	}
	return items, true
}

func executeNIP86Mutation(c *fiber.Ctx, method string, values ...string) error {
	result, err := executeNIP86MutationAndDecodeBool(c, method, values...)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": result})
}

func executeNIP86MutationAndDecodeBool(c *fiber.Ctx, method string, values ...string) (bool, error) {
	resp, ok := executeNIP86(c, method, values...)
	if !ok {
		return false, nil
	}
	result, valid := resp.Result.(bool)
	if valid {
		return result, nil
	}
	return true, nil
}

func executeNIP86(c *fiber.Ctx, method string, values ...string) (intnip86.Response, bool) {
	if intnip86.S == nil || !intnip86.S.Enabled() {
		_ = c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "nip86 is disabled"})
		return intnip86.Response{}, false
	}
	params := make([]json.RawMessage, 0, len(values))
	for _, value := range values {
		payload, err := json.Marshal(value)
		if err != nil {
			_ = internalServerError(c, err)
			return intnip86.Response{}, false
		}
		params = append(params, payload)
	}
	resp := intnip86.S.Execute(c.UserContext(), intnip86.Request{Method: method, Params: params}, intnip86.CallContext{AdminPubKey: config.Cfg.AdminPubKey, RemoteIP: c.IP()})
	if resp.Error != "" {
		_ = c.Status(resp.HTTPStatus).JSON(fiber.Map{"error": resp.Error})
		return intnip86.Response{}, false
	}
	return resp, true
}

func filterAllowedPubkeys(items []db.NIP86PubKeyRecord, query string) []db.NIP86PubKeyRecord {
	return filterRecords(items, query, func(item db.NIP86PubKeyRecord) string { return item.PubKey + " " + item.Reason })
}

func filterBlockedIPs(items []db.NIP86IPRecord, query string) []db.NIP86IPRecord {
	return filterRecords(items, query, func(item db.NIP86IPRecord) string { return item.IP + " " + item.Reason })
}

func filterBannedEvents(items []db.NIP86EventRecord, query string) []db.NIP86EventRecord {
	return filterRecords(items, query, func(item db.NIP86EventRecord) string { return item.EventID + " " + item.Reason })
}

func filterRecords[T any](items []T, query string, flatten func(T) string) []T {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return items
	}
	filtered := make([]T, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(flatten(item)), query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
