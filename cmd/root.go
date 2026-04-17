/*
Copyright © 2024 Gabriel Moura <gmouradev96@gmail.com>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nrserver",
	Short: "Nostr relay server and operations CLI",
	Long: "nrserver provides runtime and operational commands for the Nostr Relay Server.\n\n" +
		"Use it to start relay services, manage configuration, run maintenance jobs,\n" +
		"seed database schema, and execute import/export/sync workflows.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {}
