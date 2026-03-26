package policies

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/minio/sha256-simd"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip13"
	"github.com/tmthrgd/go-hex"
)

type Policies struct {
	Config *config.Config
}

var P *Policies

func Init() {
	P = &Policies{Config: config.Cfg}
}

func (p Policies) ValidateIncomingEvent(ctx context.Context, evt *nostr.Event) (bool, string) {
	if reject, reason := p.validateEventIdentity(evt); reject {
		return true, reason
	}
	return p.validateStorageEvent(ctx, evt)
}

func (p Policies) ValidateBatchEvent(ctx context.Context, evt *nostr.Event) (bool, string) {
	if reject, reason := p.validateEventIdentity(evt); reject {
		return true, reason
	}
	return p.validateStorageEvent(ctx, evt)
}

func (p Policies) ValidateReq(ctx context.Context, ws *dto.WsServer, filters nostr.Filters) (nostr.Filters, bool, string) {
	return p.validateRequestFilters(ctx, ws, filters)
}

func (p Policies) ValidateCount(ctx context.Context, ws *dto.WsServer, filters nostr.Filters) (nostr.Filters, bool, string) {
	return p.validateRequestFilters(ctx, ws, filters)
}

func (p Policies) validateRequestFilters(ctx context.Context, ws *dto.WsServer, filters nostr.Filters) (nostr.Filters, bool, string) {
	normalized := make(nostr.Filters, 0, len(filters))
	for _, filter := range filters {
		normalized = append(normalized, p.normalizeFilter(filter))
	}

	if reject, reason := p.rejectReqBannedUser(ctx, ws); reject {
		return nil, true, reason
	}

	if reject, reason := p.rejectReqWithoutAuth(normalized, ws); reject {
		return nil, true, reason
	}

	for _, filter := range normalized {
		if reject, reason := p.noEmptyFilters(filter); reject {
			return nil, true, reason
		}
		if reject, reason := p.antiSyncBots(filter); reject {
			return nil, true, reason
		}
		if reject, reason := p.checkKindsAuth(filter, ws); reject {
			return nil, true, reason
		}
		if reject, reason := p.checkDirectMessageAccess(filter, ws); reject {
			return nil, true, reason
		}
	}

	return normalized, false, ""
}

func (p Policies) normalizeFilter(filter nostr.Filter) nostr.Filter {
	if filter.Limit == 0 {
		filter.Limit = p.Config.Relay.FilterLimit
	}
	if filter.Limit > p.Config.Relay.FilterLimit {
		filter.Limit = p.Config.Relay.FilterLimit
	}
	return filter
}

func (p Policies) validateEventIdentity(evt *nostr.Event) (bool, string) {
	if evt == nil {
		return true, "invalid: missing event"
	}
	hash := sha256.Sum256(evt.Serialize())
	if id := hex.EncodeToString(hash[:]); id != evt.ID {
		return true, "invalid: event id is computed incorrectly"
	}
	if ok, err := evt.CheckSignature(); err != nil {
		return true, "error: failed to verify signature"
	} else if !ok {
		return true, "invalid: signature is invalid"
	}
	return false, ""
}

func (p Policies) validateStorageEvent(ctx context.Context, evt *nostr.Event) (bool, string) {
	if reject, reason := p.rejectEventBannedUser(ctx, evt); reject {
		return true, reason
	}

	encoded, _ := json.Marshal(evt)
	if len(encoded) > p.Config.Relay.MaxEventSize {
		return true, "very big event"
	}
	if reject, reason := p.rejectExpiredEvent(evt); reject {
		return true, reason
	}
	if reject, reason := p.checkMinimumPow(evt); reject {
		return true, reason
	}
	if reject, reason := p.preventLargeTags(evt); reject {
		return true, reason
	}
	if reject, reason := p.preventTooManyIndexableTags(evt); reject {
		return true, reason
	}
	if reject, reason := p.rejectEventsWithBase64Media(evt); reject {
		return true, reason
	}
	return false, ""
}

func (p Policies) rejectEventBannedUser(ctx context.Context, evt *nostr.Event) (bool, string) {
	if evt.PubKey == "" {
		return true, "invalid: missing public key"
	}
	reason, exists, err := cache.WrapGetBanned(db.DbQueries.GetUserBannedByKey)(ctx, evt.PubKey)
	if err != nil {
		return true, fmt.Sprintf("error: %s", err.Error())
	}
	if exists {
		return true, fmt.Sprintf("banned: %s", reason)
	}
	return false, ""
}

func (p Policies) rejectReqBannedUser(ctx context.Context, ws *dto.WsServer) (bool, string) {
	if p.Config.Relay.EnableAnonymousReq || ws.Authed == "" {
		return false, ""
	}
	reason, exists, err := cache.WrapGetBanned(db.DbQueries.GetUserBannedByKey)(ctx, ws.Authed)
	if err != nil {
		return true, fmt.Sprintf("error: %s", err.Error())
	}
	if exists {
		return true, fmt.Sprintf("banned: %s", reason)
	}
	return false, ""
}

func (p Policies) rejectExpiredEvent(event *nostr.Event) (bool, string) {
	for _, tag := range event.Tags {
		if len(tag) < 2 || tag[0] != "expiration" {
			continue
		}
		expiration, err := strconv.ParseInt(tag[1], 10, 64)
		if err != nil {
			return true, "invalid expiration tag"
		}
		if expiration < time.Now().Unix() {
			return true, "expired event"
		}
		break
	}
	return false, ""
}

func (p Policies) checkMinimumPow(evt *nostr.Event) (bool, string) {
	if p.Config.Relay.MinimumPOWLimit == 0 {
		return false, ""
	}
	if err := nip13.Check(evt.ID, p.Config.Relay.MinimumPOWLimit); err != nil {
		return true, "blocked: minimum POW not obtained"
	}
	return false, ""
}

func (p Policies) preventLargeTags(event *nostr.Event) (bool, string) {
	for _, tag := range event.Tags {
		if len(tag) > 1 && len(tag[0]) == 1 && len(tag[1]) > p.Config.Relay.MaxTagValueLength {
			return true, "event contains too large tags"
		}
	}
	return false, ""
}

func (p Policies) preventTooManyIndexableTags(event *nostr.Event) (bool, string) {
	count := 0
	for _, tag := range event.Tags {
		if len(tag) > 0 && len(tag[0]) == 1 {
			count++
		}
	}
	if count > p.Config.Relay.FilterLimit {
		return true, "too many indexable tags"
	}
	return false, ""
}

func (p Policies) rejectEventsWithBase64Media(evt *nostr.Event) (bool, string) {
	if strings.Contains(evt.Content, "data:image/") || strings.Contains(evt.Content, "data:video/") {
		return true, "event with base64 media"
	}
	return false, ""
}

func (p Policies) rejectReqWithoutAuth(filters nostr.Filters, ws *dto.WsServer) (bool, string) {
	if !p.Config.Ws.RequireAuthForReq() || ws.Authed != "" {
		return false, ""
	}
	_ = filters
	return true, "auth-required: this relay requires NIP-42 authentication before REQ"
}

func (p Policies) noEmptyFilters(filter nostr.Filter) (bool, string) {
	if p.Config.Relay.EnableEmptyFilter {
		return false, ""
	}
	count := len(filter.Kinds) + len(filter.IDs) + len(filter.Authors)
	for _, tagItems := range filter.Tags {
		count += len(tagItems)
	}
	if count == 0 {
		return true, "can't handle empty filters"
	}
	return false, ""
}

func (p Policies) antiSyncBots(filter nostr.Filter) (bool, string) {
	if p.Config.Ws.RequireAuthForReq() && (len(filter.Kinds) == 0 || slices.Contains(filter.Kinds, nostr.KindTextNote)) && len(filter.Authors) == 0 {
		return true, "auth-required: an author must be specified to get their kind:1 notes"
	}
	return false, ""
}

func (p Policies) checkKindsAuth(filter nostr.Filter, ws *dto.WsServer) (bool, string) {
	if ws.Authed == "" {
		return false, ""
	}
	receivers := filter.Tags["p"]
	if !slices.Contains(filter.Authors, ws.Authed) || !slices.Contains(receivers, ws.Authed) {
		return false, ""
	}
	for _, kind := range p.Config.Relay.ProtectedKinds {
		if slices.Contains(filter.Kinds, kind) {
			return true, fmt.Sprintf("auth-required: authentication is required to access kind %d", kind)
		}
	}
	return false, ""
}

func (p Policies) checkDirectMessageAccess(filter nostr.Filter, ws *dto.WsServer) (bool, string) {
	if !p.Config.Ws.RequireAuthForReq() || !slices.Contains(filter.Kinds, nostr.KindEncryptedDirectMessage) {
		return false, ""
	}

	senders := filter.Authors
	receivers, _ := filter.Tags["p"]
	switch {
	case ws.Authed == "":
		return true, "restricted: this relay does not serve kind-4 to unauthenticated users, does your client implement NIP-42?"
	case len(senders) == 1 && len(receivers) < 2 && senders[0] == ws.Authed:
		return false, ""
	case len(receivers) == 1 && len(senders) < 2 && receivers[0] == ws.Authed:
		return false, ""
	default:
		return true, "restricted: authenticated user does not have authorization for requested filters."
	}
}
