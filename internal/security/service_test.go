package security

import (
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func TestNewWhitelistNormalizesPubkeysAndCIDRs(t *testing.T) {
	t.Parallel()

	const pubkey = "7ef721e77149c737014971b141b0b590a5ebe82b79130228cdbe56e9be2d8e50"
	npub, err := nip19.EncodePublicKey(pubkey)
	if err != nil {
		t.Fatalf("encode npub: %v", err)
	}

	whitelist, err := newWhitelist(config.SecurityWhitelistConfig{
		Enabled: true,
		PubKeys: []string{npub},
		IPs:     []string{"127.0.0.1"},
		CIDRs:   []string{"10.0.0.0/8"},
	})
	if err != nil {
		t.Fatalf("new whitelist: %v", err)
	}

	if !whitelist.isPubKeyWhitelisted(pubkey) {
		t.Fatal("expected hex pubkey to match whitelisted npub")
	}
	if !whitelist.isIPWhitelisted("127.0.0.1") {
		t.Fatal("expected exact IP whitelist match")
	}
	if !whitelist.isIPWhitelisted("10.1.2.3") {
		t.Fatal("expected CIDR whitelist match")
	}
}

func TestNormalizeFilterClampsToSecurityMaxLimit(t *testing.T) {
	t.Parallel()

	svc := &Service{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				Enabled: true,
				Limits: config.SecurityLimitsConfig{
					MaxLimit: 500,
				},
			},
			Relay: config.RelayConfig{
				FilterLimit: 1000,
			},
		},
	}

	filter := svc.NormalizeFilter(nostr.Filter{Limit: 9999})
	if filter.Limit != 500 {
		t.Fatalf("expected limit 500, got %d", filter.Limit)
	}
}
