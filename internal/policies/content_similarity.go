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
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/security"
	"github.com/nbd-wtf/go-nostr"
)

const simhashBands = 4

type contentSimilarityFinder func(uint64, time.Duration) ([]uint64, error)

type contentSimilarityChecker struct {
	cfg            config.ContentSimilarityConfig
	isCacheEnabled func() bool
	findCandidates contentSimilarityFinder
	addPubkey      contentPubkeyAdder
}

func newContentSimilarityChecker(cfg config.ContentSimilarityConfig) *contentSimilarityChecker {
	return &contentSimilarityChecker{
		cfg:            cfg,
		isCacheEnabled: cache.IsEnabled,
		findCandidates: cache.FindAndAddContentSimhash,
		addPubkey:      cache.AddContentPubkey,
	}
}

func (c *contentSimilarityChecker) Check(ctx context.Context, evt *nostr.Event) (bool, bool, string) {
	if !c.shouldCheck(ctx, evt) {
		return false, false, ""
	}
	fingerprint := contentSimhash(evt.Content)
	ttl := time.Duration(c.cfg.WindowSeconds) * time.Second
	candidates, err := c.findCandidates(fingerprint, ttl)
	if err != nil {
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
		if err == nil && count > maxCount {
			maxCount = count
		}
	}
	if maxCount < int64(c.cfg.ThresholdPubkeys) {
		return false, false, ""
	}
	metrics.NostrContentSimilarityThresholdExceededTotal.Inc()
	return false, true, ""
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
