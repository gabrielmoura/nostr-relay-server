package cmd

import (
	_import "github.com/gabrielmoura/nostr-relay-server/cmd/internal/import"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import from a JSON file",
	Long:  "Imports Nostr events from a JSON file, one event per line.",
	Run:   runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().StringP("file", "f", "events.jsonl", "JSONL file to import")
	importCmd.Flags().IntP("batch-size", "b", 100, "Batch size for import")
	importCmd.Flags().IntP("num-workers", "w", 2, "Number of workers for parallel import")
}
func runImport(cmd *cobra.Command, _ []string) {
	filename := cmd.Flag("file").Value.String()
	batchSize, err := cmd.Flags().GetInt("batch-size")
	if err != nil {
		cmd.PrintErrf("Error getting batch size: %v\n", err)
		return
	}
	numWorkers, err := cmd.Flags().GetInt("num-workers")
	if err != nil {
		cmd.PrintErrf("Error getting number of workers: %v\n", err)
		return
	}
	err = _import.ParallelImport(filename, batchSize, numWorkers)
	if err != nil {
		return
	}
}
