package policies

import (
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/nbd-wtf/go-nostr"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

const testPrivKey = "98ee1fe7cf2b55d0242ec7771ce0e1f709451390d8e9b29fc719fc18382f0789"

func TestValidateMarmotMIP00EventDisabledKeepsGenericBehavior(t *testing.T) {
	cfg := newTestPolicyConfig()
	cfg.Marmot.Enabled = false
	cfg.Marmot.MIP00.Enabled = false

	p := Policies{Config: cfg}
	evt := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      marmotKindKeyPackage,
		Tags: nostr.Tags{
			{"d", "slot-1"},
		},
		Content: "payload",
	})

	reject, reason := p.validateMarmotMIP00Event(evt)
	if reject {
		t.Fatalf("validateMarmotMIP00Event() reject = true, reason = %q", reason)
	}
}

func TestValidateMarmotMIP00EventKeyPackageValid(t *testing.T) {
	cfg := newTestPolicyConfig()

	p := Policies{Config: cfg}
	before := testutil.ToFloat64(metrics.NostrMarmotMIP00EventsTotal.WithLabelValues("30443", "accepted"))
	evt := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      marmotKindKeyPackage,
		Tags: nostr.Tags{
			{"d", "c9a1e3f72b8d4056e6a2c891f3d75b0e4f2a1c8d9e7b3f6a4c2d1e8f5b9a7c3"},
			{"encoding", "base64"},
			{"mls_protocol_version", "1.0"},
			{"mls_ciphersuite", "0x0001"},
			{"mls_extensions", "0xf2ee", "0x000a"},
			{"mls_proposals", "0x000a"},
			{"i", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6"},
			{"relays", "wss://relay1.example", "wss://relay2.example"},
		},
		Content: "U0dWc2JHOGdUVWxRTURBPQ==",
	})

	reject, reason := p.validateMarmotMIP00Event(evt)
	if reject {
		t.Fatalf("validateMarmotMIP00Event() reject = true, reason = %q", reason)
	}
	after := testutil.ToFloat64(metrics.NostrMarmotMIP00EventsTotal.WithLabelValues("30443", "accepted"))
	if after != before+1 {
		t.Fatalf("accepted metric delta = %v, want %v", after-before, 1.0)
	}
}

func TestValidateMarmotMIP00EventKeyPackageRejectsMissingITag(t *testing.T) {
	cfg := newTestPolicyConfig()

	p := Policies{Config: cfg}
	before := testutil.ToFloat64(metrics.NostrMarmotMIP00EventsTotal.WithLabelValues("30443", "rejected"))
	evt := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      marmotKindKeyPackage,
		Tags: nostr.Tags{
			{"d", "slot-1"},
			{"encoding", "base64"},
			{"mls_protocol_version", "1.0"},
			{"mls_ciphersuite", "0x0001"},
			{"mls_extensions", "0xf2ee", "0x000a"},
			{"mls_proposals", "0x000a"},
			{"relays", "wss://relay1.example"},
		},
		Content: "U0dWc2JHOGdUVWxRTURBPQ==",
	})

	reject, reason := p.validateMarmotMIP00Event(evt)
	if !reject {
		t.Fatal("validateMarmotMIP00Event() reject = false, want true")
	}
	if reason != "invalid: marmot mip00 keypackage missing required i tag" {
		t.Fatalf("validateMarmotMIP00Event() reason = %q", reason)
	}
	after := testutil.ToFloat64(metrics.NostrMarmotMIP00EventsTotal.WithLabelValues("30443", "rejected"))
	if after != before+1 {
		t.Fatalf("rejected metric delta = %v, want %v", after-before, 1.0)
	}
}

func TestValidateMarmotMIP00EventRelayListRejectsInvalidRelayURL(t *testing.T) {
	cfg := newTestPolicyConfig()

	p := Policies{Config: cfg}
	evt := mustSignEvent(t, nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      marmotKindKeyPackageList,
		Tags: nostr.Tags{
			{"relay", "https://relay1.example"},
		},
		Content: "",
	})

	reject, reason := p.validateMarmotMIP00Event(evt)
	if !reject {
		t.Fatal("validateMarmotMIP00Event() reject = false, want true")
	}
	if reason != "invalid: marmot mip00 relay list contains invalid relay url" {
		t.Fatalf("validateMarmotMIP00Event() reason = %q", reason)
	}
}

func newTestPolicyConfig() *config.Config {
	return &config.Config{
		Marmot: config.MarmotConfig{
			Enabled: true,
			MIP00: config.MarmotMIP00Config{
				Enabled:                  true,
				AcceptKind30443:          true,
				AcceptKind10051:          true,
				AcceptLegacyKind443:      false,
				ValidationMode:           "basic",
				RequireITag:              true,
				RequireBase64EncodingTag: true,
				RequireRelaysTag:         true,
				RequireMLSExtensions:     true,
				RequireMLSProposals:      true,
				RequireWSRelayURLs:       true,
				MaxRelaysPerEvent:        10,
				MaxContentSizeBytes:      1024,
			},
		},
		Relay: config.RelayConfig{
			MaxEventSize:      1024 * 1024,
			MaxTagValueLength: 512,
			FilterLimit:       100,
		},
	}
}

func mustSignEvent(t *testing.T, evt nostr.Event) *nostr.Event {
	t.Helper()
	if err := evt.Sign(testPrivKey); err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	return &evt
}
