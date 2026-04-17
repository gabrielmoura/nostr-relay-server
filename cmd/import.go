package cmd

import (
	"time"

	importcmd "github.com/gabrielmoura/nostr-relay-server/cmd/internal/import"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import events from JSONL file",
	Long: "Import Nostr events from a JSONL file into the local database.\n\n" +
		"Set --batch-size 0 to process line-by-line mode.",
	Example: "  nrserver import --file events.jsonl\n" +
		"  nrserver import --file events.jsonl --batch-size 500 --num-workers 4\n" +
		"  nrserver import --batch-size 0 --num-workers 8 --fail-on-error",
	Run: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().StringP("file", "f", "events.jsonl", "JSONL file to import")
	importCmd.Flags().IntP("batch-size", "b", 100, "Batch size for import (0 = line-by-line mode)")
	importCmd.Flags().IntP("num-workers", "w", 2, "Number of parallel import workers")
	importCmd.Flags().Duration("stats-interval", 5*time.Second, "Interval for import progress logs (0 disables)")
	importCmd.Flags().Bool("fail-on-error", false, "Return non-zero status when row-level import errors occur")
}

func runImport(cmd *cobra.Command, _ []string) {
	filename, _ := cmd.Flags().GetString("file")
	batchSize, _ := cmd.Flags().GetInt("batch-size")
	numWorkers, _ := cmd.Flags().GetInt("num-workers")
	statsInterval, _ := cmd.Flags().GetDuration("stats-interval")
	failOnError, _ := cmd.Flags().GetBool("fail-on-error")

	cobra.CheckErr(importcmd.Run(&importcmd.CLIOptions{
		Filename:      filename,
		BatchSize:     batchSize,
		NumWorkers:    numWorkers,
		StatsInterval: statsInterval,
		FailOnError:   failOnError,
	}))
}
