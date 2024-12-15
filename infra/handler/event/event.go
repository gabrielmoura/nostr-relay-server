package event

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"
	"log/slog"
	"regexp"
	"time"
)

var ErrDupEvent = errors.New("duplicate: event already exists")

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
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "error: failed to verify signature"}
		return ""
	} else if !ok {
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "invalid: signature is invalid"}
		return ""
	}

	if evt.Kind == nostr.KindDeletion {
		// event deletion -- nip09
		for _, tag := range evt.Tags {
			if len(tag) >= 2 && tag[0] == "e" {
				ctx, cancel := context.WithTimeout(ws.Ctx, time.Millisecond*200)
				defer cancel()

				// fetch event to be deleted
				res, err := db.DbQueries.QueryEvents(ctx, nostr.Filter{IDs: []string{tag[1]}})
				if err != nil {
					//ws.Conn.WriteJSON(nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "failed to query for target event"})
					ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "failed to query for target event"}
					return ""
				}

				var target *nostr.Event
				exists := false
				select {
				case target, exists = <-res:
				case <-ctx.Done():
				}
				if !exists {
					// this will happen if event is not in the database
					// or when when the query is taking too long, so we just give up
					continue
				}

				// check if this can be deleted
				if target.PubKey != evt.PubKey {
					//ws.Conn.WriteJSON(nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "insufficient permissions"})
					ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "insufficient permissions"}
					return ""
				}

				//if advancedDeleter != nil {
				//	advancedDeleter.BeforeDelete(ctx, tag[1], evt.PubKey)
				//}

				if err := db.DbQueries.DeleteEvent(ctx, target.ID); err != nil {
					//ws.Conn.WriteJSON(nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: fmt.Sprintf("error: %s", err.Error())})
					ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: fmt.Sprintf("error: %s", err.Error())}
					return ""
				}

				//if advancedDeleter != nil {
				//	advancedDeleter.AfterDelete(tag[1], evt.PubKey)
				//}
			}
		}

		listener.NotifyListeners(&evt)
		//ws.Conn.WriteJSON(nostr.OKEnvelope{EventID: evt.ID, OK: true})
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: true}
		return ""
	}
	if evt.Kind == nostr.KindReporting {
		reportingEvent(ws.Ctx, evt)
	}
	ok, reason := AddEvent(ws, &evt)
	log.Logger.Info("event", evt.ID, "kind", evt.Kind, "ok", ok, "reason", reason)

	//ws.Conn.WriteJSON(nostr.OKEnvelope{EventID: evt.ID, OK: ok, Reason: reason})
	ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: ok, Reason: reason}
	return ""
}

var nip20prefixmatcher = regexp.MustCompile(`^\w+: `)

// AddEvent has a business rule to add an event to the relayer
func AddEvent(ws *dto.WsServer, evt *nostr.Event) (accepted bool, message string) {
	if evt == nil {
		return false, ""
	}
	//
	//store := relay.Storage(ctx)
	//wrapper := &eventstore.RelayWrapper{
	//	Store: store,
	//}
	//advancedSaver, _ := store.(AdvancedSaver)

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
			switch saveErr {
			case ErrDupEvent:
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

	return true, ""
}

func publish(ctx context.Context, evt nostr.Event) error {
	if nostr.IsEphemeralKind(evt.Kind) {
		// do not store ephemeral events
		processEphemeralEvents(ctx, &evt)
		return nil
	} else if nostr.IsReplaceableKind(evt.Kind) {
		// replaceable event, delete before storing
		ch, err := db.DbQueries.QueryEvents(ctx, nostr.Filter{Authors: []string{evt.PubKey}, Kinds: []int{evt.Kind}})
		if err != nil {
			return fmt.Errorf("failed to query before replacing: %w", err)
		}
		if previous := <-ch; previous != nil && isOlder(previous, &evt) {
			if err := db.DbQueries.DeleteEvent(ctx, previous.ID); err != nil {
				return fmt.Errorf("failed to delete event for replacing: %w", err)
			}
		}
	} else if nostr.IsAddressableKind(evt.Kind) {
		// parameterized replaceable event, delete before storing
		d := evt.Tags.GetFirst([]string{"d", ""})
		if d != nil {
			ch, err := db.DbQueries.QueryEvents(ctx, nostr.Filter{Authors: []string{evt.PubKey}, Kinds: []int{evt.Kind}, Tags: nostr.TagMap{"d": []string{d.Value()}}})
			if err != nil {
				return fmt.Errorf("failed to query before parameterized replacing: %w", err)
			}
			if previous := <-ch; previous != nil && isOlder(previous, &evt) {
				if err := db.DbQueries.DeleteEvent(ctx, previous.ID); err != nil {
					return fmt.Errorf("failed to delete event for parameterized replacing: %w", err)
				}
			}
		}
	}

	if err := db.DbQueries.InsertEvent(ctx, &evt); err != nil && !errors.Is(err, ErrDupEvent) {
		log.Logger.Error("failed to save", err)
		return fmt.Errorf("failed to save: %w", err)
	}

	return nil
}

func isOlder(previous, next *nostr.Event) bool {
	return previous.CreatedAt < next.CreatedAt ||
		(previous.CreatedAt == next.CreatedAt && previous.ID > next.ID)
}

func processEphemeralEvents(ctx context.Context, evt *nostr.Event) {
	log.Logger.InfoContext(ctx, "ephemeral events", slog.Any("event", evt))
}
