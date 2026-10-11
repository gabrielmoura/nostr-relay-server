package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

const defaultHealthcheckURL = "http://127.0.0.1:9090/healthz"

var healthcheckCmd = &cobra.Command{
	Use:          "healthcheck",
	Short:        "Check the local server health endpoint",
	Args:         cobra.NoArgs,
	RunE:         runHealthcheck,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(healthcheckCmd)
}

func runHealthcheck(_ *cobra.Command, _ []string) error {
	client := &http.Client{Timeout: 4 * time.Second}
	response, err := client.Get(defaultHealthcheckURL)
	if err != nil {
		return fmt.Errorf("healthcheck request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("healthcheck returned HTTP %s", response.Status)
	}

	return nil
}
