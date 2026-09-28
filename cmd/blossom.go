package cmd

import (
	"context"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/cmd/internal/blossommigrate"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/internal/blobstore"
	"github.com/spf13/cobra"
)

var blossomCmd = &cobra.Command{
	Use:   "blossom",
	Short: "Manage Blossom blob storage",
}

var blossomMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Copy legacy local Blossom blobs to S3 without deleting local files",
	Long: "Copies blobs from the local files directory to configured S3 storage. " +
		"Existing S3 objects with the same size are skipped; size conflicts fail safely.",
	Args: cobra.NoArgs,
	RunE: runBlossomMigrate,
}

func init() {
	blossomMigrateCmd.Flags().String("from", blossommigrate.BackendLocal, "Source backend (local)")
	blossomMigrateCmd.Flags().String("to", blossommigrate.BackendS3, "Target backend (s3)")
	blossomMigrateCmd.Flags().Bool("dry-run", false, "Report blobs that would be copied without writing to S3")
	blossomMigrateCmd.Flags().Int("workers", 4, "Number of concurrent copy workers")
	blossomCmd.AddCommand(blossomMigrateCmd)
	rootCmd.AddCommand(blossomCmd)
}

func runBlossomMigrate(cmd *cobra.Command, _ []string) error {
	from, err := cmd.Flags().GetString("from")
	if err != nil {
		return err
	}
	to, err := cmd.Flags().GetString("to")
	if err != nil {
		return err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}
	workers, err := cmd.Flags().GetInt("workers")
	if err != nil {
		return err
	}
	options, err := blossommigrate.BuildOptions(blossommigrate.Options{From: from, To: to, DryRun: dryRun, Workers: workers})
	if err != nil {
		return err
	}
	if err := config.LoadConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	source, err := blobstore.NewLocalStore("files")
	if err != nil {
		return fmt.Errorf("initialize local Blossom store: %w", err)
	}
	target, err := blobstore.NewS3Store(ctx, config.Cfg.Store.S3)
	if err != nil {
		return fmt.Errorf("initialize S3 Blossom store: %w", err)
	}

	result, runErr := blossommigrate.Run(ctx, options, source, target)
	_, outputErr := fmt.Fprintf(cmd.OutOrStdout(), "scanned=%d migrated=%d skipped=%d would_migrate=%d dry_run=%t\n", result.Scanned, result.Migrated, result.Skipped, result.WouldMigrate, options.DryRun)
	if outputErr != nil {
		return outputErr
	}
	return runErr
}
