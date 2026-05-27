package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	croncmd "github.com/gabrielmoura/nostr-relay-server/cmd/internal/cron"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	redisqueue "github.com/gabrielmoura/nostr-relay-server/infra/queue/redis"
	"github.com/gabrielmoura/nostr-relay-server/infra/redis"
	internalblossom "github.com/gabrielmoura/nostr-relay-server/internal/blossom"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/down"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	syncjob "github.com/gabrielmoura/nostr-relay-server/internal/sync"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Run Redis-backed operational job workers",
	Long:  "Run Redis-backed operational job workers without starting the full relay HTTP/WebSocket server.",
	Run:   runWorker,
}

func init() {
	rootCmd.AddCommand(workerCmd)
}

func runWorker(_ *cobra.Command, _ []string) {
	if err := config.LoadConfig(); err != nil {
		cobra.CheckErr(err)
	}

	log.Init()
	metrics.RegisterMetrics()
	metrics.RegisterSecurityMetrics()

	if !config.Cfg.Jobs.Enabled || !config.Cfg.Redis.Queue.Enabled {
		cobra.CheckErr(fmt.Errorf("jobs.enabled and redis.queue.enabled must be true to run worker"))
	}
	if err := redis.Init(&config.Cfg.Redis); err != nil {
		cobra.CheckErr(err)
	}

	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := db.Init(mainCtx); err != nil {
		cobra.CheckErr(err)
	}

	runtime, err := redisqueue.NewRuntime(redis.GetClient(), config.Cfg.Redis.Queue, config.Cfg.Jobs)
	if err != nil {
		cobra.CheckErr(err)
	}
	if err := down.RegisterQueueHandlers(runtime.Registry()); err != nil {
		cobra.CheckErr(err)
	}
	if err := syncjob.RegisterQueueHandlers(runtime.Registry()); err != nil {
		cobra.CheckErr(err)
	}
	if err := internalblossom.RegisterQueueHandlers(runtime.Registry()); err != nil {
		cobra.CheckErr(err)
	}
	if err := croncmd.RegisterQueueHandlers(runtime.Registry()); err != nil {
		cobra.CheckErr(err)
	}
	jobcore.SetDefault(runtime.Service())
	if err := runtime.Start(mainCtx); err != nil {
		cobra.CheckErr(err)
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan
	log.Logger.Info("worker shutdown signal received")
	cancel()
	if client := redis.GetClient(); client != nil {
		if closeErr := client.Close(); closeErr != nil {
			log.Logger.Warn("failed to close redis client", zap.Error(closeErr))
		}
	}
}
