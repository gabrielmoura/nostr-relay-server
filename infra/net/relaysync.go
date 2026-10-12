package net

import (
	"net"
	"net/http"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	relaysyncfiber "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/fiber"
	"github.com/gofiber/fiber/v2"
)

// RegisterRelaySync mounts RelaySync on an existing Fiber app. It does not
// create a listener, configure TLS or change HTTP/2 settings. The allowlist
// middleware runs before capability discovery and all session operations.
func RegisterRelaySync(app *fiber.App, cfg config.RelaySyncConfig, adapter *core.HTTPAdapter, services relaysyncfiber.Services) error {
	if !cfg.Enabled || !cfg.HTTPEnabled {
		return nil
	}
	allowlist, err := newIPAllowlist(cfg.AllowedIPs)
	if err != nil {
		return err
	}
	app.Use("/sync/v2", func(c *fiber.Ctx) error {
		effectiveIP := c.IP()
		if !allowlist(effectiveIP) {
			return c.SendStatus(http.StatusForbidden)
		}
		// fasthttpadaptor exposes the socket RemoteAddr to net/http. Override a
		// reserved header after Fiber has applied its trusted-proxy policy so the
		// resolver cannot be spoofed by an inbound request header.
		c.Request().Header.Set("X-RelaySync-Effective-IP", effectiveIP)
		return c.Next()
	})
	return relaysyncfiber.RegisterServices(app, adapter, services)
}

func newIPAllowlist(values []string) (func(string) bool, error) {
	allowed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		allowed = append(allowed, value)
	}
	if len(allowed) == 0 {
		return nil, ErrRelaySyncAllowlist
	}
	// Parse via net.ParseIP/ParseCIDR once and retain normalized ranges.
	type rangeEntry struct {
		ip      net.IP
		network *net.IPNet
	}
	ranges := make([]rangeEntry, 0, len(allowed))
	for _, value := range allowed {
		if ip := net.ParseIP(value); ip != nil {
			ranges = append(ranges, rangeEntry{ip: ip})
			continue
		}
		ip, network, err := net.ParseCIDR(value)
		if err != nil {
			return nil, err
		}
		ranges = append(ranges, rangeEntry{ip: ip, network: network})
	}
	return func(remote string) bool {
		ip := net.ParseIP(remote)
		if ip == nil {
			return false
		}
		for _, entry := range ranges {
			if entry.network != nil && entry.network.Contains(ip) {
				return true
			}
			if entry.ip != nil && entry.ip.Equal(ip) {
				return true
			}
		}
		return false
	}, nil
}

var ErrRelaySyncAllowlist = &relaySyncConfigError{"relay_sync.allowed_ips must contain at least one IP or CIDR"}

type relaySyncConfigError struct{ message string }

func (e *relaySyncConfigError) Error() string { return e.message }
