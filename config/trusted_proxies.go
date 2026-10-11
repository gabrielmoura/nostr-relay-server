package config

import (
	"fmt"
	"net/netip"
	"strings"
)

// ValidateTrustedProxies rejects malformed proxy peers before Fiber starts.
// Entries may be individual IP addresses or CIDR prefixes.
func (cfg *Config) ValidateTrustedProxies() error {
	if cfg == nil {
		return nil
	}

	for _, proxy := range cfg.Security.TrustedProxies {
		trimmedProxy := strings.TrimSpace(proxy)
		if trimmedProxy == "" {
			return fmt.Errorf("security.trusted_proxies must not contain an empty value")
		}
		if proxy != trimmedProxy {
			return fmt.Errorf("security.trusted_proxies must not contain whitespace-padded values: %q", proxy)
		}
		if _, err := netip.ParseAddr(trimmedProxy); err == nil {
			continue
		}
		if _, err := netip.ParsePrefix(trimmedProxy); err == nil {
			continue
		}
		return fmt.Errorf("security.trusted_proxies contains invalid IP or CIDR %q", proxy)
	}
	return nil
}
