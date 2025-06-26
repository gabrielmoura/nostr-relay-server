package cmd

import (
	"github.com/gabrielmoura/nostr-relay-server/cmd/internal/sync"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Syncs the local database with the remote database",
	Run:   runSync,
}

func runSync(cmd *cobra.Command, args []string) {
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

	pk := cmd.Flag("pk").Value.String()
	//if pk == "" {
	//	cmd.PrintErr("Public Key is required")
	//	return
	//}
	sync.Sync(&sync.ConfSync{
		Remote:     cmd.Flag("remote").Value.String(),
		Pk:         pk,
		Direction:  cmd.Flag("direction").Value.String(),
		BatchSize:  batchSize,
		NumWorkers: numWorkers,
	})
}

func init() {
	syncCmd.Flags().StringP("remote", "r", "", "Remote Nostr Server")
	syncCmd.Flags().StringP("pk", "p", "", "Public Key")
	syncCmd.Flags().StringP("direction", "d", "both", "Direction of the sync (up, down, both)")
	syncCmd.Flags().IntP("batch-size", "b", 100, "Batch size for processing")
	syncCmd.Flags().IntP("num-workers", "w", 2, "Number of workers for parallel processing")
	rootCmd.AddCommand(syncCmd)
}
