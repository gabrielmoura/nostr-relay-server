package seed

import (
	"fmt"
	"time"
)

type CLIOptions struct {
	Bootstrap           bool
	BootstrapIdempotent bool
	SkipMigrate         bool
	DryRun              bool
	Timeout             time.Duration
}

func BuildOptions(raw CLIOptions) (*CLIOptions, error) {
	if raw.SkipMigrate && !raw.Bootstrap {
		return nil, fmt.Errorf("invalid seed options: --skip-migrate requires --bootstrap")
	}

	if raw.BootstrapIdempotent && !raw.Bootstrap {
		return nil, fmt.Errorf("invalid seed options: --bootstrap-idempotent requires --bootstrap")
	}

	if raw.Timeout <= 0 {
		return nil, fmt.Errorf("invalid --timeout %s: must be greater than 0", raw.Timeout)
	}

	return &raw, nil
}
