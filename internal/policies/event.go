package policies

import (
	"context"
	json "github.com/bytedance/sonic"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip13"
	"go.uber.org/zap"
	"slices"
	"strconv"
	"strings"
	"time"
)

func (p Policies) AcceptEvent(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
	GetUserBannedByKeyCached := cache.WrapGetBanned(db.DbQueries.GetUserBannedByKey)

	reason, exists, err := GetUserBannedByKeyCached(ctx, event.PubKey)
	if err != nil {
		log.Logger.Error("Erro ao verificar se o usuário está banido", zap.Error(err))
		return true, "error: " + err.Error()
	}
	if exists {
		log.Logger.Info("Usuário banido", zap.String("reason", reason))
		return true, "banned user: " + reason
	}

	jsonb, _ := json.Marshal(event)
	if len(jsonb) > config.Cfg.Relay.MaxEventSize {
		log.Logger.Debug(
			"very big event",
			zap.Int("size", len(jsonb)),
			zap.Int("max", config.Cfg.Relay.MaxEventSize),
		)
		return true, "very big event"
	}

	return false, "event accepted"
}

// RejectExpiredEvent rejects events that are older than the maximum allowed age
// NIP-40 https://github.com/nostr-protocol/nips/blob/master/40.md
func (p Policies) RejectExpiredEvent(event nostr.Event) (reject bool, msg string) {
	for e := range event.Tags {
		if event.Tags[e][0] == "expiration" {
			if len(event.Tags[e]) < 2 {
				return true, "invalid expiration tag"
			}
			if expiration, err := strconv.ParseInt(event.Tags[e][1], 10, 64); err != nil {
				now := time.Now().Unix()
				if expiration < now {
					log.Logger.Debug(
						"expired event",
						zap.Int64("expiration", expiration),
						zap.Int64("now", now),
					)
					return true, "expired event"
				}
			} else {
				log.Logger.Error("Erro ao analisar a tag de expiração do evento", zap.Error(err))
				return true, "invalid expiration tag"
			}
			break
		}
	}
	return false, "event not expired"

}

func (p Policies) CheckMinimumPow(evt nostr.Event) (reject bool, msg string) {
	if p.Config.Relay.MinimumPOWLimit == 0 {
		return false, ""
	}
	err := nip13.Check(evt.ID, config.Cfg.Relay.MinimumPOWLimit)
	if err != nil {
		return true, "blocked: minimum POW not obtained"
	} else {
		return false, ""
	}
}

// PreventLargeTags rejects events that have indexable tag values greater than maxTagValueLen.
func PreventLargeTags(maxTagValueLen int) func(context.Context, *nostr.Event) (bool, string) {
	return func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
		for _, tag := range event.Tags {
			if len(tag) > 1 && len(tag[0]) == 1 {
				if len(tag[1]) > maxTagValueLen {
					return true, "event contains too large tags"
				}
			}
		}
		return false, ""
	}
}

// PreventTooManyIndexableTags returns a function that can be used as a RejectFilter that will reject
// events with more indexable (single-character) tags than the specified number.
//
// If ignoreKinds is given this restriction will not apply to these kinds (useful for allowing a bigger).
// If onlyKinds is given then all other kinds will be ignored.
func PreventTooManyIndexableTags(max int, ignoreKinds []int, onlyKinds []int) func(context.Context, *nostr.Event) (bool, string) {
	slices.Sort(ignoreKinds)
	slices.Sort(onlyKinds)

	ignore := func(kind int) bool { return false }
	if len(ignoreKinds) > 0 {
		ignore = func(kind int) bool {
			_, isIgnored := slices.BinarySearch(ignoreKinds, kind)
			return isIgnored
		}
	}
	if len(onlyKinds) > 0 {
		ignore = func(kind int) bool {
			_, isApplicable := slices.BinarySearch(onlyKinds, kind)
			return !isApplicable
		}
	}

	return func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
		if ignore(event.Kind) {
			return false, ""
		}

		ntags := 0
		for _, tag := range event.Tags {
			if len(tag) > 0 && len(tag[0]) == 1 {
				ntags++
			}
		}
		if ntags > max {
			return true, "too many indexable tags"
		}
		return false, ""
	}
}

func RejectEventsWithBase64Media(ctx context.Context, evt *nostr.Event) (bool, string) {
	return strings.Contains(evt.Content, "data:image/") || strings.Contains(evt.Content, "data:video/"), "event with base64 media"
}
