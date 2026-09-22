//go:build !windows

package cmd

import (
	"errors"
	"fmt"
	"syscall"
)

func processRunning(pid int) (bool, error) {
	err := syscall.Kill(pid, 0)
	if err == nil || errors.Is(err, syscall.EPERM) {
		return true, nil
	}
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}
	return false, fmt.Errorf("signal PID %d: %w", pid, err)
}
