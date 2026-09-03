package privacy

import "testing"

// TestLoadOrCreateStringFresh verifies SYNC-1 regression: a fresh (missing)
// entry must NOT be fatal — it returns ("", nil) so callers fall back to a
// transient value and persist after first connect.
func TestLoadOrCreateStringFresh(t *testing.T) {
	store := NewKeyStore(t.TempDir())
	v, err := loadOrCreateString(store, "i2p.key")
	if err != nil {
		t.Fatalf("fresh loadOrCreateString should not error, got %v", err)
	}
	if v != "" {
		t.Fatalf("fresh loadOrCreateString should return empty, got %q", v)
	}

	// After persisting, it must be reusable.
	if err := store.Save("i2p.key", []byte("abc123")); err != nil {
		t.Fatalf("save: %v", err)
	}
	v2, err := loadOrCreateString(store, "i2p.key")
	if err != nil {
		t.Fatalf("reuse loadOrCreateString errored: %v", err)
	}
	if v2 != "abc123" {
		t.Fatalf("expected persisted value, got %q", v2)
	}
}
