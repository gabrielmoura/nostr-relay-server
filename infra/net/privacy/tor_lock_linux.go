//go:build linux

package privacy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"go.uber.org/zap"
)

func prepareTorDataDir(dataDir string, logger *zap.Logger) error {
	if dataDir == "" {
		return nil
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("create Tor data directory: %w", err)
	}
	lockPath := filepath.Join(dataDir, "lock")
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open Tor data-directory lock: %w", err)
	}
	defer lockFile.Close()
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return fmt.Errorf("Tor data directory is locked by a running process; operator action is required")
		}
		return fmt.Errorf("inspect Tor data-directory lock: %w", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	for _, pattern := range []string{"lock", "control-port-*", "torrc-*"} {
		matches, globErr := filepath.Glob(filepath.Join(dataDir, pattern))
		if globErr != nil {
			return fmt.Errorf("find stale Tor artifacts: %w", globErr)
		}
		for _, path := range matches {
			if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
				return fmt.Errorf("remove stale Tor artifact: %w", removeErr)
			}
			logger.Debug("removed stale Tor artifact", zap.String("path", path))
		}
	}
	return nil
}
