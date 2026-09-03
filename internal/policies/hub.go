package policies

import (
	"context"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gabrielmoura/nostr-relay-server/internal/security"
	"github.com/gabrielmoura/nostr-relay-server/internal/wot"
	"github.com/minio/sha256-simd"
	"github.com/nbd-wtf/go-nostr"
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
	if reject, reason := p.validateStorageEvent(ctx, evt); reject {
		return true, reason
	}
	return groups.ValidateIncomingEvent(ctx, evt)
}

// RejectProtectedEvent implements NIP-70 validation.
// It must be called from the event handler where the authenticated pubkey is available.
func (p Policies) RejectProtectedEvent(evt *nostr.Event, authedPubkey string) (bool, string) {
	return p.rejectProtectedEvent(evt, authedPubkey)
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
	if security.S != nil {
		var reject bool
		var reason string
		normalized, reject, reason = security.S.ValidateRequest(ctx, ws, normalized)
		if reject {
			return nil, true, reason
		}
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
		if reject, reason := groups.ValidateFilter(ctx, ws.Authed, filter); reject {
			return nil, true, reason
		}
	}

	return normalized, false, ""
}

func (p Policies) normalizeFilter(filter nostr.Filter) nostr.Filter {
	if security.S != nil {
		return security.S.NormalizeFilter(filter)
	}
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
		metrics.NostrSecuritySignatureChecksTotal.WithLabelValues("invalid_id").Inc()
		return true, security.Reason(security.PrefixInvalid, "event id is computed incorrectly")
	}
	if ok, err := evt.CheckSignature(); err != nil {
		metrics.NostrRelayEventSignatureFailures.Inc()
		metrics.NostrSecuritySignatureChecksTotal.WithLabelValues("error").Inc()
		return true, security.Reason(security.PrefixInvalid, "failed to verify signature")
	} else if !ok {
		metrics.NostrRelayEventSignatureFailures.Inc()
		metrics.NostrSecuritySignatureChecksTotal.WithLabelValues("invalid").Inc()
		return true, security.Reason(security.PrefixInvalid, "signature is invalid")
	}
	metrics.NostrSecuritySignatureChecksTotal.WithLabelValues("valid").Inc()
	return false, ""
}

func (p Policies) validateStorageEvent(ctx context.Context, evt *nostr.Event) (bool, string) {
	encoded, _ := json.Marshal(evt)
	if len(encoded) > p.Config.Relay.MaxEventSize {
		return true, security.Reason(security.PrefixRestricted, "event exceeds configured size")
	}
	if security.S != nil {
		if reject, reason := security.S.ValidateEventPayload(ctx, evt); reject {
			return true, reason
		}
	}
	if security.BypassFromContext(ctx).PublicationRestrictionsBypassed() {
		return false, ""
	}
	if reject, reason := p.rejectEventBannedUser(ctx, evt); reject {
		return true, reason
	}
	if reject, reason := p.rejectBannedEventID(ctx, evt); reject {
		return true, reason
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
	if reject, reason := p.rejectRepostOfProtectedEvent(evt); reject {
		return true, reason
	}
	if reject, reason := p.validateMarmotMIP00Event(evt); reject {
		return true, reason
	}
	if reject, reason := p.validateWhitelistBlacklist(evt); reject {
		return true, reason
	}
	if !wot.Validate(evt.PubKey) {
		return true, security.Reason(security.PrefixRestricted, "pubkey not in web of trust")
	}
	return false, ""
}
