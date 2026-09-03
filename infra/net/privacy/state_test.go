package privacy

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestKeyStoreLoadOrCreateAndReuse(t *testing.T) {
	dir := t.TempDir()
	store := NewKeyStore(dir)

	// First call generates + persists.
	var genCalls int
	key1, err := store.LoadOrCreate("test.key", func() ([]byte, error) {
		genCalls++
		return []byte("first-key"), nil
	})
	if err != nil {
		t.Fatalf("first LoadOrCreate: %v", err)
	}
	if string(key1) != "first-key" {
		t.Fatalf("expected generated key, got %q", key1)
	}
	if genCalls != 1 {
		t.Fatalf("generator should run once, ran %d times", genCalls)
	}

	// Second call must reuse the persisted value, NOT regenerate.
	key2, err := store.LoadOrCreate("test.key", func() ([]byte, error) {
		genCalls++
		return []byte("new-key"), nil
	})
	if err != nil {
		t.Fatalf("second LoadOrCreate: %v", err)
	}
	if string(key2) != "first-key" {
		t.Fatalf("expected persisted key reused, got %q", key2)
	}
	if genCalls != 1 {
		t.Fatalf("generator should not run on reuse, ran %d times", genCalls)
	}

	// The persisted file exists.
	if _, err := os.Stat(filepath.Join(dir, "test.key")); err != nil {
		t.Fatalf("persisted file missing: %v", err)
	}
}

func TestKeyStorePermissions(t *testing.T) {
	dir := t.TempDir()
	store := NewKeyStore(filepath.Join(dir, "nested", "state"))

	if err := store.Save("k", []byte("secret")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	fi, err := os.Stat(filepath.Join(dir, "nested", "state", "k"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("key file perms = %v, want 0600", fi.Mode().Perm())
	}
	di, err := os.Stat(filepath.Join(dir, "nested", "state"))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if di.Mode().Perm() != 0o700 {
		t.Errorf("state dir perms = %v, want 0700", di.Mode().Perm())
	}
}

func TestKeyStoreLoadNotFound(t *testing.T) {
	store := NewKeyStore(t.TempDir())
	_, err := store.Load("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLoadOrCreateStringContract(t *testing.T) {
	store := NewKeyStore(t.TempDir())

	// Fresh store: nothing persisted yet -> ("", nil) so the caller falls back to
	// a transient/on-demand value and persists later. NOT an error.
	got, err := loadOrCreateString(store, "i2p.key")
	if err != nil {
		t.Fatalf("fresh store: expected no error, got %v", err)
	}
	if got != "" {
		t.Fatalf("fresh store: expected empty string, got %q", got)
	}

	// Persist a value, then it must be readable back.
	if err := store.Save("i2p.key", []byte("dest-blob")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got2, err := loadOrCreateString(store, "i2p.key")
	if err != nil {
		t.Fatalf("after save: unexpected error %v", err)
	}
	if got2 != "dest-blob" {
		t.Fatalf("expected persisted value, got %q", got2)
	}
}
