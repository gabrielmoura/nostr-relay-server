package policies

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gabrielmoura/nostr-relay-server/internal/nip86"
	"github.com/gabrielmoura/nostr-relay-server/internal/security"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip13"
)

func (p Policies) validateWhitelistBlacklist(evt *nostr.Event) (bool, string) {
	if len(p.Config.Relay.WhitelistKinds) > 0 && !slices.Contains(p.Config.Relay.WhitelistKinds, evt.Kind) {
		metrics.NostrBlockedKindsTotal.WithLabelValues("whitelist").Inc()
		return true, security.Reason(security.PrefixRestricted, "event kind is not in the whitelist")
	}
	if len(p.Config.Relay.BlacklistKinds) > 0 && slices.Contains(p.Config.Relay.BlacklistKinds, evt.Kind) {
		metrics.NostrBlockedKindsTotal.WithLabelValues("blacklist").Inc()
		return true, security.Reason(security.PrefixRestricted, "event kind is blacklisted")
	}
	return false, ""
}

func (p Policies) rejectEventBannedUser(ctx context.Context, evt *nostr.Event) (bool, string) {
	if evt.PubKey == "" {
		return true, security.Reason(security.PrefixInvalid, "missing public key")
	}
	reason, exists, err := cache.WrapGetBanned(db.DbQueries.GetUserBannedByKey)(ctx, evt.PubKey)
	if err != nil {
		return true, fmt.Sprintf("error: %s", err.Error())
	}
	if exists {
		return true, security.Reason(security.PrefixBlocked, reason)
	}
	return false, ""
}

func (p Policies) rejectBannedEventID(ctx context.Context, evt *nostr.Event) (bool, string) {
	if evt == nil || evt.ID == "" || nip86.S == nil || !nip86.S.Enabled() {
		return false, ""
	}
	reason, exists, err := nip86.S.IsEventBanned(ctx, evt.ID)
	if err != nil {
		return true, security.Reason(security.PrefixRestricted, "event moderation lookup failed")
	}
	if exists {
		return true, security.Reason(security.PrefixBlocked, firstNonEmpty(reason, "event is banned"))
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
		return true, security.Reason(security.PrefixBlocked, reason)
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
			return true, security.Reason(security.PrefixInvalid, "invalid expiration tag")
		}
		if expiration < time.Now().Unix() {
			return true, security.Reason(security.PrefixInvalid, "event has already expired")
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
		return true, security.Reason(security.PrefixPoW, "minimum proof of work not satisfied")
	}
	return false, ""
}

func (p Policies) preventLargeTags(event *nostr.Event) (bool, string) {
	for _, tag := range event.Tags {
		if len(tag) > 1 && len(tag[0]) == 1 && len(tag[1]) > p.Config.Relay.MaxTagValueLength {
			return true, security.Reason(security.PrefixRestricted, "event contains oversized indexed tags")
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
		return true, security.Reason(security.PrefixRestricted, "event exceeds indexed tag limit")
	}
	return false, ""
}

func (p Policies) rejectEventsWithBase64Media(evt *nostr.Event) (bool, string) {
	if strings.Contains(evt.Content, "data:image/") || strings.Contains(evt.Content, "data:video/") {
		return true, security.Reason(security.PrefixRestricted, "event content embeds base64 media")
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

// rejectProtectedEvent implements NIP-70: Protected Events.
// Events carrying the ["-"] tag can only be published by their author.
// If the client is not authenticated, the relay sends an auth-required message.
// If authenticated but the pubkey does not match, the event is rejected.
func (p Policies) rejectProtectedEvent(evt *nostr.Event, authedPubkey string) (bool, string) {
	if !hasProtectedTag(evt) {
		return false, ""
	}
	if authedPubkey == "" {
		return true, "auth-required: this event may only be published by its author"
	}
	if authedPubkey != evt.PubKey {
		return true, security.Reason(security.PrefixRestricted, "protected event can only be published by its author")
	}
	return false, ""
}

// rejectRepostOfProtectedEvent implements the NIP-70 repost rule:
// reposts (kind 6 and kind 16) MUST NOT embed a protected event.
func (p Policies) rejectRepostOfProtectedEvent(evt *nostr.Event) (bool, string) {
	if evt.Kind != nostr.KindRepost && evt.Kind != nostr.KindGenericRepost {
		return false, ""
	}
	if strings.TrimSpace(evt.Content) == "" {
		return false, ""
	}
	var embedded nostr.Event
	if err := json.Unmarshal([]byte(evt.Content), &embedded); err != nil {
		// Content is not a valid embedded event; nothing to check.
		return false, ""
	}
	if hasProtectedTag(&embedded) {
		return true, security.Reason(security.PrefixRestricted, "repost must not embed a protected event")
	}
	return false, ""
}

// hasProtectedTag returns true when the event contains the NIP-70 ["-"] tag.
func hasProtectedTag(evt *nostr.Event) bool {
	for _, tag := range evt.Tags {
		if len(tag) == 1 && tag[0] == "-" {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
