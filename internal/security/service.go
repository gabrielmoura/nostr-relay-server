package security

import (
	"context"
	"fmt"
	"sync"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/db/helper"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	redisinfra "github.com/gabrielmoura/nostr-relay-server/infra/redis"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
)

type Service struct {
	cfg          *config.Config
	whitelist    *whitelist
	defender     *defender
	connections  map[string]int
	connectionsM sync.Mutex
}

var S *Service

func Init() error {
	whitelist, err := newWhitelist(config.Cfg.Security.Whitelist)
	if err != nil {
		return err
	}
	S = &Service{
		cfg:         config.Cfg,
		whitelist:   whitelist,
		defender:    newDefender(config.Cfg.Security.Defense, redisinfra.GetClient()),
		connections: map[string]int{},
	}
	return nil
}

func (s *Service) IsIPWhitelisted(ip string) bool {
	if s == nil {
		return false
	}
	return s.whitelist.isIPWhitelisted(ip)
}

func (s *Service) IsPubKeyWhitelisted(pubkey string) bool {
	if s == nil {
		return false
	}
	return s.whitelist.isPubKeyWhitelisted(pubkey)
}

func (s *Service) BypassFor(ip, pubkey string) BypassContext {
	bypass := BypassContext{
		IPWhitelisted:     s.IsIPWhitelisted(ip),
		PubKeyWhitelisted: s.IsPubKeyWhitelisted(pubkey),
	}
	if bypass.IPWhitelisted {
		metrics.NostrSecurityWhitelistBypassTotal.WithLabelValues("ip", "all").Inc()
	}
	if bypass.PubKeyWhitelisted {
		metrics.NostrSecurityWhitelistBypassTotal.WithLabelValues("pubkey", "event").Inc()
	}
	return bypass
}

func (s *Service) MaxMessageLength() int {
	if s == nil || !s.cfg.Security.Enabled || s.cfg.Security.Limits.MaxMessageLength <= 0 {
		return 1024 * 1024
	}
	return s.cfg.Security.Limits.MaxMessageLength
}

func (s *Service) AcquireConnection(ip string) (bool, string) {
	if s == nil || s.cfg == nil || !s.cfg.Security.Enabled {
		return true, ""
	}
	if s.IsIPWhitelisted(ip) {
		metrics.NostrSecurityWhitelistBypassTotal.WithLabelValues("ip", "connection").Inc()
		return true, ""
	}
	maxPerIP := s.cfg.Security.Limits.MaxConnectionsPerIP
	if maxPerIP <= 0 {
		return true, ""
	}

	s.connectionsM.Lock()
	defer s.connectionsM.Unlock()
	if s.connections[ip] >= maxPerIP {
		metrics.NostrSecurityConnectionsRejectedTotal.WithLabelValues("max_connections_per_ip").Inc()
		return false, Reason(PrefixRateLimited, "too many concurrent connections for this IP")
	}
	s.connections[ip]++
	return true, ""
}

func (s *Service) ReleaseConnection(ip string) {
	if s == nil || ip == "" {
		return
	}
	s.connectionsM.Lock()
	defer s.connectionsM.Unlock()
	if s.connections[ip] <= 1 {
		delete(s.connections, ip)
		return
	}
	s.connections[ip]--
}

func (s *Service) ValidateRequest(ctx context.Context, ws *dto.WsServer, filters nostr.Filters) (nostr.Filters, bool, string) {
	if s == nil || !s.cfg.Security.Enabled {
		return filters, false, ""
	}

	bypass := s.BypassFor(ws.RemoteIP, ws.Authed)
	if bypass.RequestRestrictionsBypassed() {
		return filters, false, ""
	}

	maxFilters := s.cfg.Security.Limits.MaxFiltersPerReq
	if maxFilters > 0 && len(filters) > maxFilters {
		metrics.NostrSecurityReqRejectedTotal.WithLabelValues("max_filters_per_req").Inc()
		return nil, true, Reason(PrefixRateLimited, "too many filters")
	}

	for i := range filters {
		filters[i] = s.NormalizeFilter(filters[i])
		if err := helper.ValidateFilterLimits(&s.cfg.Relay, filters[i]); err != nil {
			metrics.NostrSecurityReqRejectedTotal.WithLabelValues("filter_limits").Inc()
			return nil, true, Reason(PrefixRestricted, fmt.Sprintf("query exceeds configured limits: %s", err.Error()))
		}
	}

	if bypass.RateLimitsBypassed() {
		return filters, false, ""
	}
	decision, err := s.defender.evaluateReq(ctx, ws.RemoteIP, ws.Authed)
	if err != nil {
		return nil, true, Reason(PrefixRestricted, "temporary request defense unavailable")
	}
	if decision.action != "" {
		metrics.NostrSecurityReqRejectedTotal.WithLabelValues(decision.action).Inc()
		return nil, true, Reason(decision.prefix, decision.msg)
	}

	return filters, false, ""
}

func (s *Service) ValidateEvent(ctx context.Context, ws *dto.WsServer, evt *nostr.Event) (context.Context, bool, string) {
	if s == nil || !s.cfg.Security.Enabled {
		return ctx, false, ""
	}
	bypass := s.BypassFor(ws.RemoteIP, evt.PubKey)
	ctx = WithBypassContext(ctx, bypass)
	if bypass.RateLimitsBypassed() {
		return ctx, false, ""
	}
	decision, err := s.defender.evaluateEvent(ctx, ws.RemoteIP, evt.PubKey)
	if err != nil {
		return ctx, true, Reason(PrefixRestricted, "temporary event defense unavailable")
	}
	if decision.action != "" {
		metrics.NostrSecurityEventRejectedTotal.WithLabelValues(decision.action).Inc()
		return ctx, true, Reason(decision.prefix, decision.msg)
	}
	return ctx, false, ""
}

func (s *Service) ValidateEventPayload(ctx context.Context, evt *nostr.Event) (bool, string) {
	if s == nil || !s.cfg.Security.Enabled {
		return false, ""
	}
	bypass := BypassFromContext(ctx)
	if bypass.PublicationRestrictionsBypassed() {
		return false, ""
	}
	maxTags := s.cfg.Security.Limits.MaxEventTags
	if maxTags > 0 && len(evt.Tags) > maxTags {
		metrics.NostrSecurityEventRejectedTotal.WithLabelValues("max_event_tags").Inc()
		return true, Reason(PrefixRestricted, "event exceeds configured tag limit")
	}
	maxContentLength := s.cfg.Security.Limits.MaxContentLength
	if maxContentLength > 0 && len([]rune(evt.Content)) > maxContentLength {
		metrics.NostrSecurityEventRejectedTotal.WithLabelValues("max_content_length").Inc()
		return true, Reason(PrefixRestricted, "event content exceeds configured length")
	}
	return false, ""
}

func (s *Service) NormalizeFilter(filter nostr.Filter) nostr.Filter {
	if filter.Limit == 0 {
		filter.Limit = s.cfg.Relay.FilterLimit
	}
	if s.cfg.Security.Limits.MaxLimit > 0 && filter.Limit > s.cfg.Security.Limits.MaxLimit {
		filter.Limit = s.cfg.Security.Limits.MaxLimit
	}
	if s.cfg.Relay.FilterLimit > 0 && filter.Limit > s.cfg.Relay.FilterLimit {
		filter.Limit = s.cfg.Relay.FilterLimit
	}
	return filter
}
