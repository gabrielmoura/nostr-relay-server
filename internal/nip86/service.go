package nip86

import (
	"context"
	"errors"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
)

type Service struct {
	repo    Repository
	cfg     *config.Config
	enabled bool
}

var S *Service

func Init(repo Repository) error {
	if config.Cfg == nil {
		return errors.New("config is not loaded")
	}
	S = &Service{
		repo:    repo,
		cfg:     config.Cfg,
		enabled: config.Cfg.NIP86Enabled(),
	}
	if !S.enabled {
		return nil
	}
	if config.Cfg.AdminPubKey == "" {
		return errors.New("admin_pubkey is required when nip86 is enabled")
	}
	return nil
}

func ApplyRelayMetadataOverride(ctx context.Context) error {
	if S == nil || !S.enabled {
		return nil
	}
	relayURL := strings.TrimSpace(S.cfg.RelayInformation.URL)
	if relayURL == "" {
		return nil
	}
	record, exists, err := S.repo.GetRelayMetadata(ctx, relayURL)
	if err != nil || !exists {
		return err
	}
	if record.Name != "" {
		S.cfg.RelayInformation.Name = record.Name
	}
	if record.Description != "" {
		S.cfg.RelayInformation.Description = record.Description
	}
	return nil
}

func (s *Service) Enabled() bool {
	return s != nil && s.enabled
}

func (s *Service) Execute(ctx context.Context, req Request, callCtx CallContext) Response {
	if s == nil || !s.enabled {
		return Response{HTTPStatus: 404, Result: nil, Error: "nip86 is disabled"}
	}
	method := strings.ToLower(strings.TrimSpace(req.Method))
	if method == "" {
		return Response{HTTPStatus: 400, Result: nil, Error: "missing method"}
	}

	switch method {
	case MethodSupportedMethods:
		return Response{Result: supportedMethods}
	case MethodBanPubKey:
		return s.handleBanPubKey(ctx, req.Params, callCtx)
	case MethodUnbanPubKey:
		return s.handleUnbanPubKey(ctx, req.Params, callCtx)
	case MethodListBannedPubKeys:
		return s.handleListBannedPubKeys(ctx)
	case MethodAllowPubKey:
		return s.handleAllowPubKey(ctx, req.Params, callCtx)
	case MethodUnallowPubKey:
		return s.handleUnallowPubKey(ctx, req.Params, callCtx)
	case MethodListAllowedPubKeys:
		return s.handleListAllowedPubKeys(ctx)
	case MethodAllowEvent:
		return s.handleAllowEvent(ctx, req.Params, callCtx)
	case MethodBanEvent:
		return s.handleBanEvent(ctx, req.Params, callCtx)
	case MethodListBannedEvents:
		return s.handleListBannedEvents(ctx)
	case MethodChangeRelayName:
		return s.handleChangeRelayName(ctx, req.Params, callCtx)
	case MethodChangeRelayDesc:
		return s.handleChangeRelayDescription(ctx, req.Params, callCtx)
	case MethodBlockIP:
		return s.handleBlockIP(ctx, req.Params, callCtx)
	case MethodUnblockIP:
		return s.handleUnblockIP(ctx, req.Params, callCtx)
	case MethodListBlockedIPs:
		return s.handleListBlockedIPs(ctx)
	default:
		return Response{HTTPStatus: 400, Result: nil, Error: "unsupported method"}
	}
}
