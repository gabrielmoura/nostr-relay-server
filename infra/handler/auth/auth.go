package auth

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip42"
	"go.uber.org/zap"
)

const AuthContextKey = "authed"

func DoAUTH(ws *dto.WsServer, data dto.Data) string {
	if config.Cfg.Ws.Auth {
		var evt nostr.Event
		if err := json.Unmarshal(data[1], &evt); err != nil {
			return "failed to decode auth event: " + err.Error()
		}
		if pubkey, ok := nip42.ValidateAuthEvent(&evt, ws.Challenge, config.Cfg.RelayInformation.CanonicalURL); ok {
			ws.Authed = pubkey
			ws.Ctx = context.WithValue(ws.Ctx, AuthContextKey, pubkey)
			ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: true}
		} else {
			log.Logger.Warn("failed to authenticate", zap.String("event", evt.String()))
			ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "error: failed to authenticate"}
		}
	}
	return ""
}
