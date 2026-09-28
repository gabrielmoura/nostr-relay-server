// Package blossommigrate moves legacy local Blossom blobs into S3 storage.
package blossommigrate

import (
	"fmt"
	"strings"
)

const (
	BackendLocal = "local"
	BackendS3    = "s3"
)

// Options controls one local-to-S3 migration run.
type Options struct {
	From    string
	To      string
	DryRun  bool
	Workers int
}

// BuildOptions validates CLI input and applies migration defaults.
func BuildOptions(raw Options) (Options, error) {
	options := raw
	options.From = strings.ToLower(strings.TrimSpace(options.From))
	options.To = strings.ToLower(strings.TrimSpace(options.To))

	if options.From != BackendLocal {
		return Options{}, fmt.Errorf("invalid --from %q: only local is supported", raw.From)
	}
	if options.To != BackendS3 {
		return Options{}, fmt.Errorf("invalid --to %q: only s3 is supported", raw.To)
	}
	if options.Workers <= 0 {
		return Options{}, fmt.Errorf("invalid --workers %d: must be greater than zero", options.Workers)
	}

	return options, nil
}
