package cmd

import (
	"github.com/gabrielmoura/nostr-relay-server/cmd/internal/down"
	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "download",
	Short: "Download events",
	Long:  "Download events from a Nostr relay and save them to database.",
	Run:   runDownload,
}
var (
	relays    []string
	mentioned bool
	kinds     []int
	publicKey string
)

func init() {
	rootCmd.AddCommand(downCmd)
	downCmd.Flags().StringVarP(&publicKey, "public-key", "p", "", "Public key to filter events")
	downCmd.Flags().StringSliceVarP(&relays, "relay-url", "r", []string{"wss://relay.damus.io"}, "Relay URL to connect to")
	downCmd.Flags().BoolVarP(&mentioned, "mentioned", "m", false, "Download events where the public key is mentioned")
	downCmd.Flags().IntSliceVarP(&kinds, "kinds", "k", []int{
		nostr.KindTextNote,
		nostr.KindArticle,
		nostr.KindRepost,
		nostr.KindBookmarkSets,
		nostr.KindMuteSets,
		nostr.KindProfileBadges,
		nostr.KindBadgeDefinition,
		nostr.KindTorrent,
		nostr.KindTorrentComment,
		nostr.KindFileMetadata,
		nostr.KindChannelMessage,
		nostr.KindChannelMetadata,
		nostr.KindChannelCreation,
		nostr.KindProfileMetadata,
		nostr.KindReporting,
		nostr.KindDirectMessage,
	}, "Kinds of events to download")

}
func runDownload(cmd *cobra.Command, args []string) {
	down.Download(&down.DownloadOptions{
		PublicKey: publicKey,
		RelayURL:  relays,
		Mentioned: mentioned,
		Kinds:     kinds,
	})
}
