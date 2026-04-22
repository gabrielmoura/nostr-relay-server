package event

import (
	"errors"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/auth"
	"github.com/gabrielmoura/nostr-relay-server/infra/ingestion"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	nostr_custom "github.com/gabrielmoura/nostr-relay-server/infra/nostr-custom"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	policies "github.com/gabrielmoura/nostr-relay-server/internal/policies"
	"github.com/gabrielmoura/nostr-relay-server/internal/security"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

type DecodeError struct {
	Err error
}

func (e *DecodeError) Error() string {
	return "failed to decode event: " + e.Err.Error()
}

func processEvent(ws *dto.WsServer, evt *nostr.Event) string {
	if evt == nil {
		return "failed to decode event: missing payload"
	}

	if config.Cfg.Ws.RequireAuthForEvent() && ws.Authed == "" {
		auth.SendAuthChallenge(ws)
		rejectEvent(ws, evt, "auth-required: this relay requires NIP-42 authentication before EVENT")
		return ""
	}

	if handled := handleSpecialEvent(ws, evt); handled {
		return ""
	}

	ctx := ws.Ctx
	if security.S != nil {
		var reject bool
		var reason string
		ctx, reject, reason = security.S.ValidateEvent(ctx, ws, evt)
		if reject {
			rejectEvent(ws, evt, reason)
			return ""
		}
	}

	if reject, reason := policies.P.ValidateIncomingEvent(ctx, evt); reject {
		rejectEvent(ws, evt, reason)
		return ""
	}

	runEventSideEffects(ws, evt)

	if !enqueueEvent(ws, evt) {
		return ""
	}

	metrics.NostrRequestDuration.WithLabelValues("EVENT").Observe(time.Since(ws.StartTime).Seconds())
	return ""
}

func handleSpecialEvent(ws *dto.WsServer, evt *nostr.Event) bool {
	switch {
	case nostr_custom.IsJobRequest(evt.Kind):
		_ = handleJobRequest(ws, evt)
		return true
	case evt.Kind == nostr.KindNostrConnect:
		_ = handleNostrConnect(ws, evt)
		return true
	case evt.Kind == nostr.KindDeletion:
		_ = handleDeletionEvent(ws, *evt)
		return true
	case evt.Kind == nostr_custom.KindVanish:
		_ = handleDeletionVanishEvent(ws, *evt)
		return true
	default:
		return false
	}
}

func runEventSideEffects(ws *dto.WsServer, evt *nostr.Event) {
	if evt.Kind == nostr.KindProfileMetadata {
		handleProfile(ws, evt)
	}
	if evt.Kind == nostr.KindReporting {
		reportingEvent(ws.Ctx, *evt)
	}
}

func enqueueEvent(ws *dto.WsServer, evt *nostr.Event) bool {
	if ingestion.Push(evt) {
		log.Logger.Debug("event queued for ingestion", zap.String("event_id", evt.ID), zap.Int("kind", evt.Kind))
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: true}
		return true
	}

	ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "error: ingestion queue is full"}
	return false
}

func rejectEvent(ws *dto.WsServer, evt *nostr.Event, reason string) {
	if evt != nil && strings.Contains(reason, "signature") {
		metrics.NostrRelayEventSignatureFailures.Inc()
	}
	if evt == nil {
		ws.ChanSender <- nostr.NoticeEnvelope(reason)
		return
	}
	ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: reason}
}

func wrapErr(prefix string, err error) error {
	if err == nil {
		return nil
	}
	return errors.New(prefix + ": " + err.Error())
}
