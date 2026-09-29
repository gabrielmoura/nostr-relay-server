package embed

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestNostrPNGSHA256MatchesEmbeddedAsset(t *testing.T) {
	sum := sha256.Sum256(NostrPNG)
	if got := hex.EncodeToString(sum[:]); got != NostrPNGSHA256 {
		t.Fatalf("NostrPNGSHA256 = %q, want %q", NostrPNGSHA256, got)
	}
}
