package cmd

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/spf13/cobra"
)

var confCmd = &cobra.Command{
	Use:   "conf",
	Short: "Configuration commands",
	Long:  "Commands to manage the configuration file",
}

var confPrintCmd = &cobra.Command{
	Use:   "print",
	Short: "Prints the configuration file",
	Run:   runConfigPrint,
}

var confWriteCmd = &cobra.Command{
	Use:   "write",
	Short: "Write the configuration file",
	Run:   runConfigWrite,
}

func runConfigPrint(cmd *cobra.Command, args []string) {
	config.PrintYamlConfig()
}
func runConfigWrite(cmd *cobra.Command, args []string) {
	config.WriteYamlConfig("conf.yaml")
}

func init() {
	confCmd.AddCommand(confPrintCmd)
	confCmd.AddCommand(confWriteCmd)
	rootCmd.AddCommand(confCmd)
}
