//go:build !diagnostics

package cmd

import (
	"io"

	"go.uber.org/zap"
)

// startDevelopmentPprof is intentionally absent from standard builds. The
// diagnostics build tag is required in addition to app_env: development.
func startDevelopmentPprof(_ string, _ *zap.Logger) io.Closer {
	return nil
}

func closeDevelopmentPprof(_ io.Closer) {}
