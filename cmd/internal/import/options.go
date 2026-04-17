package _import

import (
	"fmt"
	"strings"
	"time"
)

type CLIOptions struct {
	Filename      string
	BatchSize     int
	NumWorkers    int
	StatsInterval time.Duration
	FailOnError   bool
}

func BuildOptions(raw CLIOptions) (*CLIOptions, error) {
	if strings.TrimSpace(raw.Filename) == "" {
		return nil, fmt.Errorf("invalid --file: cannot be empty")
	}

	if raw.BatchSize < 0 {
		return nil, fmt.Errorf("invalid --batch-size %d: must be >= 0", raw.BatchSize)
	}

	if raw.NumWorkers <= 0 {
		return nil, fmt.Errorf("invalid --num-workers %d: must be greater than 0", raw.NumWorkers)
	}

	if raw.StatsInterval < 0 {
		return nil, fmt.Errorf("invalid --stats-interval %s: must be >= 0", raw.StatsInterval)
	}

	return &raw, nil
}
