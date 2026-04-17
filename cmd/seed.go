package cmd

import (
	"time"

	seedcmd "github.com/gabrielmoura/nostr-relay-server/cmd/internal/seed"
	"github.com/spf13/cobra"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Prepare database schema and optional bootstrap events",
	Long: "Prepare the relay database for operation by running schema migration and, optionally,\n" +
		"creating bootstrap relay events.\n\n" +
		"This command requires a valid configuration and database connectivity.",
	Example: "  nrserver seed\n" +
		"  nrserver seed --bootstrap\n" +
		"  nrserver seed --bootstrap --bootstrap-idempotent\n" +
		"  nrserver seed --bootstrap --skip-migrate\n" +
		"  nrserver seed --dry-run",
	Run: runSeed,
}

func runSeed(cmd *cobra.Command, _ []string) {
	bootstrap, _ := cmd.Flags().GetBool("bootstrap")
	bootstrapIdempotent, _ := cmd.Flags().GetBool("bootstrap-idempotent")
	skipMigrate, _ := cmd.Flags().GetBool("skip-migrate")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	options, err := seedcmd.BuildOptions(seedcmd.CLIOptions{
		Bootstrap:           bootstrap,
		BootstrapIdempotent: bootstrapIdempotent,
		SkipMigrate:         skipMigrate,
		DryRun:              dryRun,
		Timeout:             timeout,
	})
	if err != nil {
		cobra.CheckErr(err)
	}

	cobra.CheckErr(seedcmd.Run(options))
}

func init() {
	seedCmd.Flags().Bool("bootstrap", false, "Create bootstrap relay events after migration")
	seedCmd.Flags().Bool("bootstrap-idempotent", false, "Skip bootstrap if marker events already exist (requires --bootstrap)")
	seedCmd.Flags().Bool("skip-migrate", false, "Skip schema migration (requires --bootstrap)")
	seedCmd.Flags().Bool("dry-run", false, "Print planned actions without writing to database")
	seedCmd.Flags().Duration("timeout", 30*time.Second, "Migration timeout (e.g. 30s, 2m)")
	rootCmd.AddCommand(seedCmd)
}
