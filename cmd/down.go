package cmd

import (
	"github.com/gabrielmoura/nostr-relay-server/cmd/internal/down"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "download",
	Short: "Download events",
	Long:  "Download events from a Nostr relay and save them to database.",
	Run:   runDownload,
}
var (
	relays *[]string
)

func init() {
	rootCmd.AddCommand(downCmd)
	downCmd.Flags().StringP("public-key", "p", "", "Public key to filter events")
	relays = downCmd.Flags().StringSliceP("relay-url", "r", []string{"wss://relay.damus.io"}, "Relay URL to connect to")

}
func runDownload(cmd *cobra.Command, args []string) {
	down.Download(&down.DownloadOptions{
		PublicKey: cmd.Flag("public-key").Value.String(),
		RelayURL:  *relays,
	})
}
