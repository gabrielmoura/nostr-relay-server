package cmd

import (
	"github.com/gabrielmoura/nostr-relay-server/cmd/internal/down"
	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "download",
	Short: "Download events",
	Long: "Download events from one or more Nostr relays and save them to the local database.\n\n" +
		"Examples:\n" +
		"  nrserver download --relay-url wss://relay.damus.io --public-key <hex-or-npub>\n" +
		"  nrserver download --relay-url wss://relay.damus.io --mentioned --public-key <hex-or-npub>\n" +
		"  nrserver download --filter '{\"authors\":[\"<hex-pubkey>\"],\"since\":1700000000}'\n" +
		"  nrserver download --filter-file ./filter.json\n" +
		"  nrserver download --public-key <hex-or-npub> --filter '{\"kinds\":[1],\"search\":\"nostr\"}'",
	Run: runDownload,
}

func init() {
	rootCmd.AddCommand(downCmd)
	downCmd.Flags().StringP("public-key", "p", "", "Public key to filter events")
	downCmd.Flags().StringSliceP("relay-url", "r", []string{"wss://relay.damus.io"}, "Relay URL to connect to")
	downCmd.Flags().StringSliceP("tags", "t", []string{}, "Tags to filter events (e.g., tag1,tag2)")
	downCmd.Flags().BoolP("mentioned", "m", false, "Download events where the public key is mentioned")
	downCmd.Flags().IntSliceP("kinds", "k", []int{
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
	downCmd.Flags().IntP("timeout", "o", 30, "Timeout in seconds for the download operation")
	downCmd.Flags().String("filter", "", "Optional Nostr filter JSON object for additional constraints")
	downCmd.Flags().String("filter-file", "", "Optional path to a JSON file containing a Nostr filter object")
	downCmd.Flags().String("filter-merge", "override", "Filter merge strategy: override or strict-conflict")

}
func runDownload(cmd *cobra.Command, _ []string) {
	publicKey, _ := cmd.Flags().GetString("public-key")
	relays, _ := cmd.Flags().GetStringSlice("relay-url")
	tags, _ := cmd.Flags().GetStringSlice("tags")
	mentioned, _ := cmd.Flags().GetBool("mentioned")
	kinds, _ := cmd.Flags().GetIntSlice("kinds")
	timeout, _ := cmd.Flags().GetInt("timeout")
	filter, _ := cmd.Flags().GetString("filter")
	filterFile, _ := cmd.Flags().GetString("filter-file")
	filterMerge, _ := cmd.Flags().GetString("filter-merge")

	options, err := down.BuildOptions(down.CLIOptions{
		PublicKey:  publicKey,
		RelayURL:   relays,
		Mentioned:  mentioned,
		Kinds:      kinds,
		Tags:       tags,
		Timeout:    timeout,
		Filter:     filter,
		FilterFile: filterFile,
		Merge:      filterMerge,
	})
	if err != nil {
		cobra.CheckErr(err)
	}

	cobra.CheckErr(down.Download(options))
}
