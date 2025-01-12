package cmd

import (
	cron2 "github.com/gabrielmoura/nostr-relay-server/infra/cron"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Run cron jobs",
	Run:   runCron,
}

func init() {
	rootCmd.AddCommand(cronCmd)
}
func runCron(cmd *cobra.Command, args []string) {
	c := cron.New()
	c.AddFunc("0 0 0 * * *", func() {
		cron2.DeleteOldEvent(cron2.GenTime("1m"))
	})
	c.Start()
}
