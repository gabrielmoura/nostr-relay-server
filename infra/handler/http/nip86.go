package http

import (
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	jsonx "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gabrielmoura/nostr-relay-server/internal/nip86"
	"github.com/gofiber/fiber/v2"
)

func handleNIP86JSONRPC(c *fiber.Ctx, cfg *config.Config) (bool, error) {
	if cfg == nil || !cfg.NIP86Enabled() {
		return false, nil
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType))), nip86ContentType()) {
		return false, nil
	}
	if c.Method() != fiber.MethodPost {
		return true, writeNIP86Response(c, fiber.StatusMethodNotAllowed, nip86.Response{Result: nil, Error: "method not allowed"})
	}

	body := append([]byte(nil), c.Body()...)
	authResult, err := nip86.Authenticate(cfg, nip86.AuthInput{
		Authorization: c.Get(fiber.HeaderAuthorization),
		Method:        c.Method(),
		URL:           absoluteRequestURL(c),
		Body:          body,
	})
	if err != nil {
		return true, writeNIP86Response(c, fiber.StatusUnauthorized, nip86.Response{Result: nil, Error: err.Error()})
	}

	var req nip86.Request
	if err := jsonx.Unmarshal(body, &req); err != nil {
		return true, writeNIP86Response(c, fiber.StatusBadRequest, nip86.Response{Result: nil, Error: "invalid json-rpc body"})
	}

	resp := nip86.S.Execute(c.UserContext(), req, nip86.CallContext{AdminPubKey: authResult.PubKey, RemoteIP: c.IP()})
	status := resp.HTTPStatus
	if status == 0 {
		status = fiber.StatusOK
	}
	return true, writeNIP86Response(c, status, resp)
}

func writeNIP86Response(c *fiber.Ctx, status int, resp nip86.Response) error {
	c.Set(fiber.HeaderContentType, nip86ContentType())
	return c.Status(status).JSON(resp)
}

func nip86ContentType() string {
	return "application/nostr+json+rpc"
}

func absoluteRequestURL(c *fiber.Ctx) string {
	return c.BaseURL() + c.OriginalURL()
}
