package event

import "testing"

func TestShouldAutoBanReportedPubkeyDefaultsToDisabled(t *testing.T) {
	if shouldAutoBanReportedPubkey(false, 5, 5) {
		t.Fatal("automatic report bans must be disabled unless explicitly enabled")
	}
}

func TestShouldAutoBanReportedPubkeyRequiresConfiguredThreshold(t *testing.T) {
	if shouldAutoBanReportedPubkey(true, 4, 0) {
		t.Fatal("automatic report ban must require a positive threshold")
	}
	if !shouldAutoBanReportedPubkey(true, 4, 5) {
		t.Fatal("automatic report ban should apply at the configured threshold")
	}
}
