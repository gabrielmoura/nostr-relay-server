package relaysync

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
)

// AllowlistAuthorizer is the deliberately simple NRServer integration policy:
// the authenticated peer identity is its source IP, and every RelaySync
// action is allowed only when that IP matches the configured address/prefix.
// It does not create or validate credentials.
type AllowlistAuthorizer struct{ prefixes []netip.Prefix }

func NewAllowlistAuthorizer(values []string) (*AllowlistAuthorizer, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("relaysync: at least one allowed IP is required")
	}
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if addr, err := netip.ParseAddr(value); err == nil {
			prefixes = append(prefixes, netip.PrefixFrom(addr.Unmap(), addr.BitLen()))
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, fmt.Errorf("relaysync: invalid allowed IP %q: %w", raw, err)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return &AllowlistAuthorizer{prefixes: prefixes}, nil
}

func (a *AllowlistAuthorizer) Authorize(ctx context.Context, principal core.Principal, _ core.Action, requested core.Scope) (core.AuthorizedScope, error) {
	if err := ctx.Err(); err != nil {
		return core.AuthorizedScope{}, err
	}
	addr, err := netip.ParseAddr(strings.TrimSpace(principal.ID))
	if err != nil {
		return core.AuthorizedScope{}, core.ErrUnauthorized
	}
	addr = addr.Unmap()
	if a.AllowedAddr(addr) {
		return core.AuthorizedScope{Scope: requested, Actions: []core.Action{core.ActionOpenSession, core.ActionPull, core.ActionPush, core.ActionReadChanges, core.ActionLive}}, nil
	}
	return core.AuthorizedScope{}, core.ErrUnauthorized
}

func (a *AllowlistAuthorizer) AllowedAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	for _, prefix := range a.prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
