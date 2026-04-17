package export

import (
	"fmt"
	"os"
	"strings"
)

type OutputFormat string

const (
	FormatJSONL OutputFormat = "jsonl"
	FormatTSV   OutputFormat = "tsv"
)

type CLIOptions struct {
	Filename      string
	BatchSize     int
	WriterWorkers int
	Format        string
	Filter        string
	FilterFile    string
	Limit         int
	SegmentSize   int
	NoHeader      bool
	Overwrite     bool
}

type Options struct {
	Filename      string
	BatchSize     int
	WriterWorkers int
	Format        OutputFormat
	Filter        string
	FilterFile    string
	Limit         int
	SegmentSize   int
	NoHeader      bool
	Overwrite     bool
}

func BuildOptions(raw CLIOptions) (*Options, error) {
	if strings.TrimSpace(raw.Filename) == "" {
		return nil, fmt.Errorf("invalid --file: cannot be empty")
	}

	if raw.BatchSize <= 0 {
		return nil, fmt.Errorf("invalid --batch-size %d: must be greater than 0", raw.BatchSize)
	}

	if raw.WriterWorkers <= 0 {
		return nil, fmt.Errorf("invalid --writer-workers %d: must be greater than 0", raw.WriterWorkers)
	}

	if raw.Limit < 0 {
		return nil, fmt.Errorf("invalid --limit %d: must be >= 0", raw.Limit)
	}

	if raw.SegmentSize < 0 {
		return nil, fmt.Errorf("invalid --segment-size %d: must be >= 0", raw.SegmentSize)
	}

	if strings.TrimSpace(raw.Filter) != "" && strings.TrimSpace(raw.FilterFile) != "" {
		return nil, fmt.Errorf("invalid filter source: use only one of --filter or --filter-file")
	}

	if strings.TrimSpace(raw.FilterFile) != "" {
		if _, err := os.Stat(strings.TrimSpace(raw.FilterFile)); err != nil {
			return nil, fmt.Errorf("invalid --filter-file %q: %w", raw.FilterFile, err)
		}
	}

	format, err := parseFormat(raw.Format)
	if err != nil {
		return nil, err
	}

	return &Options{
		Filename:      strings.TrimSpace(raw.Filename),
		BatchSize:     raw.BatchSize,
		WriterWorkers: raw.WriterWorkers,
		Format:        format,
		Filter:        strings.TrimSpace(raw.Filter),
		FilterFile:    strings.TrimSpace(raw.FilterFile),
		Limit:         raw.Limit,
		SegmentSize:   raw.SegmentSize,
		NoHeader:      raw.NoHeader,
		Overwrite:     raw.Overwrite,
	}, nil
}

func parseFormat(raw string) (OutputFormat, error) {
	value := OutputFormat(strings.ToLower(strings.TrimSpace(raw)))
	switch value {
	case FormatJSONL, "":
		return FormatJSONL, nil
	case FormatTSV:
		return FormatTSV, nil
	default:
		return "", fmt.Errorf("invalid --format %q: expected jsonl or tsv", raw)
	}
}
