package security

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr/nip19"
)

type whitelist struct {
	enabled bool
	pubkeys map[string]struct{}
	ips     map[string]struct{}
	cidrs   []netip.Prefix
}

func newWhitelist(cfg config.SecurityWhitelistConfig) (*whitelist, error) {
	w := &whitelist{
		enabled: cfg.Enabled,
		pubkeys: map[string]struct{}{},
		ips:     map[string]struct{}{},
	}
	if !cfg.Enabled {
		return w, nil
	}

	for _, raw := range cfg.PubKeys {
		pubkey, err := normalizePublicKey(raw)
		if err != nil {
			return nil, err
		}
		w.pubkeys[pubkey] = struct{}{}
	}

	for _, raw := range cfg.IPs {
		ip := strings.TrimSpace(raw)
		if ip == "" {
			continue
		}
		w.ips[ip] = struct{}{}
	}

	for _, raw := range cfg.CIDRs {
		cidr := strings.TrimSpace(raw)
		if cidr == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid whitelist cidr %q: %w", cidr, err)
		}
		w.cidrs = append(w.cidrs, prefix)
	}

	return w, nil
}

func (w *whitelist) isIPWhitelisted(ip string) bool {
	if w == nil || !w.enabled {
		return false
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return false
	}
	if _, ok := w.ips[ip]; ok {
		return true
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	for _, prefix := range w.cidrs {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func (w *whitelist) isPubKeyWhitelisted(pubkey string) bool {
	if w == nil || !w.enabled {
		return false
	}
	pubkey = strings.ToLower(strings.TrimSpace(pubkey))
	if pubkey == "" {
		return false
	}
	_, ok := w.pubkeys[pubkey]
	return ok
}

func normalizePublicKey(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", fmt.Errorf("empty whitelist pubkey")
	}
	if strings.HasPrefix(value, "npub") {
		prefix, decoded, err := nip19.Decode(value)
		if err != nil {
			return "", fmt.Errorf("invalid whitelist npub %q: %w", value, err)
		}
		if prefix != "npub" {
			return "", fmt.Errorf("invalid whitelist pubkey prefix %q", prefix)
		}
		pubkey, ok := decoded.(string)
		if !ok || strings.TrimSpace(pubkey) == "" {
			return "", fmt.Errorf("invalid whitelist npub payload")
		}
		return strings.ToLower(pubkey), nil
	}
	return value, nil
}
