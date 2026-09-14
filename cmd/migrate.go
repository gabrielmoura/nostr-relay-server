package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gabrielmoura/nostr-relay-server/config"
	storedb "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage embedded PostgreSQL schema migrations",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return withMigrationDatabase(func(dsn string) error {
			return storedb.MigrateUp(context.Background(), dsn)
		})
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Revert the latest migration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}
		return withMigrationDatabase(func(dsn string) error {
			return storedb.MigrateDown(context.Background(), dsn, all)
		})
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the applied and expected migration versions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return withMigrationDatabase(func(dsn string) error {
			status, err := storedb.MigrationStatusFor(context.Background(), dsn)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"applied=%d expected=%d dirty=%t current=%t\n",
				status.Applied,
				status.Expected,
				status.Dirty,
				status.Current(),
			)
			return err
		})
	},
}

var migrateForceCmd = &cobra.Command{
	Use:   "force <version>",
	Short: "Reconcile a verified schema with a migration version",
	Long: "Mark a migration version as clean without executing SQL. Use only after " +
		"verifying that the database schema already matches that version.",
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		version, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			return fmt.Errorf("parse migration version: %w", err)
		}
		if version < -1 {
			return fmt.Errorf("migration version must be at least -1")
		}
		return withMigrationDatabase(func(dsn string) error {
			return storedb.MigrateForce(context.Background(), dsn, int(version))
		})
	},
}

var migrateCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create paired up/down migration files for development",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := cmd.Flags().GetString("dir")
		if err != nil {
			return err
		}
		upPath, downPath, err := storedb.CreateMigration(dir, args[0])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "created %s and %s\n", upPath, downPath)
		return err
	},
}

func withMigrationDatabase(operation func(string) error) error {
	if err := config.LoadConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return operation(config.Cfg.DB.PostgresURI)
}

func init() {
	migrateDownCmd.Flags().Bool("all", false, "Revert all migrations (destructive)")
	migrateCreateCmd.Flags().String("dir", "infra/db/migrations", "Migration directory")
	migrateCmd.AddCommand(migrateUpCmd, migrateDownCmd, migrateStatusCmd, migrateForceCmd, migrateCreateCmd)
	rootCmd.AddCommand(migrateCmd)
}
