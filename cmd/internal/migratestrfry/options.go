package migratestrfry

import (
	"fmt"
	"strings"
)

const (
	SourceExport = "export"
	SourceLMDB   = "lmdb"
)

type CLIOptions struct {
	StrfryDB    string
	StrfryBin   string
	Source      string
	Since       int64
	BatchSize   int
	DryRun      bool
	FailOnError bool
}

func BuildOptions(raw CLIOptions) (CLIOptions, error) {
	options := raw
	options.StrfryDB = strings.TrimSpace(options.StrfryDB)
	options.StrfryBin = strings.TrimSpace(options.StrfryBin)
	options.Source = strings.ToLower(strings.TrimSpace(options.Source))

	if options.StrfryDB == "" {
		return CLIOptions{}, fmt.Errorf("invalid --strfry-db: cannot be empty")
	}
	if options.Source == "" {
		options.Source = SourceExport
	}
	if options.Source != SourceExport && options.Source != SourceLMDB {
		return CLIOptions{}, fmt.Errorf("invalid --source %q: must be export or lmdb", options.Source)
	}
	if options.Source == SourceExport && options.StrfryBin == "" {
		return CLIOptions{}, fmt.Errorf("invalid --strfry-bin: cannot be empty")
	}
	if options.Since < 0 {
		return CLIOptions{}, fmt.Errorf("invalid --since %d: must be greater than or equal to zero", options.Since)
	}
	if options.BatchSize <= 0 {
		return CLIOptions{}, fmt.Errorf("invalid --batch-size %d: must be greater than zero", options.BatchSize)
	}

	return options, nil
}
