package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestAcquireServerLockRejectsLiveOwner(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "nrserver.lock")
	logger := zap.NewNop()

	first, err := acquireServerLockWithChecker(lockPath, 12345, logger, func(pid int) (bool, error) {
		return pid == 12345, nil
	})
	if err != nil {
		t.Fatalf("acquire first lock: %v", err)
	}
	t.Cleanup(func() {
		if err := first.Release(); err != nil {
			t.Errorf("release first lock: %v", err)
		}
	})

	_, err = acquireServerLockWithChecker(lockPath, 67890, logger, func(pid int) (bool, error) {
		return pid == 12345, nil
	})
	var runningErr *serverAlreadyRunningError
	if !errors.As(err, &runningErr) {
		t.Fatalf("acquire second lock error = %v, want serverAlreadyRunningError", err)
	}
	if runningErr.PID != 12345 {
		t.Fatalf("running PID = %d, want 12345", runningErr.PID)
	}
}

func TestAcquireServerLockRemovesStaleOwner(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "nrserver.lock")
	if err := os.WriteFile(lockPath, []byte("12345\n"), 0o600); err != nil {
		t.Fatalf("write stale lock: %v", err)
	}

	lock, err := acquireServerLockWithChecker(lockPath, 67890, zap.NewNop(), func(int) (bool, error) {
		return false, nil
	})
	if err != nil {
		t.Fatalf("acquire after stale lock: %v", err)
	}
	t.Cleanup(func() {
		if err := lock.Release(); err != nil {
			t.Errorf("release replacement lock: %v", err)
		}
	})

	pid, err := readServerLockPID(lockPath)
	if err != nil {
		t.Fatalf("read replacement lock: %v", err)
	}
	if pid != 67890 {
		t.Fatalf("replacement PID = %d, want 67890", pid)
	}
}

func TestServerInstanceLockReleaseDoesNotRemoveNewOwner(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "nrserver.lock")
	if err := os.WriteFile(lockPath, []byte("67890\n"), 0o600); err != nil {
		t.Fatalf("write replacement lock: %v", err)
	}

	lock := &serverInstanceLock{path: lockPath, pid: 12345}
	if err := lock.Release(); err == nil {
		t.Fatal("release error = nil, want owner-change error")
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("replacement lock removed: %v", err)
	}
}
