package croncmd

import (
	"fmt"
	"strings"
	"time"
)

type Options struct {
	List    bool
	RunOnce bool
	Jobs    []string
	Timeout time.Duration
}

func BuildOptions(raw Options) (*Options, error) {
	if raw.Timeout <= 0 {
		return nil, fmt.Errorf("invalid --timeout %s: must be greater than 0", raw.Timeout)
	}

	raw.Jobs = normalizeJobs(raw.Jobs)

	if raw.List && raw.RunOnce {
		return nil, fmt.Errorf("invalid flag combination: --list cannot be used with --run-once")
	}

	return &raw, nil
}

func normalizeJobs(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		for _, split := range strings.Split(value, ",") {
			clean := strings.ToLower(strings.TrimSpace(split))
			if clean == "" {
				continue
			}
			if clean == "nip40_expiration_cleanup" {
				clean = "nip40"
			}
			if _, ok := seen[clean]; ok {
				continue
			}
			seen[clean] = struct{}{}
			normalized = append(normalized, clean)
		}
	}

	return normalized
}
