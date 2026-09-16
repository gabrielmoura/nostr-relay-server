package policies

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/nbd-wtf/go-nostr"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestContentSimhash(t *testing.T) {
	first := contentSimhash("Nostr relay content spam detection works well")
	second := contentSimhash("Nostr relay content spam detection works very well")
	if distance := simhashDistance(first, second); distance >= 32 {
		t.Fatalf("similar content distance = %d, want less than 32", distance)
	}
	if simhashBandValue(first, 0) != uint16(first) {
		t.Fatal("first band value does not contain the low 16 bits")
	}
	if got := contentSimhash("  NOSTR\u200b relay\tcontent  "); got != contentSimhash("nostr relay content") {
		t.Fatal("simhash must be stable after content normalization")
	}
}

func TestContentSimilarityCheckerObserve(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.ContentSimilarityConfig
		cacheEnabled bool
		evt          *nostr.Event
		finderError  error
		finderCalls  int
		addCalls     int
		wantFlagged  bool
	}{
		{
			name:         "disabled",
			cfg:          config.ContentSimilarityConfig{},
			cacheEnabled: true,
			evt:          similarityTestEvent(nostr.KindTextNote, "pubkey-1", "similar content"),
		},
		{
			name:         "cache disabled",
			cfg:          similarityTestConfig(),
			cacheEnabled: false,
			evt:          similarityTestEvent(nostr.KindTextNote, "pubkey-1", "similar content"),
		},
		{
			name:         "bypass pubkey",
			cfg:          similarityTestConfigWith(func(cfg *config.ContentSimilarityConfig) { cfg.BypassPubkeys = []string{"pubkey-1"} }),
			cacheEnabled: true,
			evt:          similarityTestEvent(nostr.KindTextNote, "pubkey-1", "similar content"),
		},
		{
			name:         "outside kind",
			cfg:          similarityTestConfig(),
			cacheEnabled: true,
			evt:          similarityTestEvent(nostr.KindReaction, "pubkey-1", "similar content"),
		},
		{
			name:         "below content length",
			cfg:          similarityTestConfigWith(func(cfg *config.ContentSimilarityConfig) { cfg.MinContentLength = 20 }),
			cacheEnabled: true,
			evt:          similarityTestEvent(nostr.KindTextNote, "pubkey-1", "short"),
		},
		{
			name:         "cache lookup error",
			cfg:          similarityTestConfig(),
			cacheEnabled: true,
			evt:          similarityTestEvent(nostr.KindTextNote, "pubkey-1", "similar content"),
			finderError:  errors.New("cache unavailable"),
			finderCalls:  1,
		},
		{
			name:         "threshold is observed without rejection",
			cfg:          similarityTestConfig(),
			cacheEnabled: true,
			evt:          similarityTestEvent(nostr.KindTextNote, "pubkey-1", "similar content"),
			finderCalls:  1,
			addCalls:     1,
			wantFlagged:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finderCalls := 0
			addCalls := 0
			checker := newContentSimilarityChecker(tt.cfg)
			checker.isCacheEnabled = func() bool { return tt.cacheEnabled }
			checker.findCandidates = func(uint64, time.Duration) ([]uint64, error) {
				finderCalls++
				return nil, tt.finderError
			}
			checker.addPubkey = func(string, string, time.Duration) (int64, error) {
				addCalls++
				return 2, nil
			}

			reject, flagged, reason := checker.Check(context.Background(), tt.evt)
			if reject || reason != "" {
				t.Fatalf("Check() = reject %t, reason %q; observe mode must not reject", reject, reason)
			}
			if flagged != tt.wantFlagged {
				t.Fatalf("flagged = %t, want %t", flagged, tt.wantFlagged)
			}
			if finderCalls != tt.finderCalls {
				t.Fatalf("finder calls = %d, want %d", finderCalls, tt.finderCalls)
			}
			if addCalls != tt.addCalls {
				t.Fatalf("add calls = %d, want %d", addCalls, tt.addCalls)
			}
		})
	}
}

func TestContentSimilarityChecker_ObservesThresholdMetrics(t *testing.T) {
	checker := newContentSimilarityChecker(similarityTestConfig())
	checker.isCacheEnabled = func() bool { return true }
	checker.findCandidates = func(uint64, time.Duration) ([]uint64, error) { return []uint64{}, nil }
	checker.addPubkey = func(string, string, time.Duration) (int64, error) { return 2, nil }

	hitsBefore := testutil.ToFloat64(metrics.NostrContentSimilarityHitsTotal)
	thresholdBefore := testutil.ToFloat64(metrics.NostrContentSimilarityThresholdExceededTotal)
	reject, flagged, reason := checker.Check(
		context.Background(),
		similarityTestEvent(nostr.KindTextNote, "pubkey-1", "similar content"),
	)
	if reject || !flagged || reason != "" {
		t.Fatalf("Check() = reject %t, flagged %t, reason %q", reject, flagged, reason)
	}
	if after := testutil.ToFloat64(metrics.NostrContentSimilarityHitsTotal); after != hitsBefore+1 {
		t.Fatalf("similarity hit metric delta = %v, want 1", after-hitsBefore)
	}
	if after := testutil.ToFloat64(metrics.NostrContentSimilarityThresholdExceededTotal); after != thresholdBefore+1 {
		t.Fatalf("threshold metric delta = %v, want 1", after-thresholdBefore)
	}
}

func TestContentSimilarityChecker_AppliesConfiguredAction(t *testing.T) {
	for _, tt := range []struct {
		name        string
		mode        string
		wantReject  bool
		wantFlagged bool
	}{
		{name: "flag", mode: "flag", wantFlagged: true},
		{name: "reject", mode: "reject", wantReject: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := similarityTestConfig()
			cfg.Mode = tt.mode
			checker := newContentSimilarityChecker(cfg)
			checker.isCacheEnabled = func() bool { return true }
			checker.findCandidates = func(uint64, time.Duration) ([]uint64, error) { return []uint64{}, nil }
			checker.addPubkey = func(string, string, time.Duration) (int64, error) { return 2, nil }
			checker.getAction = func(string) (string, bool) { return "", false }
			checker.setAction = func(_ string, action string, _ time.Duration) error {
				if action != tt.mode {
					t.Fatalf("cached action = %q, want %q", action, tt.mode)
				}
				return nil
			}

			reject, flagged, reason := checker.Check(
				context.Background(),
				similarityTestEvent(nostr.KindTextNote, "pubkey-1", "similar content"),
			)
			if reject != tt.wantReject || flagged != tt.wantFlagged {
				t.Fatalf("Check() = reject %t, flagged %t", reject, flagged)
			}
			if tt.wantReject && reason != "restricted: similar content across multiple accounts" {
				t.Fatalf("reject reason = %q", reason)
			}
		})
	}
}

func TestContentSimilarityChecker_UsesCachedAction(t *testing.T) {
	checker := newContentSimilarityChecker(similarityTestConfig())
	checker.isCacheEnabled = func() bool { return true }
	checker.getAction = func(string) (string, bool) { return "reject", true }
	checker.findCandidates = func(uint64, time.Duration) ([]uint64, error) {
		t.Fatal("candidate lookup must not run for a cached action")
		return nil, nil
	}

	reject, flagged, reason := checker.Check(
		context.Background(),
		similarityTestEvent(nostr.KindTextNote, "pubkey-1", "similar content"),
	)
	if !reject || flagged || reason != "restricted: similar content across multiple accounts" {
		t.Fatalf("Check() = reject %t, flagged %t, reason %q", reject, flagged, reason)
	}
}

func TestContentSimilarityChecker_IgnoresDistantCandidate(t *testing.T) {
	content := "similar content"
	fingerprint := contentSimhash(content)
	cfg := similarityTestConfig()
	cfg.HammingThreshold = 0
	checker := newContentSimilarityChecker(cfg)
	checker.isCacheEnabled = func() bool { return true }
	checker.findCandidates = func(uint64, time.Duration) ([]uint64, error) {
		return []uint64{^fingerprint}, nil
	}
	addCalls := 0
	checker.addPubkey = func(string, string, time.Duration) (int64, error) {
		addCalls++
		return 1, nil
	}

	reject, flagged, reason := checker.Check(
		context.Background(),
		similarityTestEvent(nostr.KindTextNote, "pubkey-1", content),
	)
	if reject || flagged || reason != "" {
		t.Fatalf("Check() = reject %t, flagged %t, reason %q", reject, flagged, reason)
	}
	if addCalls != 1 {
		t.Fatalf("pubkey cache calls = %d, want 1 for the event fingerprint only", addCalls)
	}
}

func similarityTestConfig() config.ContentSimilarityConfig {
	return config.ContentSimilarityConfig{
		Enabled:          true,
		Mode:             "observe",
		WindowSeconds:    60,
		ThresholdPubkeys: 2,
		HammingThreshold: 64,
		MinContentLength: 1,
		Kinds:            []int{nostr.KindTextNote},
	}
}

func similarityTestConfigWith(change func(*config.ContentSimilarityConfig)) config.ContentSimilarityConfig {
	cfg := similarityTestConfig()
	change(&cfg)
	return cfg
}

func similarityTestEvent(kind int, pubkey string, content string) *nostr.Event {
	return &nostr.Event{Kind: kind, PubKey: pubkey, Content: content}
}
