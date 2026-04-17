package cmd

import (
	"fmt"

	confcmd "github.com/gabrielmoura/nostr-relay-server/cmd/internal/conf"
	"github.com/spf13/cobra"
)

var confCmd = &cobra.Command{
	Use:     "conf",
	Aliases: []string{"config"},
	Short:   "Inspect, validate and generate configuration files",
	Long: "Manage `conf.yaml` workflows for local development and operations.\n\n" +
		"Use this command to print default templates, inspect effective runtime config,\n" +
		"validate configuration quality and generate new configuration files.",
}

var confPrintCmd = &cobra.Command{
	Use:     "print",
	Aliases: []string{"show"},
	Short:   "Print default configuration template",
	Long:    "Print the default configuration template with all supported keys.",
	Example: "  nrserver conf print\n" +
		"  nrserver conf print --format json",
	Run: runConfigPrint,
}

var confEffectiveCmd = &cobra.Command{
	Use:   "effective",
	Short: "Print effective loaded configuration",
	Long: "Load configuration exactly as runtime does and print the effective result.\n\n" +
		"By default, uses the standard config lookup paths. Use --file to validate a specific file.",
	Example: "  nrserver conf effective\n" +
		"  nrserver conf effective --file ./conf.yaml --format json",
	Run: runConfigEffective,
}

var confValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file and cron schedules",
	Long: "Validate required fields and semantic constraints used by runtime:\n" +
		"- required DB URI\n" +
		"- relay information structure\n" +
		"- cron expressions for enabled jobs\n" +
		"- reported_events_fetch relays when enabled",
	Example: "  nrserver conf validate\n" +
		"  nrserver conf validate --file ./conf.yaml",
	Run: runConfigValidate,
}

var confWriteCmd = &cobra.Command{
	Use:   "write",
	Short: "Write default configuration file",
	Long:  "Write the default configuration template to a file.",
	Example: "  nrserver conf write\n" +
		"  nrserver conf write --file /etc/nrs/conf.yaml --force",
	Run: runConfigWrite,
}

func runConfigPrint(cmd *cobra.Command, _ []string) {
	rawFormat, _ := cmd.Flags().GetString("format")
	format, err := confcmd.ParseOutputFormat(rawFormat)
	if err != nil {
		cobra.CheckErr(err)
	}

	cobra.CheckErr(confcmd.PrintDefaults(format))
}

func runConfigEffective(cmd *cobra.Command, _ []string) {
	rawFormat, _ := cmd.Flags().GetString("format")
	filePath, _ := cmd.Flags().GetString("file")

	format, err := confcmd.ParseOutputFormat(rawFormat)
	if err != nil {
		cobra.CheckErr(err)
	}

	cobra.CheckErr(confcmd.PrintEffective(filePath, format))
}

func runConfigValidate(cmd *cobra.Command, _ []string) {
	filePath, _ := cmd.Flags().GetString("file")
	if err := confcmd.ValidateConfig(filePath); err != nil {
		cobra.CheckErr(err)
	}

	if filePath == "" {
		cmd.Println("configuration is valid")
		return
	}

	cmd.Println(fmt.Sprintf("configuration file %q is valid", filePath))
}

func runConfigWrite(cmd *cobra.Command, _ []string) {
	filePath, _ := cmd.Flags().GetString("file")
	force, _ := cmd.Flags().GetBool("force")

	if err := confcmd.WriteDefaults(filePath, force); err != nil {
		cobra.CheckErr(err)
	}

	cmd.Println(fmt.Sprintf("default configuration written to %q", filePath))
}

func init() {
	confPrintCmd.Flags().String("format", "yaml", "Output format: yaml or json")

	confEffectiveCmd.Flags().String("format", "yaml", "Output format: yaml or json")
	confEffectiveCmd.Flags().String("file", "", "Load a specific config file instead of default search paths")

	confValidateCmd.Flags().String("file", "", "Validate a specific config file instead of default search paths")

	confWriteCmd.Flags().String("file", "conf.yaml", "Destination file path")
	confWriteCmd.Flags().Bool("force", false, "Overwrite destination file if it already exists")

	confCmd.AddCommand(confPrintCmd)
	confCmd.AddCommand(confEffectiveCmd)
	confCmd.AddCommand(confValidateCmd)
	confCmd.AddCommand(confWriteCmd)
	rootCmd.AddCommand(confCmd)
}
