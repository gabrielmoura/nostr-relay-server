package cmd

import (
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/cmd/internal/export"
	"github.com/spf13/cobra"
	"time"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export to json file",
	Long:  "Export all events from the database to a JSON file.",
	Run:   runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)
	filename := fmt.Sprintf("export-%d.jsonl", time.Now().Unix())
	exportCmd.Flags().StringP("file", "f", filename, "File to export events to")
	exportCmd.Flags().IntP("batch-size", "b", 100, "Number of events to export in each batch")
}
func runExport(cmd *cobra.Command, args []string) {
	filename := cmd.Flag("file").Value.String()

	size, err := cmd.Flags().GetInt("batch-size")
	if err != nil {
		fmt.Printf("Error getting batch size: %v\n", err)
		return
	}

	err = export.Export(&export.Options{
		Filename:  filename,
		BatchSize: size,
	})
	if err != nil {
		return
	}

}
