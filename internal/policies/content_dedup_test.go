package policies

import (
	"context"
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/nbd-wtf/go-nostr"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNormalizeContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "normalizes whitespace case and zero width characters",
			content: "  HELLO​\t  World\n",
			want:    "hello world",
		},
		{
			name:    "preserves non whitespace characters",
			content: "Nostr, café!",
			want:    "nostr, café!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeContent(tt.content); got != tt.want {
				t.Fatalf("normalizeContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContentHashStable(t *testing.T) {
	content := normalizeContent("Hello World")
	if first, second := contentHash(content), contentHash(content); first != second {
		t.Fatalf("contentHash() = %q then %q, want stable hash", first, second)
	}
	if contentHash(content) == contentHash(normalizeContent("Hello Nostr")) {
		t.Fatal("contentHash() collision for distinct normalized content")
	}
}

func TestContentDedupChecker(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.ContentDedupConfig
		evt          *nostr.Event
		cacheEnabled bool
		calls        int
		wantFlagged  bool
	}{
		{
			name: "counts distinct pubkeys and observes threshold",
			cfg:  contentDedupTestConfig(),
			evt: &nostr.Event{
				Kind:    nostr.KindTextNote,
				Content: "same content",
				PubKey:  "pubkey-1",
			},
			cacheEnabled: true,
			calls:        1,
		},
		{
			name: "does nothing when cache is disabled",
			cfg:  contentDedupTestConfig(),
			evt: &nostr.Event{
				Kind:    nostr.KindTextNote,
				Content: "same content",
				PubKey:  "pubkey-1",
			},
			cacheEnabled: false,
		},
		{
			name: "does nothing when disabled",
			cfg: func() config.ContentDedupConfig {
				cfg := contentDedupTestConfig()
				cfg.Enabled = false
				return cfg
			}(),
			evt: &nostr.Event{
				Kind:    nostr.KindTextNote,
				Content: "same content",
				PubKey:  "pubkey-1",
			},
			cacheEnabled: true,
		},
		{
			name: "does nothing for configured bypass pubkey",
			cfg: func() config.ContentDedupConfig {
				cfg := contentDedupTestConfig()
				cfg.BypassPubkeys = []string{"pubkey-1"}
				return cfg
			}(),
			evt: &nostr.Event{
				Kind:    nostr.KindTextNote,
				Content: "same content",
				PubKey:  "pubkey-1",
			},
			cacheEnabled: true,
		},
		{
			name: "does nothing outside the configured kind",
			cfg:  contentDedupTestConfig(),
			evt: &nostr.Event{
				Kind:    nostr.KindReaction,
				Content: "same content",
				PubKey:  "pubkey-1",
			},
			cacheEnabled: true,
		},
		{
			name: "does nothing below the configured content length",
			cfg: func() config.ContentDedupConfig {
				cfg := contentDedupTestConfig()
				cfg.MinContentLength = 20
				return cfg
			}(),
			evt: &nostr.Event{
				Kind:    nostr.KindTextNote,
				Content: "short",
				PubKey:  "pubkey-1",
			},
			cacheEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			checker := newContentDedupChecker(tt.cfg, func(_ string, _ string, _ time.Duration) (int64, error) {
				calls++
				return 1, nil
			})
			checker.isCacheEnabled = func() bool { return tt.cacheEnabled }

			_, flagged, _ := checker.Check(context.Background(), tt.evt)
			if calls != tt.calls {
				t.Fatalf("cache calls = %d, want %d", calls, tt.calls)
			}
			if flagged != tt.wantFlagged {
				t.Fatalf("flagged = %t, want %t", flagged, tt.wantFlagged)
			}
		})
	}
}

func TestContentDedupChecker_ObservesThresholdWithoutRejecting(t *testing.T) {
	counts := map[string]map[string]struct{}{}
	checker := newContentDedupChecker(contentDedupTestConfig(), func(hash string, pubkey string, _ time.Duration) (int64, error) {
		pubkeys := counts[hash]
		if pubkeys == nil {
			pubkeys = map[string]struct{}{}
			counts[hash] = pubkeys
		}
		pubkeys[pubkey] = struct{}{}
		return int64(len(pubkeys)), nil
	})
	checker.isCacheEnabled = func() bool { return true }

	hitsBefore := testutil.ToFloat64(metrics.NostrContentDedupHitsTotal)
	thresholdBefore := testutil.ToFloat64(metrics.NostrContentDedupThresholdExceededTotal)
	for _, pubkey := range []string{"pubkey-1", "pubkey-1", "pubkey-2"} {
		reject, flagged, reason := checker.Check(context.Background(), &nostr.Event{
			Kind:    nostr.KindTextNote,
			Content: "  SAME​ content ",
			PubKey:  pubkey,
		})
		if reject || reason != "" {
			t.Fatalf("Check() = reject %t, reason %q; observe mode must not reject", reject, reason)
		}
		if pubkey == "pubkey-2" && !flagged {
			t.Fatal("Check() flagged = false after second distinct pubkey, want true")
		}
	}
	if got := len(counts); got != 1 {
		t.Fatalf("normalized content hashes = %d, want 1", got)
	}
	if after := testutil.ToFloat64(metrics.NostrContentDedupHitsTotal); after != hitsBefore+1 {
		t.Fatalf("dedup hit metric delta = %v, want 1", after-hitsBefore)
	}
	if after := testutil.ToFloat64(metrics.NostrContentDedupThresholdExceededTotal); after != thresholdBefore+1 {
		t.Fatalf("threshold metric delta = %v, want 1", after-thresholdBefore)
	}
}

func TestContentDedupChecker_AppliesConfiguredAction(t *testing.T) {
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
			cfg := contentDedupTestConfig()
			cfg.Mode = tt.mode
			checker := newContentDedupChecker(cfg, func(string, string, time.Duration) (int64, error) { return 2, nil })
			checker.isCacheEnabled = func() bool { return true }
			checker.getAction = func(string) (string, bool) { return "", false }
			checker.setAction = func(_ string, action string, _ time.Duration) error {
				if action != tt.mode {
					t.Fatalf("cached action = %q, want %q", action, tt.mode)
				}
				return nil
			}

			reject, flagged, reason := checker.Check(context.Background(), &nostr.Event{Kind: nostr.KindTextNote, Content: "content", PubKey: "pubkey"})
			if reject != tt.wantReject || flagged != tt.wantFlagged {
				t.Fatalf("Check() = reject %t, flagged %t", reject, flagged)
			}
			if tt.wantReject && reason != "restricted: duplicate content across multiple accounts" {
				t.Fatalf("reject reason = %q", reason)
			}
		})
	}
}

func contentDedupTestConfig() config.ContentDedupConfig {
	return config.ContentDedupConfig{
		Enabled:          true,
		Mode:             "observe",
		WindowSeconds:    60,
		ThresholdPubkeys: 2,
		MinContentLength: 1,
		Kinds:            []int{nostr.KindTextNote},
	}
}
