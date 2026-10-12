package db

import "testing"

func TestNewRelaySyncJournalProviderRequiresEpoch(t *testing.T) {
	if _, err := NewRelaySyncJournalProvider(nil, 1); err == nil {
		t.Fatal("expected nil queries error")
	}
	if _, err := NewRelaySyncJournalProvider(nil, 0); err == nil {
		t.Fatal("expected invalid options error")
	}
}
