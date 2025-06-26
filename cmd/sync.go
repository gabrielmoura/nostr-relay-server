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
	pk := cmd.Flag("pk").Value.String()
	sync.Sync(&sync.ConfSync{
		Remote:    cmd.Flag("remote").Value.String(),
		Pk:        pk,
		Direction: cmd.Flag("direction").Value.String(),
	})
}

func init() {
	syncCmd.Flags().StringP("remote", "r", "", "Remote Nostr Server")
	syncCmd.Flags().StringP("pk", "p", "", "Public Key")
	syncCmd.Flags().StringP("direction", "d", "both", "Direction of the sync (up, down, both)")
	rootCmd.AddCommand(syncCmd)
}
