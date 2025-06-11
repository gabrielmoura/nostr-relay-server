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
}
func runImport(cmd *cobra.Command, _ []string) {
	filename := cmd.Flag("file").Value.String()
	err := _import.Import(filename)
	if err != nil {
		return
	}
}
