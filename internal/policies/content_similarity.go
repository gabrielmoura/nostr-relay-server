package policies

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/bits"
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
	"go.uber.org/zap"
)

const simhashBands = 4

type contentSimilarityFinder func(uint64, time.Duration) ([]uint64, error)

type contentSimilarityChecker struct {
	cfg            config.ContentSimilarityConfig
	isCacheEnabled func() bool
	findCandidates contentSimilarityFinder
	addPubkey      contentPubkeyAdder
	getAction      contentActionLookup
	setAction      contentActionSetter
}

func newContentSimilarityChecker(cfg config.ContentSimilarityConfig) *contentSimilarityChecker {
	return &contentSimilarityChecker{
		cfg:            cfg,
		isCacheEnabled: cache.IsEnabled,
		findCandidates: cache.FindAndAddContentSimhash,
		addPubkey:      cache.AddContentPubkey,
		getAction:      cache.GetContentAction,
		setAction:      cache.SetContentAction,
	}
}

func (c *contentSimilarityChecker) Check(ctx context.Context, evt *nostr.Event) (bool, bool, string) {
	if !c.shouldCheck(ctx, evt) {
		return false, false, ""
	}
	fingerprint := contentSimhash(evt.Content)
	ttl := time.Duration(c.cfg.WindowSeconds) * time.Second
	actionKey := contentSimilarityActionKey(fingerprint)
	if action, found := c.getAction(actionKey); found {
		return contentSimilarityDecision(action)
	}
	candidates, err := c.findCandidates(fingerprint, ttl)
	if err != nil {
		if log.Logger != nil {
			log.Logger.Debug("content similarity cache lookup failed", zap.Error(err))
		}
		return false, false, ""
	}
	candidates = append(candidates, fingerprint)
	processed := make(map[uint64]struct{}, len(candidates))
	var maxCount int64
	for _, candidate := range candidates {
		if _, found := processed[candidate]; found {
			continue
		}
		processed[candidate] = struct{}{}
		if simhashDistance(fingerprint, candidate) > c.cfg.HammingThreshold {
			continue
		}
		count, err := c.addPubkey(fmt.Sprintf("%016x", candidate), evt.PubKey, ttl)
		if err != nil && log.Logger != nil {
			log.Logger.Debug("content similarity cache update failed", zap.Error(err))
		}
		if err == nil && count > maxCount {
			maxCount = count
		}
	}
	if maxCount > 1 {
		metrics.NostrContentSimilarityHitsTotal.Inc()
	}
	if maxCount < int64(c.cfg.ThresholdPubkeys) {
		return false, false, ""
	}
	metrics.NostrContentSimilarityThresholdExceededTotal.Inc()
	if log.Logger != nil {
		log.Logger.Info("content similarity threshold observed", zap.Int64("distinct_pubkeys", maxCount))
	}
	if c.cfg.Mode == "observe" {
		return false, true, ""
	}
	if err := c.setAction(actionKey, c.cfg.Mode, ttl); err != nil && log.Logger != nil {
		log.Logger.Debug("content similarity action cache update failed", zap.Error(err))
	}
	return contentSimilarityDecision(c.cfg.Mode)
}

func (c *contentSimilarityChecker) shouldCheck(ctx context.Context, evt *nostr.Event) bool {
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

func contentSimhash(content string) uint64 {
	weights := [64]int{}
	words := strings.Fields(normalizeContent(content))
	for index := range words {
		shingle := words[index]
		if index+1 < len(words) {
			shingle += " " + words[index+1]
		}
		hasher := fnv.New64a()
		_, _ = hasher.Write([]byte(shingle))
		value := hasher.Sum64()
		for bit := range 64 {
			if value&(uint64(1)<<bit) != 0 {
				weights[bit]++
			} else {
				weights[bit]--
			}
		}
	}

	var fingerprint uint64
	for bit, weight := range weights {
		if weight > 0 {
			fingerprint |= uint64(1) << bit
		}
	}
	return fingerprint
}

func simhashDistance(left uint64, right uint64) int {
	return bits.OnesCount64(left ^ right)
}

func simhashBandValue(fingerprint uint64, band int) uint16 {
	return uint16(fingerprint >> (band * 16))
}

func contentSimilarityActionKey(fingerprint uint64) string {
	return fmt.Sprintf("similarity:%016x", fingerprint)
}

func contentSimilarityDecision(action string) (bool, bool, string) {
	switch action {
	case "reject":
		return true, false, security.Reason(security.PrefixRestricted, "similar content across multiple accounts")
	case "flag", "observe":
		return false, true, ""
	default:
		return false, false, ""
	}
}

func ContentSimilarityFlagged(evt *nostr.Event) bool {
	if evt == nil || config.Cfg == nil || config.Cfg.Security.Defense.ContentSimilarity.Mode != "flag" {
		return false
	}
	_, found := cache.GetContentAction(contentSimilarityActionKey(contentSimhash(evt.Content)))
	return found
}
