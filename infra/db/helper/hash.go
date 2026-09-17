package helper

import (
	"crypto/sha256"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/config"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

func FilterHash(cfg *config.RelayConfig, filter nostr.Filter, doCount bool) string {
	return NormalizedFilterHash(NormalizeFilter(cfg, filter), doCount)
}

func NormalizedFilterHash(filter nostr.Filter, doCount bool) string {
	payload, _ := json.Marshal(struct {
		DoCount bool         `json:"do_count"`
		Filter  nostr.Filter `json:"filter"`
	}{
		DoCount: doCount,
		Filter:  filter,
	})
	sum := sha256.Sum256(payload)
	return fmt.Sprintf("%x", sum[:])
}
