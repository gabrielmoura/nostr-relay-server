package event

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/fiatjaf/khatru/policies"
	"github.com/gabrielmoura/nostr-relay-server/config"
	db2 "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	nostr_custom "github.com/gabrielmoura/nostr-relay-server/infra/nostr-custom"
	"github.com/gabrielmoura/nostr-relay-server/infra/stream"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	policies2 "github.com/gabrielmoura/nostr-relay-server/internal/policies"
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"regexp"
	"time"
)

func DoEVENT(ws *dto.WsServer, data dto.Data) string {
	latestIndex := len(data) - 1

	// it's a new event
	var evt nostr.Event
	if err := json.Unmarshal(data[latestIndex], &evt); err != nil {
		return "failed to decode event: " + err.Error()
	}

	// check id
	hash := sha256.Sum256(evt.Serialize())
	if id := hex.EncodeToString(hash[:]); id != evt.ID {
		reason := "invalid: event id is computed incorrectly"
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: reason}
		return ""
	}

	// check signature
	if ok, err := evt.CheckSignature(); err != nil {
		metrics.NostrRelayEventSignatureFailures.Inc()
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "error: failed to verify signature"}
		return ""
	} else if !ok {
		metrics.NostrRelayEventSignatureFailures.Inc()
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "invalid: signature is invalid"}
		return ""
	}

	if reject, msg := policies2.P.RejectExpiredEvent(evt); reject {
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: msg}
	}

	if reject, msg := policies2.P.CheckMinimumPow(evt); reject {
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: msg}
	}

	if ok, err := policies2.P.RejectEventBannedUser(ws.Ctx, &evt); ok {
		log.Logger.Debug("Rejecting event", zap.String("event", evt.ID), zap.String("reason", err))
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: ok, Reason: err}
		return ""
	}

	if ok, err := policies.PreventLargeTags(config.Cfg.Relay.MaxTagValueLength)(ws.Ctx, &evt); ok {
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: ok, Reason: err}
		return ""
	}

	if ok, err := policies.PreventTooManyIndexableTags(config.Cfg.Relay.MaxTagValueLength, []int{}, []int{})(ws.Ctx, &evt); ok {
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: ok, Reason: err}
		return ""
	}

	if ok, err := policies.RejectEventsWithBase64Media(ws.Ctx, &evt); ok {
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: ok, Reason: err}
		return ""
	}

	if evt.Kind == nostr.KindProfileMetadata {
		handleProfile(ws, &evt)
	}

	if nostr_custom.IsJobRequest(evt.Kind) {
		return handleJobRequest(ws, &evt)
	}

	if evt.Kind == nostr.KindNostrConnect {
		return handleNostrConnect(ws, &evt)
	}

	if evt.Kind == nostr.KindDeletion {
		return handleDeletionEvent(ws, evt)
	}

	if evt.Kind == nostr.KindReporting {
		reportingEvent(ws.Ctx, evt)
	}
	ok, reason := AddEvent(ws, &evt)

	log.Logger.Debug(
		"Acceptation event",
		zap.Bool("Accept", ok),
		zap.String("reason", reason),
		zap.String("event", evt.ID),
	)

	ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: ok, Reason: reason}
	return ""
}

var nip20prefixmatcher = regexp.MustCompile(`^\w+: `)

// AddEvent has a business rule to add an event to the relayer
func AddEvent(ws *dto.WsServer, evt *nostr.Event) (accepted bool, message string) {
	if evt == nil {
		return false, ""
	}

	// regra para aceitar ou não o evento
	if !ws.AcceptEvent(evt) {
		return false, "blocked: event blocked by relay"
	}

	if nostr.IsEphemeralKind(evt.Kind) {

		// do not store ephemeral events
		processEphemeralEvents(ws.Ctx, evt)

	} else {
		//if advancedSaver != nil {
		//	advancedSaver.BeforeSave(ctx, evt)
		//}

		if saveErr := publish(ws.Ctx, *evt); saveErr != nil {
			switch {
			case errors.Is(saveErr, db2.ErrDupEvent):
				metrics.NostrRelayEventDuplicateRejections.Inc()
				return true, saveErr.Error()
			default:
				errmsg := saveErr.Error()
				if nip20prefixmatcher.MatchString(errmsg) {
					return false, errmsg
				} else {
					return false, fmt.Sprintf("error: failed to save (%s)", errmsg)
				}
			}
		}

		//if advancedSaver != nil {
		//	advancedSaver.AfterSave(evt)
		//}
	}

	listener.NotifyListeners(evt)
	metrics.NostrRequestDuration.WithLabelValues("EVENT").Observe(time.Since(ws.StartTime).Seconds())

	return true, ""
}

func publish(ctx context.Context, evt nostr.Event) error {
	if nostr.IsEphemeralKind(evt.Kind) {
		// do not store ephemeral events
		processEphemeralEvents(ctx, &evt)
		return nil
	} else if nostr.IsReplaceableKind(evt.Kind) {
		// replaceable event, delete before storing
		ch, err := db.DbQueries.QueryEventsChan(ctx, nostr.Filter{Authors: []string{evt.PubKey}, Kinds: []int{evt.Kind}})
		if err != nil {
			return fmt.Errorf("failed to query before replacing: %w", err)
		}
		if previous := <-ch; previous != nil && isOlder(previous, &evt) {
			if err := db.DbQueries.DeleteEvent(ctx, previous.ID, evt.ID); err != nil {
				return fmt.Errorf("failed to delete event for replacing: %w", err)
			}
		}
	} else if nostr.IsAddressableKind(evt.Kind) {
		// parameterized replaceable event, delete before storing
		d := evt.Tags.GetFirst([]string{"d", ""})
		if d != nil {
			ch, err := db.DbQueries.QueryEventsChan(ctx, nostr.Filter{Authors: []string{evt.PubKey}, Kinds: []int{evt.Kind}, Tags: nostr.TagMap{"d": []string{d.Value()}}})
			if err != nil {
				return fmt.Errorf("failed to query before parameterized replacing: %w", err)
			}
			if previous := <-ch; previous != nil && isOlder(previous, &evt) {
				if err := db.DbQueries.DeleteEvent(ctx, previous.ID, evt.ID); err != nil {
					return fmt.Errorf("failed to delete event for parameterized replacing: %w", err)
				}
			}
		}
	}

	stream.ForwardEvent(evt)

	if err := db.DbQueries.InsertEvent(ctx, &evt); err != nil {
		if !errors.Is(err, db2.ErrDupEvent) {
			log.Logger.Error("failed to save", zap.Error(err))
			return fmt.Errorf("failed to save: %w", err)
		}
		metrics.NostrRelayEventDuplicateRejections.Inc()
	}
	metrics.NostrKindEventCounter.WithLabelValues(metrics.GetKindName(evt.Kind)).Inc()
	metrics.NostrUserEventCounter.WithLabelValues(evt.PubKey).Inc()

	return nil
}

func isOlder(previous, next *nostr.Event) bool {
	return previous.CreatedAt < next.CreatedAt ||
		(previous.CreatedAt == next.CreatedAt && previous.ID > next.ID)
}

func processEphemeralEvents(ctx context.Context, evt *nostr.Event) {
	log.Logger.Info("ephemeral events", zap.Any("event", evt))
}
