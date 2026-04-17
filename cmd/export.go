package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/cmd/internal/export"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export local events to file",
	Long: "Export events from local database into a file.\n\n" +
		"Supported formats:\n" +
		"- jsonl (default)\n" +
		"- tsv\n\n" +
		"Optional filter and segmentation are available for safer operational exports.",
	Example: "  nrserver export\n" +
		"  nrserver export --filter '{\"authors\":[\"<hex-pubkey>\"]}'\n" +
		"  nrserver export --filter-file ./filter.json\n" +
		"  nrserver export --limit 1000 --segment-size 500\n" +
		"  nrserver export --format tsv --file events.tsv\n" +
		"  nrserver export --format tsv --segment-size 500 --no-header --overwrite",
	Run: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringP("file", "f", defaultExportFilename("jsonl"), "Destination file path")
	exportCmd.Flags().String("format", "jsonl", "Export format: jsonl or tsv")
	exportCmd.Flags().String("filter", "", "Optional Nostr filter JSON object to select events")
	exportCmd.Flags().String("filter-file", "", "Optional path to JSON file containing export filter object")
	exportCmd.Flags().Int("limit", 0, "Maximum number of events to export (0 = no limit)")
	exportCmd.Flags().Int("segment-size", 0, "Events per output file segment (0 = disabled)")
	exportCmd.Flags().Bool("no-header", false, "Do not emit header row when format is tsv")
	exportCmd.Flags().Bool("overwrite", false, "Allow overwriting existing output files")
	exportCmd.Flags().IntP("batch-size", "b", 100, "Number of events fetched per database batch")
	exportCmd.Flags().IntP("writer-workers", "w", 2, "Number of encoder workers (reserved for compatibility)")
}

func runExport(cmd *cobra.Command, _ []string) {
	format, _ := cmd.Flags().GetString("format")
	file, _ := cmd.Flags().GetString("file")
	filter, _ := cmd.Flags().GetString("filter")
	filterFile, _ := cmd.Flags().GetString("filter-file")
	limit, _ := cmd.Flags().GetInt("limit")
	segmentSize, _ := cmd.Flags().GetInt("segment-size")
	noHeader, _ := cmd.Flags().GetBool("no-header")
	overwrite, _ := cmd.Flags().GetBool("overwrite")
	batchSize, _ := cmd.Flags().GetInt("batch-size")
	workers, _ := cmd.Flags().GetInt("writer-workers")

	if isDefaultExportFile(file) {
		file = defaultExportFilename(format)
	}

	options, err := export.BuildOptions(export.CLIOptions{
		Filename:      file,
		BatchSize:     batchSize,
		WriterWorkers: workers,
		Format:        format,
		Filter:        filter,
		FilterFile:    filterFile,
		Limit:         limit,
		SegmentSize:   segmentSize,
		NoHeader:      noHeader,
		Overwrite:     overwrite,
	})
	if err != nil {
		cobra.CheckErr(err)
	}

	cobra.CheckErr(export.Export(options))
}

func defaultExportFilename(format string) string {
	ext := "jsonl"
	if strings.EqualFold(strings.TrimSpace(format), "tsv") {
		ext = "tsv"
	}

	return fmt.Sprintf("export-%d.%s", time.Now().Unix(), ext)
}

func isDefaultExportFile(path string) bool {
	trimmed := strings.TrimSpace(path)
	return strings.HasPrefix(trimmed, "export-") && (strings.HasSuffix(trimmed, ".jsonl") || strings.HasSuffix(trimmed, ".tsv"))
}
