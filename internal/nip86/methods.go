package nip86

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"go.uber.org/zap"
)

func (s *Service) handleBanPubKey(ctx context.Context, params []json.RawMessage, callCtx CallContext) Response {
	pubkey, reason, err := parseTargetAndReason(params)
	if err != nil {
		return errorResponse(err)
	}
	pubkey, err = normalizeHex32(pubkey, "pubkey")
	if err != nil {
		return errorResponse(err)
	}
	if err := s.repo.BanUserByPubKey(ctx, pubkey, reason, nil); err != nil {
		return internalErrorResponse(err)
	}
	logMutation("banpubkey", pubkey, callCtx.AdminPubKey, reason, callCtx.RemoteIP)
	return okResponse(true)
}

func (s *Service) handleUnbanPubKey(ctx context.Context, params []json.RawMessage, callCtx CallContext) Response {
	pubkey, reason, err := parseTargetAndReason(params)
	if err != nil {
		return errorResponse(err)
	}
	pubkey, err = normalizeHex32(pubkey, "pubkey")
	if err != nil {
		return errorResponse(err)
	}
	if err := s.repo.UnbanUserByPubKey(ctx, pubkey); err != nil {
		return internalErrorResponse(err)
	}
	logMutation("unbanpubkey", pubkey, callCtx.AdminPubKey, reason, callCtx.RemoteIP)
	return okResponse(true)
}

func (s *Service) handleListBannedPubKeys(ctx context.Context) Response {
	items, err := s.repo.ListBannedPubKeys(ctx)
	if err != nil {
		return internalErrorResponse(err)
	}
	return okResponse(items)
}

func (s *Service) handleAllowPubKey(ctx context.Context, params []json.RawMessage, callCtx CallContext) Response {
	pubkey, reason, err := parseTargetAndReason(params)
	if err != nil {
		return errorResponse(err)
	}
	pubkey, err = normalizeHex32(pubkey, "pubkey")
	if err != nil {
		return errorResponse(err)
	}
	if err := s.repo.UpsertAllowedPubKey(ctx, pubkey, reason, callCtx.AdminPubKey); err != nil {
		return internalErrorResponse(err)
	}
	setCachedEntry(allowedPubKeyCacheKey(pubkey), cacheEntry{State: cacheStatePresent, Reason: reason})
	logMutation("allowpubkey", pubkey, callCtx.AdminPubKey, reason, callCtx.RemoteIP)
	return okResponse(true)
}

func (s *Service) handleUnallowPubKey(ctx context.Context, params []json.RawMessage, callCtx CallContext) Response {
	pubkey, err := parseSingleString(params, 0, "pubkey")
	if err != nil {
		return errorResponse(err)
	}
	pubkey, err = normalizeHex32(pubkey, "pubkey")
	if err != nil {
		return errorResponse(err)
	}
	if err := s.repo.DeleteAllowedPubKey(ctx, pubkey); err != nil {
		return internalErrorResponse(err)
	}
	invalidateKeys(allowedPubKeyCacheKey(pubkey))
	logMutation("unallowpubkey", pubkey, callCtx.AdminPubKey, "", callCtx.RemoteIP)
	return okResponse(true)
}

func (s *Service) handleListAllowedPubKeys(ctx context.Context) Response {
	items, err := s.repo.ListAllowedPubKeys(ctx)
	if err != nil {
		return internalErrorResponse(err)
	}
	return okResponse(items)
}

func (s *Service) handleAllowEvent(ctx context.Context, params []json.RawMessage, callCtx CallContext) Response {
	eventID, err := parseSingleString(params, 0, "event id")
	if err != nil {
		return errorResponse(err)
	}
	eventID, err = normalizeHex32(eventID, "event id")
	if err != nil {
		return errorResponse(err)
	}
	if err := s.repo.DeleteBannedEvent(ctx, eventID); err != nil {
		return internalErrorResponse(err)
	}
	invalidateKeys(bannedEventCacheKey(eventID))
	logMutation("allowevent", eventID, callCtx.AdminPubKey, "", callCtx.RemoteIP)
	return okResponse(true)
}

func (s *Service) handleBanEvent(ctx context.Context, params []json.RawMessage, callCtx CallContext) Response {
	eventID, reason, err := parseTargetAndReason(params)
	if err != nil {
		return errorResponse(err)
	}
	eventID, err = normalizeHex32(eventID, "event id")
	if err != nil {
		return errorResponse(err)
	}
	if err := s.repo.UpsertBannedEvent(ctx, eventID, reason, callCtx.AdminPubKey); err != nil {
		return internalErrorResponse(err)
	}
	_ = s.repo.DeleteEvent(ctx, eventID, eventID)
	setCachedEntry(bannedEventCacheKey(eventID), cacheEntry{State: cacheStatePresent, Reason: reason})
	logMutation("banevent", eventID, callCtx.AdminPubKey, reason, callCtx.RemoteIP)
	return okResponse(true)
}

func (s *Service) handleListBannedEvents(ctx context.Context) Response {
	items, err := s.repo.ListBannedEvents(ctx)
	if err != nil {
		return internalErrorResponse(err)
	}
	return okResponse(items)
}

func (s *Service) handleChangeRelayName(ctx context.Context, params []json.RawMessage, callCtx CallContext) Response {
	name, err := parseSingleString(params, 0, "name")
	if err != nil {
		return errorResponse(err)
	}
	return s.updateRelayMetadata(ctx, name, s.cfg.RelayInformation.Description, callCtx, "changerelayname", name)
}

func (s *Service) handleChangeRelayDescription(ctx context.Context, params []json.RawMessage, callCtx CallContext) Response {
	description, err := parseSingleString(params, 0, "description")
	if err != nil {
		return errorResponse(err)
	}
	return s.updateRelayMetadata(ctx, s.cfg.RelayInformation.Name, description, callCtx, "changerelaydescription", description)
}

func (s *Service) handleBlockIP(ctx context.Context, params []json.RawMessage, callCtx CallContext) Response {
	ip, reason, err := parseTargetAndReason(params)
	if err != nil {
		return errorResponse(err)
	}
	ip, err = normalizeIP(ip)
	if err != nil {
		return errorResponse(err)
	}
	if err := s.repo.UpsertBlockedIP(ctx, ip, reason, callCtx.AdminPubKey); err != nil {
		return internalErrorResponse(err)
	}
	setCachedEntry(blockIPCacheKey(ip), cacheEntry{State: cacheStatePresent, Reason: reason})
	listener.DisconnectByIP(ip)
	logMutation("blockip", ip, callCtx.AdminPubKey, reason, callCtx.RemoteIP)
	return okResponse(true)
}

func (s *Service) handleUnblockIP(ctx context.Context, params []json.RawMessage, callCtx CallContext) Response {
	ip, err := parseSingleString(params, 0, "ip")
	if err != nil {
		return errorResponse(err)
	}
	ip, err = normalizeIP(ip)
	if err != nil {
		return errorResponse(err)
	}
	if err := s.repo.DeleteBlockedIP(ctx, ip); err != nil {
		return internalErrorResponse(err)
	}
	invalidateKeys(blockIPCacheKey(ip))
	logMutation("unblockip", ip, callCtx.AdminPubKey, "", callCtx.RemoteIP)
	return okResponse(true)
}

func (s *Service) handleListBlockedIPs(ctx context.Context) Response {
	items, err := s.repo.ListBlockedIPs(ctx)
	if err != nil {
		return internalErrorResponse(err)
	}
	return okResponse(items)
}

func (s *Service) updateRelayMetadata(ctx context.Context, name, description string, callCtx CallContext, action string, reason string) Response {
	relayURL := strings.TrimSpace(s.cfg.RelayInformation.URL)
	if relayURL == "" {
		return Response{HTTPStatus: 500, Result: nil, Error: "relay_information.url is not configured"}
	}
	if err := s.repo.UpsertRelayMetadata(ctx, relayURL, name, description, callCtx.AdminPubKey); err != nil {
		return internalErrorResponse(err)
	}
	s.cfg.RelayInformation.Name = name
	s.cfg.RelayInformation.Description = description
	logMutation(action, relayURL, callCtx.AdminPubKey, reason, callCtx.RemoteIP)
	return okResponse(true)
}

func okResponse(result any) Response {
	return Response{HTTPStatus: 200, Result: result}
}

func errorResponse(err error) Response {
	return Response{HTTPStatus: 400, Result: nil, Error: err.Error()}
}

func internalErrorResponse(err error) Response {
	return Response{HTTPStatus: 500, Result: nil, Error: err.Error()}
}

func logMutation(action, target, admin, reason, ip string) {
	fields := []zap.Field{
		zap.String("action", action),
		zap.String("target", target),
		zap.String("admin", admin),
		zap.String("reason", reason),
		zap.String("ip", ip),
	}
	log.Logger.Info("nip86 mutation", fields...)
}
