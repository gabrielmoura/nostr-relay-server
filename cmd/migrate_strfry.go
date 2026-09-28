package cmd

import (
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/cmd/internal/migratestrfry"
	"github.com/spf13/cobra"
)

var migrateStrfryCmd = &cobra.Command{
	Use:   "migrate-strfry",
	Short: "Migrate events from a strfry database",
	Long: "Export events through strfry and persist them into the database configured by conf.yaml. " +
		"The target configuration is always loaded from the existing YAML configuration.",
	Args: cobra.NoArgs,
	RunE: runMigrateStrfry,
}

func init() {
	rootCmd.AddCommand(migrateStrfryCmd)
	migrateStrfryCmd.Flags().String("strfry-db", "", "Path to the source strfry database")
	migrateStrfryCmd.Flags().String("strfry-bin", "strfry", "Path to the strfry executable")
	migrateStrfryCmd.Flags().String("source", migratestrfry.SourceExport, "Source type: export or lmdb")
	migrateStrfryCmd.Flags().Int64("since", 0, "Only export events created at or after this Unix timestamp (0 = all events)")
	migrateStrfryCmd.Flags().Int("batch-size", 1000, "Number of events persisted per batch")
	migrateStrfryCmd.Flags().Bool("dry-run", false, "Validate exported events without writing to the target database")
	migrateStrfryCmd.Flags().Bool("fail-on-error", false, "Return an error when invalid or rejected rows are found")
}

func runMigrateStrfry(cmd *cobra.Command, _ []string) error {
	strfryDB, err := cmd.Flags().GetString("strfry-db")
	if err != nil {
		return err
	}
	strfryBin, err := cmd.Flags().GetString("strfry-bin")
	if err != nil {
		return err
	}
	source, err := cmd.Flags().GetString("source")
	if err != nil {
		return err
	}
	since, err := cmd.Flags().GetInt64("since")
	if err != nil {
		return err
	}
	batchSize, err := cmd.Flags().GetInt("batch-size")
	if err != nil {
		return err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}
	failOnError, err := cmd.Flags().GetBool("fail-on-error")
	if err != nil {
		return err
	}

	result, err := migratestrfry.Run(migratestrfry.CLIOptions{
		StrfryDB:    strfryDB,
		StrfryBin:   strfryBin,
		Source:      source,
		Since:       since,
		BatchSize:   batchSize,
		DryRun:      dryRun,
		FailOnError: failOnError,
	})
	_, outputErr := fmt.Fprintf(
		cmd.OutOrStdout(),
		"read=%d valid=%d persisted=%d invalid=%d rejected=%d\n",
		result.Read,
		result.Valid,
		result.Persisted,
		result.Invalid,
		result.Rejected,
	)
	if outputErr != nil {
		return outputErr
	}
	return err
}
