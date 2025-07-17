package store

import (
	"encoding/base64"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	errors2 "github.com/gabrielmoura/nostr-relay-server/internal/errors"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"strings"
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
