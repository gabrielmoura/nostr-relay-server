package helper

import (
	"crypto/sha256"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/config"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

func FilterHash(cfg *config.RelayConfig, filter nostr.Filter, doCount bool) string {
	normalized := NormalizeFilter(cfg, filter)
	payload, _ := json.Marshal(struct {
		DoCount bool         `json:"do_count"`
		Filter  nostr.Filter `json:"filter"`
	}{
		DoCount: doCount,
		Filter:  normalized,
	})
	sum := sha256.Sum256(payload)
	return fmt.Sprintf("%x", sum[:])
}
