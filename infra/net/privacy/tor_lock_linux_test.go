//go:build linux

package privacy

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"go.uber.org/zap"
)

func TestPrepareTorDataDirRemovesOnlyStaleArtifacts(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"lock", "control-port-old", "torrc-old", "state"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	if err := prepareTorDataDir(dir, zap.NewNop()); err != nil {
		t.Fatalf("prepare Tor data dir: %v", err)
	}
	for _, name := range []string{"lock", "control-port-old", "torrc-old"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stale artifact %s still exists or stat failed: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "state")); err != nil {
		t.Fatalf("persistent state was removed: %v", err)
	}
}

func TestPrepareTorDataDirDoesNotRemoveLiveLock(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "lock")
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("open lock: %v", err)
	}
	t.Cleanup(func() { _ = lockFile.Close() })
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("lock file: %v", err)
	}
	t.Cleanup(func() { _ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN) })

	if err := prepareTorDataDir(dir, zap.NewNop()); err == nil {
		t.Fatal("prepare Tor data dir error = nil, want live-lock error")
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("live lock was removed: %v", err)
	}
}
