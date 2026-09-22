package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

const serverLockAttempts = 3

type serverAlreadyRunningError struct {
	PID  int
	Path string
}

func (e *serverAlreadyRunningError) Error() string {
	return fmt.Sprintf("nrserver instance with PID %d already owns %q", e.PID, e.Path)
}

type serverInstanceLock struct {
	path string
	pid  int
}

func acquireServerLock(path string, logger *zap.Logger) (*serverInstanceLock, error) {
	return acquireServerLockWithChecker(path, os.Getpid(), logger, processRunning)
}

func acquireServerLockWithChecker(
	path string,
	pid int,
	logger *zap.Logger,
	isProcessRunning func(int) (bool, error),
) (*serverInstanceLock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create nrserver lock directory: %w", err)
	}

	for range serverLockAttempts {
		lockFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			if writeErr := writeServerLock(lockFile, pid); writeErr != nil {
				_ = os.Remove(path)
				return nil, writeErr
			}

			lockPID, readErr := readServerLockPID(path)
			if readErr != nil {
				return nil, fmt.Errorf("confirm nrserver lock ownership: %w", readErr)
			}
			if lockPID != pid {
				return nil, fmt.Errorf("confirm nrserver lock ownership: found PID %d, want %d", lockPID, pid)
			}
			return &serverInstanceLock{path: path, pid: pid}, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, fmt.Errorf("create nrserver lock: %w", err)
		}

		lockPID, readErr := readServerLockPID(path)
		if readErr != nil {
			logger.Warn("nrserver lock inválido, removendo artefato stale",
				zap.String("lock", path),
				zap.Error(readErr))
			if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
				return nil, fmt.Errorf("remove invalid nrserver lock: %w", removeErr)
			}
			continue
		}

		logger.Debug("nrserver.lock encontrado, verificando processo dono",
			zap.Int("pid", lockPID),
			zap.String("lock", path))
		running, checkErr := isProcessRunning(lockPID)
		if checkErr != nil {
			return nil, fmt.Errorf("check nrserver lock owner PID %d: %w", lockPID, checkErr)
		}
		if running {
			return nil, &serverAlreadyRunningError{PID: lockPID, Path: path}
		}

		logger.Warn("processo dono do lock não está mais ativo, removendo lock stale",
			zap.Int("pid", lockPID),
			zap.String("lock", path))
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			return nil, fmt.Errorf("remove stale nrserver lock: %w", removeErr)
		}
	}

	return nil, fmt.Errorf("acquire nrserver lock after %d attempts", serverLockAttempts)
}

func writeServerLock(lockFile *os.File, pid int) error {
	if _, err := fmt.Fprintf(lockFile, "%d\n", pid); err != nil {
		_ = lockFile.Close()
		return fmt.Errorf("write nrserver lock: %w", err)
	}
	if err := lockFile.Close(); err != nil {
		return fmt.Errorf("close nrserver lock: %w", err)
	}
	return nil
}

func readServerLockPID(path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read nrserver lock: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("parse nrserver lock PID: %w", err)
	}
	return pid, nil
}

func (l *serverInstanceLock) Path() string {
	return l.path
}

func (l *serverInstanceLock) Release() error {
	lockPID, err := readServerLockPID(l.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if lockPID != l.pid {
		return fmt.Errorf("nrserver lock owner changed to PID %d", lockPID)
	}
	if err := os.Remove(l.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove nrserver lock: %w", err)
	}
	return nil
}
