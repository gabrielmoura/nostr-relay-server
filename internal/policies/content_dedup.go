package policies

import (
	"context"
	"encoding/hex"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/security"
	"github.com/nbd-wtf/go-nostr"
	"github.com/zeebo/blake3"
	"go.uber.org/zap"
)

type contentPubkeyAdder func(hash string, pubkey string, ttl time.Duration) (int64, error)
type contentActionLookup func(hash string) (string, bool)
type contentActionSetter func(hash string, action string, ttl time.Duration) error

type contentDedupChecker struct {
	cfg            config.ContentDedupConfig
	addPubkey      contentPubkeyAdder
	isCacheEnabled func() bool
	getAction      contentActionLookup
	setAction      contentActionSetter
}

func newContentDedupChecker(cfg config.ContentDedupConfig, addPubkey contentPubkeyAdder) *contentDedupChecker {
	return &contentDedupChecker{
		cfg:            cfg,
		addPubkey:      addPubkey,
		isCacheEnabled: cache.IsEnabled,
		getAction:      cache.GetContentAction,
		setAction:      cache.SetContentAction,
	}
}

func (c *contentDedupChecker) Check(ctx context.Context, evt *nostr.Event) (bool, bool, string) {
	if !c.shouldCheck(ctx, evt) {
		return false, false, ""
	}

	hash := contentHash(normalizeContent(evt.Content))
	if action, found := c.getAction(hash); found {
		return contentDedupDecision(action)
	}

	count, err := c.addPubkey(
		hash,
		evt.PubKey,
		time.Duration(c.cfg.WindowSeconds)*time.Second,
	)
	if err != nil {
		if log.Logger != nil {
			log.Logger.Debug("content dedup cache update failed", zap.Error(err))
		}
		return false, false, ""
	}
	if count > 1 {
		metrics.NostrContentDedupHitsTotal.Inc()
	}
	if count < int64(c.cfg.ThresholdPubkeys) {
		return false, false, ""
	}

	metrics.NostrContentDedupThresholdExceededTotal.Inc()
	if log.Logger != nil {
		log.Logger.Info("content dedup threshold observed", zap.Int64("distinct_pubkeys", count))
	}
	if c.cfg.Mode == "observe" {
		return false, true, ""
	}
	if err := c.setAction(hash, c.cfg.Mode, time.Duration(c.cfg.WindowSeconds)*time.Second); err != nil && log.Logger != nil {
		log.Logger.Debug("content dedup action cache update failed", zap.Error(err))
	}
	return contentDedupDecision(c.cfg.Mode)
}

func contentDedupDecision(action string) (bool, bool, string) {
	switch action {
	case "reject":
		return true, false, security.Reason(security.PrefixRestricted, "duplicate content across multiple accounts")
	case "flag", "observe":
		return false, true, ""
	default:
		return false, false, ""
	}
}

func (c *contentDedupChecker) shouldCheck(ctx context.Context, evt *nostr.Event) bool {
	if !c.cfg.Enabled || evt == nil || !c.isCacheEnabled() {
		return false
	}
	if security.BypassFromContext(ctx).PublicationRestrictionsBypassed() || slices.Contains(c.cfg.BypassPubkeys, evt.PubKey) {
		return false
	}
	if !slices.Contains(c.cfg.Kinds, evt.Kind) {
		return false
	}
	return utf8.RuneCountInString(evt.Content) >= c.cfg.MinContentLength
}

func normalizeContent(content string) string {
	withoutZeroWidth := strings.Map(func(r rune) rune {
		switch r {
		case '\u200b', '\u200c', '\u200d', '\ufeff':
			return -1
		default:
			return r
		}
	}, content)
	return strings.ToLower(strings.Join(strings.Fields(withoutZeroWidth), " "))
}

func contentHash(normalizedContent string) string {
	hash := blake3.Sum256([]byte(normalizedContent))
	return hex.EncodeToString(hash[:])
}

func ContentDedupFlagged(evt *nostr.Event) bool {
	if evt == nil || config.Cfg == nil || config.Cfg.Security.Defense.ContentDedup.Mode != "flag" {
		return false
	}
	_, found := cache.GetContentAction(contentHash(normalizeContent(evt.Content)))
	return found
}
