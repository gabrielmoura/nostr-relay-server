//go:build !linux

package privacy

import (
	"fmt"

	"go.uber.org/zap"
)

func prepareTorDataDir(dataDir string, _ *zap.Logger) error {
	if dataDir == "" {
		return nil
	}
	return fmt.Errorf("cannot safely inspect Tor data-directory lock on this platform")
}
