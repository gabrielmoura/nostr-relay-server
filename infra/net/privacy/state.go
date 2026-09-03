package privacy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// KeyStore persists a per-network identity/private-key material as a raw byte
// blob on disk, ensuring the same cryptographic identity (and therefore the
// same network address: .onion / IPv6 / .b32.i2p) is reused across restarts.
//
// Security properties:
//   - the state directory is created with 0700 permissions;
//   - key files are written with 0600 permissions;
//   - writes are atomic (temp file + rename) so a crash never corrupts an
//     existing identity.
type KeyStore struct {
	dir string
}

// NewKeyStore builds a KeyStore rooted at the given directory. The directory is
// created (with 0700) when first used.
func NewKeyStore(dir string) *KeyStore {
	return &KeyStore{dir: dir}
}

// ErrNotFound is returned by Load when no key has been persisted yet.
var ErrNotFound = errors.New("state: key not found")

// Load reads the persisted key material for a named entry. It returns
// ErrNotFound when the entry does not exist yet. Any read/stat failure is
// wrapped with the original cause for traceability.
func (s *KeyStore) Load(name string) ([]byte, error) {
	path := filepath.Join(s.dir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("loading %s: %w: %w", name, ErrNotFound, err)
		}
		return nil, fmt.Errorf("loading %s from %q: %w", name, path, err)
	}
	return data, nil
}

// LoadOrCreate returns the persisted key material for `name`, or, when absent,
// generates a fresh one via `generate` and persists it atomically. This is the
// workhorse used by every privacy service on startup: reuse if present, else
// create-and-remember.
func (s *KeyStore) LoadOrCreate(name string, generate func() ([]byte, error)) ([]byte, error) {
	if data, err := s.Load(name); err == nil {
		return data, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	data, err := generate()
	if err != nil {
		return nil, fmt.Errorf("generating %s: %w", name, err)
	}
	if err := s.Save(name, data); err != nil {
		return nil, err
	}
	return data, nil
}

// Save writes the key material for a named entry atomically (temp file +
// rename) with 0600 permissions, creating the directory (0700) if needed.
func (s *KeyStore) Save(name string, data []byte) error {
	if s.dir == "" {
		return errors.New("state: empty state directory")
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("creating state dir %q: %w", s.dir, err)
	}
	path := filepath.Join(s.dir, name)
	tmp, err := os.CreateTemp(s.dir, "."+name+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file for %q in %q: %w", name, s.dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing key material for %q: %w", name, err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod key file for %q: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing key file for %q: %w", name, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("finalizing key file %q: %w", path, err)
	}
	return nil
}

// Dir returns the state directory root (for logging/inspection).
func (s *KeyStore) Dir() string { return s.dir }

// loadOrCreateString is a convenience wrapper for string key material (e.g. a
// SAM destination blob). Unlike LoadOrCreate it does NOT generate on the spot:
// a missing entry means "fresh run, nothing persisted yet" and returns ("", nil)
// so the caller can fall back to a transient/on-demand value and persist later.
func loadOrCreateString(store *KeyStore, name string) (string, error) {
	raw, err := store.Load(name)
	if err == nil {
		return string(raw), nil
	}
	if errors.Is(err, ErrNotFound) {
		return "", nil // fresh: no persisted value yet
	}
	return "", err
}
