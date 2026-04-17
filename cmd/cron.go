package cmd

import (
	"time"

	croncmd "github.com/gabrielmoura/nostr-relay-server/cmd/internal/cron"
	"github.com/spf13/cobra"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Run configured maintenance jobs",
	Long: "Run maintenance jobs configured under `cron.*` in `conf.yaml`.\n\n" +
		"By default this command starts the scheduler and waits for shutdown signals.\n" +
		"Use --run-once for a single execution cycle or --list to inspect jobs.",
	Example: "  nrserver cron\n" +
		"  nrserver cron --list\n" +
		"  nrserver cron --run-once\n" +
		"  nrserver cron --run-once --job db_optimization\n" +
		"  nrserver cron --job nip40 --timeout 5m",
	Run: runCron,
}

func init() {
	cronCmd.Flags().Bool("list", false, "List cron jobs and current enable/schedule state")
	cronCmd.Flags().Bool("run-once", false, "Execute selected enabled jobs once and exit")
	cronCmd.Flags().StringSlice("job", nil, "Filter jobs by name (repeatable or comma-separated)")
	cronCmd.Flags().Duration("timeout", 30*time.Minute, "Per-job execution timeout (e.g. 30m, 5m)")
	rootCmd.AddCommand(cronCmd)
}

func runCron(cmd *cobra.Command, _ []string) {
	list, _ := cmd.Flags().GetBool("list")
	runOnce, _ := cmd.Flags().GetBool("run-once")
	jobs, _ := cmd.Flags().GetStringSlice("job")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	options, err := croncmd.BuildOptions(croncmd.Options{
		List:    list,
		RunOnce: runOnce,
		Jobs:    jobs,
		Timeout: timeout,
	})
	if err != nil {
		cobra.CheckErr(err)
	}

	cobra.CheckErr(croncmd.Run(options))
}
