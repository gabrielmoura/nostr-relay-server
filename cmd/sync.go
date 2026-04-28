package cmd

import (
	"fmt"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/internal/sync"
	"github.com/spf13/cobra"
)

var syncDirectionFlag string

var syncCmd = &cobra.Command{
	Use:   "sync <url>",
	Short: "Synchronize local database with a remote relay",
	Long: "Synchronize events with a remote relay using Negentropy.\n\n" +
		"Examples:\n" +
		"  nrserver sync wss://relay.example.com\n" +
		"  nrserver sync --remote wss://relay.example.com --dir down\n" +
		"  nrserver sync wss://relay.example.com --filter '{\"authors\":[\"<hex-pubkey>\"]}'\n" +
		"  nrserver sync wss://relay.example.com --timeout 30",
	Args: validateSyncArgs,
	Run:  runSync,
}

func runSync(cmd *cobra.Command, args []string) {
	remote, _ := cmd.Flags().GetString("remote")
	if len(args) > 0 && strings.TrimSpace(remote) == "" {
		remote = args[0]
	}

	dir := syncDirectionFlag
	pk, _ := cmd.Flags().GetString("pk")
	filter, _ := cmd.Flags().GetString("filter")
	timeout, _ := cmd.Flags().GetInt64("timeout")

	cfg, err := sync.BuildConfig(sync.CLIOptions{
		Remote:    remote,
		Direction: dir,
		Pk:        pk,
		Filter:    filter,
		Timeout:   timeout,
	})
	if err != nil {
		cobra.CheckErr(err)
	}

	sync.Sync(cfg)
}

func validateSyncArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("accepts at most one positional url argument")
	}

	remote, _ := cmd.Flags().GetString("remote")
	remote = strings.TrimSpace(remote)

	if len(args) == 0 && remote == "" {
		return fmt.Errorf("remote relay URL is required (use positional <url> or --remote)")
	}

	if len(args) == 1 && remote != "" && strings.TrimSpace(args[0]) != remote {
		return fmt.Errorf("conflicting remote URLs: positional %q and --remote %q", args[0], remote)
	}

	return nil
}

func init() {
	syncCmd.Flags().StringP("remote", "r", "", "Remote relay URL (ws:// or wss://)")
	syncCmd.Flags().StringP("pk", "p", "", "Author public key (hex or npub); applied as author constraint")
	syncCmd.Flags().StringVarP(&syncDirectionFlag, "dir", "d", "both", "Direction: both, down, up, none")
	syncCmd.Flags().StringVar(&syncDirectionFlag, "direction", "both", "Deprecated alias for --dir")
	_ = syncCmd.Flags().MarkDeprecated("direction", "use --dir")
	syncCmd.Flags().String("filter", "{}", "Nostr filter JSON (object or array of filters)")
	syncCmd.Flags().Int64("timeout", 0, "Abort sync if no activity for N seconds (0 disables timeout)")
	rootCmd.AddCommand(syncCmd)
}
